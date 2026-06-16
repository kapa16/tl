package recognize

import (
	"image"

	"tl/fuel-statement-ocr/internal/layout"
	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/types"
)

type inkBlob struct {
	x0, y0, x1, y1 int
}

func recognizeRowFieldByInkRuns(ink, gray *image.Gray, templates DigitTemplates, col mask.TableColumnDef, band layout.RowBand) types.Field {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	cx0, cx1 := col.ColumnXRange()
	y0, y1 := bandHandwritingY(band, h)
	x0 := int(cx0 * float64(w))
	x1 := int(cx1 * float64(w))
	rects := inkRunRects(ink, gray, x0, y0, x1, y1, col.Cells)
	if len(rects) == 0 {
		return types.Field{ID: col.ID, Status: "empty"}
	}
	var cells []types.Cell
	var digits []int
	var confs []float64
	for i, rect := range rects {
		d, conf, st := RecognizeCellHandwritten(ink, gray, templates, rect)
		cells = append(cells, types.Cell{Index: i, Digit: d, Confidence: conf, Status: st})
		if cellRecognized(types.Cell{Digit: d, Confidence: conf, Status: st}, 0.08) {
			digits = append(digits, *d)
			confs = append(confs, conf)
		}
	}
	if len(digits) == 0 {
		return types.Field{ID: col.ID, Status: "empty", Cells: cells}
	}
	vs := ""
	for _, d := range digits {
		vs += string(rune('0' + d))
	}
	val := parseNumber(vs, col.DecimalPlaces)
	conf := average(confs)
	st := "ok"
	if len(digits) < len(rects) {
		st = "partial"
	}
	return types.Field{ID: col.ID, Status: st, Confidence: conf, Value: &val, ValueString: vs, Cells: cells}
}

func inkRunRects(ink, gray *image.Gray, x0, y0, x1, y1, maxDigits int) []mask.Rect {
	if x1 <= x0 || y1 <= y0 {
		return nil
	}
	// Exclude rightmost grid line inside column band.
	margin := (x1 - x0) / 25
	if margin < 3 {
		margin = 3
	}
	x1 -= margin
	rowH := y1 - y0
	profile := make([]int, x1-x0)
	for x := x0; x < x1; x++ {
		for y := y0; y < y1; y++ {
			if ink.GrayAt(x, y).Y > 128 {
				profile[x-x0]++
			}
		}
	}
	maxP := 0
	for _, v := range profile {
		if v > maxP {
			maxP = v
		}
	}
	if maxP < 4 {
		return nil
	}
	threshold := maxP / 6
	if threshold < 3 {
		threshold = 3
	}
	type run struct{ s, e int }
	var runs []run
	in := false
	start := 0
	for i, v := range profile {
		if v >= threshold {
			if !in {
				start = i
				in = true
			}
			continue
		}
		if in {
			runs = append(runs, run{start, i})
			in = false
		}
	}
	if in {
		runs = append(runs, run{start, len(profile)})
	}
	if len(runs) == 0 {
		return nil
	}
	if len(runs) > maxDigits {
		// merge smallest gaps between adjacent runs
		for len(runs) > maxDigits && len(runs) > 1 {
			bestGap, bestIdx := len(profile), 0
			for i := 0; i < len(runs)-1; i++ {
				gap := runs[i+1].s - runs[i].e
				if gap < bestGap {
					bestGap = gap
					bestIdx = i
				}
			}
			runs[bestIdx].e = runs[bestIdx+1].e
			runs = append(runs[:bestIdx+1], runs[bestIdx+2:]...)
		}
	}
	nw, nh := float64(ink.Bounds().Dx()), float64(ink.Bounds().Dy())
	rects := make([]mask.Rect, 0, len(runs))
	for _, r := range runs {
		runW := r.e - r.s
		if runW < 8 {
			continue
		}
		// Drop vertical grid-line sticks.
		if runW*4 < rowH {
			continue
		}
		rx0 := x0 + r.s
		rx1 := x0 + r.e
		ry0, ry1 := y0, y1
		// tighten vertical to ink
		ry0, ry1 = inkVerticalBounds(ink, gray, rx0, ry0, rx1, ry1)
		rects = append(rects, mask.Rect{
			X: float64(rx0) / nw,
			Y: float64(ry0) / nh,
			W: float64(rx1-rx0) / nw,
			H: float64(ry1-ry0) / nh,
		})
	}
	return rects
}

func inkVerticalBounds(ink, gray *image.Gray, x0, y0, x1, y1 int) (int, int) {
	minY, maxY := y1, y0
	found := false
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if ink.GrayAt(x, y).Y > 128 || (gray != nil && gray.GrayAt(x, y).Y < 100) {
				found = true
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if !found {
		return y0, y1
	}
	pad := (maxY - minY) / 6
	if pad < 2 {
		pad = 2
	}
	minY -= pad
	maxY += pad
	if minY < y0 {
	 minY = y0
	}
	if maxY > y1 {
		maxY = y1
	}
	return minY, maxY + 1
}

func recognizeRowFieldSegmented(ink, gray *image.Gray, templates DigitTemplates, col mask.TableColumnDef, band layout.RowBand) types.Field {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	cx0, cx1 := col.ColumnXRange()
	y0, y1 := bandHandwritingY(band, h)
	x0 := int(cx0 * float64(w))
	x1 := int(cx1 * float64(w))
	blobs := findInkBlobs(ink, x0, y0, x1, y1, 25, col.Cells+1)
	if len(blobs) == 0 {
		return types.Field{ID: col.ID, Status: "empty"}
	}
	var cells []types.Cell
	var digits []int
	var confs []float64
	nw, nh := float64(w), float64(h)
	for i, b := range blobs {
		r := mask.Rect{
			X: float64(b.x0) / nw,
			Y: float64(b.y0) / nh,
			W: float64(b.x1-b.x0) / nw,
			H: float64(b.y1-b.y0) / nh,
		}
		d, conf, st := RecognizeCellHandwritten(ink, gray, templates, r)
		cells = append(cells, types.Cell{Index: i, Digit: d, Confidence: conf, Status: st})
		if cellRecognized(types.Cell{Digit: d, Confidence: conf, Status: st}, 0.08) {
			digits = append(digits, *d)
			confs = append(confs, conf)
		}
	}
	if len(digits) == 0 {
		return types.Field{ID: col.ID, Status: "empty", Cells: cells, Confidence: 0}
	}
	vs := ""
	for _, d := range digits {
		vs += string(rune('0' + d))
	}
	val := parseNumber(vs, col.DecimalPlaces)
	conf := average(confs)
	st := "ok"
	if len(digits) < len(blobs) {
		st = "partial"
	}
	return types.Field{ID: col.ID, Status: st, Confidence: conf, Value: &val, ValueString: vs, Cells: cells}
}

func findInkBlobs(ink *image.Gray, x0, y0, x1, y1, minPixels, maxBlobs int) []inkBlob {
	b := ink.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	if x1 <= x0 || y1 <= y0 {
		return nil
	}
	visited := make([][]bool, y1-y0)
	for i := range visited {
		visited[i] = make([]bool, x1-x0)
	}
	var blobs []inkBlob
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if visited[y-y0][x-x0] || ink.GrayAt(x, y).Y <= 128 {
				continue
			}
			blob, n := floodBlob(ink, visited, x0, y0, x1, y1, x, y)
			if n < minPixels {
				continue
			}
			bw := blob.x1 - blob.x0
			bh := blob.y1 - blob.y0
			if bh < 6 || bw < 4 {
				continue
			}
			if bw > bh*3 || bh > bw*5 {
				continue
			}
			blobs = append(blobs, blob)
		}
	}
	sortBlobsByX(blobs)
	if maxBlobs > 0 && len(blobs) > maxBlobs {
		blobs = keepLargestBlobs(blobs, maxBlobs)
		sortBlobsByX(blobs)
	}
	return blobs
}

func keepLargestBlobs(blobs []inkBlob, n int) []inkBlob {
	type scored struct {
		b inkBlob
		a int
	}
	list := make([]scored, len(blobs))
	for i, b := range blobs {
		list[i] = scored{b: b, a: (b.x1 - b.x0) * (b.y1 - b.y0)}
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].a > list[i].a {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	out := make([]inkBlob, n)
	for i := 0; i < n; i++ {
		out[i] = list[i].b
	}
	return out
}

func floodBlob(ink *image.Gray, visited [][]bool, x0, y0, x1, y1, sx, sy int) (inkBlob, int) {
	type pt struct{ x, y int }
	stack := []pt{{sx, sy}}
	minX, minY, maxX, maxY := sx, sy, sx, sy
	n := 0
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if p.x < x0 || p.x >= x1 || p.y < y0 || p.y >= y1 {
			continue
		}
		if visited[p.y-y0][p.x-x0] {
			continue
		}
		if ink.GrayAt(p.x, p.y).Y <= 128 {
			continue
		}
		visited[p.y-y0][p.x-x0] = true
		n++
		if p.x < minX {
			minX = p.x
		}
		if p.x > maxX {
			maxX = p.x
		}
		if p.y < minY {
			minY = p.y
		}
		if p.y > maxY {
			maxY = p.y
		}
		stack = append(stack,
			pt{p.x + 1, p.y}, pt{p.x - 1, p.y},
			pt{p.x, p.y + 1}, pt{p.x, p.y - 1},
		)
	}
	return inkBlob{x0: minX, y0: minY, x1: maxX + 1, y1: maxY + 1}, n
}

func sortBlobsByX(blobs []inkBlob) {
	for i := 0; i < len(blobs); i++ {
		for j := i + 1; j < len(blobs); j++ {
			if blobs[j].x0 < blobs[i].x0 {
				blobs[i], blobs[j] = blobs[j], blobs[i]
			}
		}
	}
}

// bandHandwritingY returns the vertical slice where handwritten digits sit (skip printed header in band).
func bandHandwritingY(band layout.RowBand, h int) (y0, y1 int) {
	y0 = int(band.Y0 * float64(h))
	y1 = int(band.Y1 * float64(h))
	rowH := y1 - y0
	if rowH < 8 {
		return y0, y1
	}
	skip := rowH * 38 / 100
	y0 += skip
	if y1-y0 < 6 {
		y0 = y1 - 6
	}
	return y0, y1
}
