package layout

import (
	"image"
	"math"
	"sort"

	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/types"
)

// TableLayout describes detected table geometry in normalized coordinates.
type TableLayout struct {
	TableFound bool
	Columns    []ColumnBand
	RowBands   []RowBand
}

type ColumnBand struct {
	ID            string
	X0            float64
	X1            float64
	MaxCells      int
	DecimalPlaces int
	Source        string
}

type RowBand struct {
	RowIndex int
	Y0       float64
	Y1       float64
}

// Detect finds table columns and row bands on an aligned image.
func Detect(gray *image.Gray, tmpl *mask.Template) TableLayout {
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	layout := TableLayout{TableFound: true}

	headerY0, headerY1 := headerBandY(tmpl)
	vLines := detectVerticalLines(gray, 0, w, int(headerY0*float64(h)), int(0.88*float64(h)))

	layout.Columns = mapColumns(tmpl, gray, w, h, headerY0, headerY1, vLines)
	layout.RowBands = DetectInkBands(tmpl)

	if len(layout.Columns) == 0 || len(layout.RowBands) == 0 {
		layout.TableFound = false
	}
	return layout
}

func headerBandY(tmpl *mask.Template) (float64, float64) {
	if tmpl.Table.HeaderBand != nil {
		return tmpl.Table.HeaderBand.Y0, tmpl.Table.HeaderBand.Y1
	}
	return 0.16, 0.20
}

func mapColumns(tmpl *mask.Template, gray *image.Gray, w, h int, headerY0, headerY1 float64, vLines []int) []ColumnBand {
	_ = gray
	_ = headerY0
	_ = headerY1
	var cols []ColumnBand
	for _, colDef := range tmpl.Table.Columns {
		if !colDef.ShouldRecognize() {
			continue
		}
		x0, x1 := colDef.ColumnXRange()
		source := "fallback"
		if colDef.FallbackX0 <= 0 {
			source = "template"
		}
		if refined, ok := refineByGrid(x0, x1, vLines, w); ok && source == "template" {
			x0, x1 = refined[0], refined[1]
			if source == "fallback" {
				source = "grid"
			}
		}
		cols = append(cols, ColumnBand{
			ID:            colDef.ID,
			X0:            clamp01(x0),
			X1:            clamp01(x1),
			MaxCells:      colDef.Cells,
			DecimalPlaces: colDef.DecimalPlaces,
			Source:        source,
		})
	}
	return cols
}

type textBlob struct {
	x0, x1 int
	center int
	width  int
}

func findPrintedBlobs(gray *image.Gray, x0, x1, y0, y1 int) []textBlob {
	var xs []int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x < 0 || y < 0 || x >= gray.Bounds().Dx() || y >= gray.Bounds().Dy() {
				continue
			}
			if gray.GrayAt(x, y).Y < 150 {
				xs = append(xs, x)
			}
		}
	}
	if len(xs) == 0 {
		return nil
	}
	sort.Ints(xs)
	gap := 20
	var groups [][]int
	cur := []int{xs[0]}
	for _, x := range xs[1:] {
		if x-cur[len(cur)-1] > gap {
			groups = append(groups, cur)
			cur = []int{x}
		} else {
			cur = append(cur, x)
		}
	}
	groups = append(groups, cur)
	var blobs []textBlob
	for _, g := range groups {
		if len(g) < 15 {
			continue
		}
		blobs = append(blobs, textBlob{
			x0: g[0], x1: g[len(g)-1],
			center: (g[0] + g[len(g)-1]) / 2,
			width:  g[len(g)-1] - g[0],
		})
	}
	return blobs
}

func matchHeaderBlob(blobs []textBlob, col mask.TableColumnDef, w int) (centerX int, ok bool) {
	if len(blobs) == 0 {
		return 0, false
	}
	expX := col.X
	if col.FallbackX0 > 0 {
		expX = (col.FallbackX0 + col.FallbackX1) * 0.5
	}
	bestDist := math.MaxFloat64
	best := -1
	for _, b := range blobs {
		cx := float64(b.center) / float64(w)
		d := math.Abs(cx - expX)
		if d < bestDist {
			bestDist = d
			best = b.center
		}
	}
	if best < 0 || bestDist > 0.08 {
		return 0, false
	}
	return best, true
}

func refineByGrid(x0, x1 float64, vLines []int, w int) ([2]float64, bool) {
	if len(vLines) < 2 {
		return [2]float64{}, false
	}
	x0p := int(x0 * float64(w))
	x1p := int(x1 * float64(w))
	left, right := -1, -1
	for _, vl := range vLines {
		if vl <= x0p+10 && vl > left {
			left = vl
		}
		if vl >= x1p-10 && (right < 0 || vl < right) {
			right = vl
		}
	}
	if left >= 0 && right > left && right-left > 20 {
		return [2]float64{float64(left) / float64(w), float64(right) / float64(w)}, true
	}
	return [2]float64{}, false
}

func mapRows(tmpl *mask.Template, h int, hLines []int) []RowBand {
	firstY := tmpl.Table.FirstRowY
	rowH := tmpl.Table.RowHeight
	count := tmpl.Table.RowCount

	if len(hLines) >= count+1 {
		startY := int(firstY * float64(h))
		approxH := int(rowH * float64(h))
		if bands := extractEvenBands(hLines, startY, count, approxH, h); len(bands) == count {
			avgH := averageBandHeight(bands)
			if avgH >= rowH*0.65 {
				return bands
			}
		}
	}

	var bands []RowBand
	for row := 1; row <= count; row++ {
		y0 := firstY + float64(row-1)*rowH
		y1 := firstY + float64(row)*rowH
		bands = append(bands, RowBand{RowIndex: row, Y0: y0, Y1: y1})
	}
	return bands
}

func averageBandHeight(bands []RowBand) float64 {
	if len(bands) == 0 {
		return 0
	}
	var sum float64
	for _, b := range bands {
		sum += b.Y1 - b.Y0
	}
	return sum / float64(len(bands))
}

func extractEvenBands(lines []int, startY, count, approxH, imgH int) []RowBand {
	sort.Ints(lines)
	var near []int
	for _, y := range lines {
		if y >= startY-int(float64(approxH)*0.5) && y <= startY+count*approxH+approxH {
			near = append(near, y)
		}
	}
	if len(near) < count+1 {
		return nil
	}
	bestIdx := 0
	bestScore := math.MaxFloat64
	for i := 0; i+count < len(near); i++ {
		var score float64
		step := float64(near[i+1] - near[i])
		for j := 1; j < count; j++ {
			s := float64(near[i+j+1] - near[i+j])
			score += math.Abs(s - step)
		}
		if score < bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	var bands []RowBand
	for row := 1; row <= count; row++ {
		y0pix := near[bestIdx+row-1]
		y1pix := near[bestIdx+row]
		bands = append(bands, RowBand{
			RowIndex: row,
			Y0:       float64(y0pix) / float64(imgH),
			Y1:       float64(y1pix) / float64(imgH),
		})
	}
	return bands
}

func detectHorizontalLines(gray *image.Gray, y0, y1 int) []int {
	w := gray.Bounds().Dx()
	if y1 > gray.Bounds().Dy() {
		y1 = gray.Bounds().Dy()
	}
	profile := make([]int, y1-y0)
	for y := y0; y < y1; y++ {
		var sum int
		for x := 0; x < w; x += 2 {
			if x > 0 {
				g0 := int(gray.GrayAt(x-1, y).Y)
				g1 := int(gray.GrayAt(x, y).Y)
				if g0-g1 > 25 || g1-g0 > 25 {
					sum++
				}
			}
		}
		profile[y-y0] = sum
	}
	return peakPositions(profile, y0, w/80, 8)
}

func detectVerticalLines(gray *image.Gray, x0, x1, y0, y1 int) []int {
	h := gray.Bounds().Dy()
	if y1 > h {
		y1 = h
	}
	profile := make([]int, x1-x0)
	for x := x0; x < x1; x++ {
		var sum int
		for y := y0; y < y1; y += 2 {
			if y > 0 {
				g0 := int(gray.GrayAt(x, y-1).Y)
				g1 := int(gray.GrayAt(x, y).Y)
				if g0-g1 > 25 || g1-g0 > 25 {
					sum++
				}
			}
		}
		profile[x-x0] = sum
	}
	return peakPositions(profile, x0, h/60, 6)
}

func peakPositions(profile []int, offset, threshold, minDist int) []int {
	type peak struct {
		idx int
		val int
	}
	var peaks []peak
	for i, v := range profile {
		if v < threshold {
			continue
		}
		if i > 0 && i < len(profile)-1 {
			if v < profile[i-1] || v < profile[i+1] {
				continue
			}
		}
		peaks = append(peaks, peak{i, v})
	}
	sort.Slice(peaks, func(i, j int) bool { return peaks[i].val > peaks[j].val })
	var out []int
	for _, p := range peaks {
		px := p.idx + offset
		tooClose := false
		for _, existing := range out {
			if abs(px-existing) < minDist {
				tooClose = true
				break
			}
		}
		if !tooClose {
			out = append(out, px)
		}
	}
	sort.Ints(out)
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ToTypes converts layout to JSON debug structs.
func (l TableLayout) ToTypes() *types.LayoutInfo {
	if l.TableFound == false && len(l.Columns) == 0 {
		return &types.LayoutInfo{TableFound: false}
	}
	info := &types.LayoutInfo{TableFound: l.TableFound}
	for _, c := range l.Columns {
		info.Columns = append(info.Columns, types.LayoutColumn{
			ID: c.ID, X0: c.X0, X1: c.X1, Source: c.Source,
		})
	}
	for _, r := range l.RowBands {
		info.RowBands = append(info.RowBands, types.LayoutRowBand{
			RowIndex: r.RowIndex, Y0: r.Y0, Y1: r.Y1,
		})
	}
	return info
}

// VerifyDocumentTitle checks if title region matches expected template type keywords.
func VerifyDocumentTitle(gray *image.Gray, tmpl *mask.Template) bool {
	if tmpl.DocumentTitle == nil || len(tmpl.DocumentTitle.Keywords) == 0 {
		return true
	}
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	r := tmpl.DocumentTitle.SearchRegion
	x0 := int(r.X * float64(w))
	y0 := int(r.Y * float64(h))
	x1 := int((r.X + r.W) * float64(w))
	y1 := int((r.Y + r.H) * float64(h))
	var dark int
	var total int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			total++
			if gray.GrayAt(x, y).Y < 160 {
				dark++
			}
		}
	}
	if total == 0 {
		return true
	}
	return float64(dark)/float64(total) > 0.03
}
