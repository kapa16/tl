package calibrate

import (
	"image"
	"image/color"
	"math"

	"tl/fuel-statement-ocr/internal/mask"
)

type digitRun struct {
	s, e, weight int
}

// StripDigitRects locates printed digit crops inside the 1234567890 strip via column projection.
// Returns up to 10 rects in order 1,2,...,9,0.
func StripDigitRects(gray *image.Gray, strip mask.Rect) []mask.Rect {
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	x0, y0, x1, y1 := strip.PixelRect(w, h)
	if x1-x0 < 20 || y1-y0 < 4 {
		return nil
	}
	innerY0 := y0 + (y1-y0)/5
	innerY1 := y0 + (y1-y0)*4/5
	rowH := innerY1 - innerY0
	profile := columnDarkProfile(gray, x0, innerY0, x1, innerY1)
	if len(profile) == 0 {
		return nil
	}
	smooth := smoothProfile(profile, 2)
	maxP := maxInt(smooth)
	if maxP < 3 {
		return nil
	}
	threshold := maxP / 4
	if threshold < 2 {
		threshold = 2
	}
	runs := extractRuns(smooth, threshold)
	runs = filterStripRuns(runs, rowH)
	if len(runs) == 0 {
		return nil
	}
	runs = selectDigitRuns(runs, 10)
	if len(runs) < 8 {
		return slotDigitRects(gray, x0, y0, x1, y1, innerY0, innerY1, runs)
	}
	nw, nh := float64(w), float64(h)
	cellH := float64(y1-y0) * 0.75 / nh
	rects := make([]mask.Rect, 0, len(runs))
	for _, r := range runs {
		pad := (r.e - r.s) / 8
		if pad < 1 {
			pad = 1
		}
		sx := r.s + pad
		ex := r.e - pad
		if ex <= sx+2 {
			sx, ex = r.s, r.e
		}
		tx0, _, tx1, _ := tightInkBounds(gray, x0+sx, innerY0, x0+ex, innerY1)
		if tx1-tx0 < 3 {
			tx0, tx1 = x0+sx, x0+ex
		}
		rects = append(rects, mask.Rect{
			X: float64(tx0) / nw,
			Y: float64(innerY0) / nh,
			W: float64(tx1-tx0) / nw,
			H: cellH,
		})
	}
	return rects
}

func slotDigitRects(gray *image.Gray, x0, y0, x1, y1, innerY0, innerY1 int, runs []digitRun) []mask.Rect {
	left, right := runs[0].s, runs[len(runs)-1].e
	if right-left < 30 {
		return nil
	}
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	nw, nh := float64(w), float64(h)
	cellH := float64(y1-y0) * 0.72 / nh
	slotW := (right - left) / 10
	padX := slotW / 6
	if padX < 2 {
		padX = 2
	}
	rects := make([]mask.Rect, 0, 10)
	for i := 0; i < 10; i++ {
		sx := left + i*slotW + padX
		ex := left + (i+1)*slotW - padX
		if ex <= sx+4 {
			continue
		}
		tx0, _, tx1, _ := tightInkBounds(gray, x0+sx, innerY0, x0+ex, innerY1)
		if tx1-tx0 < 3 {
			tx0, tx1 = x0+sx, x0+ex
		}
		rects = append(rects, mask.Rect{
			X: float64(tx0) / nw,
			Y: float64(innerY0) / nh,
			W: float64(tx1-tx0) / nw,
			H: cellH,
		})
	}
	return rects
}

func selectDigitRuns(runs []digitRun, want int) []digitRun {
	if len(runs) <= want {
		return runs
	}
	out := append([]digitRun(nil), runs...)
	for len(out) > want && len(out) > 1 {
		bestGap, bestIdx := len(out)*1000, 0
		for i := 0; i < len(out)-1; i++ {
			gap := out[i+1].s - out[i].e
			if gap < bestGap {
				bestGap = gap
				bestIdx = i
			}
		}
		out[bestIdx].e = out[bestIdx+1].e
		out[bestIdx].weight += out[bestIdx+1].weight
		out = append(out[:bestIdx+1], out[bestIdx+2:]...)
	}
	return out
}

func columnDarkProfile(gray *image.Gray, x0, y0, x1, y1 int) []int {
	profile := make([]int, x1-x0)
	for x := x0; x < x1; x++ {
		for y := y0; y < y1; y++ {
			if gray.GrayAt(x, y).Y < 150 {
				profile[x-x0]++
			}
		}
	}
	return profile
}

func smoothProfile(profile []int, k int) []int {
	out := make([]int, len(profile))
	for i := range profile {
		sum, n := 0, 0
		for j := i - k; j <= i+k; j++ {
			if j < 0 || j >= len(profile) {
				continue
			}
			sum += profile[j]
			n++
		}
		out[i] = sum / n
	}
	return out
}

func maxInt(v []int) int {
	m := 0
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	return m
}

func extractRuns(profile []int, threshold int) []digitRun {
	var runs []digitRun
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
			runs = append(runs, digitRun{start, i, sumRange(profile, start, i)})
			in = false
		}
	}
	if in {
		runs = append(runs, digitRun{start, len(profile), sumRange(profile, start, len(profile))})
	}
	return runs
}

func filterStripRuns(runs []digitRun, rowH int) []digitRun {
	out := runs[:0]
	maxW := rowH * 18 / 10
	if maxW < 8 {
		maxW = 8
	}
	minW := rowH / 6
	if minW < 3 {
		minW = 3
	}
	for _, r := range runs {
		width := r.e - r.s
		if width < minW || width > maxW {
			continue
		}
		if width*5 < rowH {
			continue
		}
		out = append(out, r)
	}
	return out
}

func filterGridRuns(runs []digitRun, rowH int) []digitRun {
	out := runs[:0]
	for _, r := range runs {
		width := r.e - r.s
		if width < 3 {
			continue
		}
		if width*5 < rowH {
			continue
		}
		out = append(out, r)
	}
	return out
}

// tightInkBounds returns pixel bounds of printed ink, skipping vertical grid columns.
func tightInkBounds(gray *image.Gray, x0, y0, x1, y1 int) (int, int, int, int) {
	minX, minY, maxX, maxY := x1, y1, x0, y0
	found := false
	rowH := y1 - y0
	for x := x0; x < x1; x++ {
		dark := 0
		for y := y0; y < y1; y++ {
			if gray.GrayAt(x, y).Y < 150 {
				dark++
			}
		}
		if dark > rowH*70/100 {
			continue
		}
		for y := y0; y < y1; y++ {
			if gray.GrayAt(x, y).Y >= 150 {
				continue
			}
			found = true
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if !found {
		return x0, y0, x1, y1
	}
	pad := 1
	minX -= pad
	minY -= pad
	maxX += pad
	maxY += pad
	if minX < x0 {
		minX = x0
	}
	if minY < y0 {
		minY = y0
	}
	if maxX >= x1 {
		maxX = x1 - 1
	}
	if maxY >= y1 {
		maxY = y1 - 1
	}
	return minX, minY, maxX + 1, maxY + 1
}

func sumRange(v []int, s, e int) int {
	sum := 0
	for i := s; i < e; i++ {
		sum += v[i]
	}
	return sum
}

// StripDigitRegularity reports how evenly spaced digit peaks are in a strip candidate (0..1).
func StripDigitRegularity(gray *image.Gray, strip mask.Rect) float64 {
	return stripDigitRegularity(gray, strip)
}

func stripDigitRegularity(gray *image.Gray, strip mask.Rect) float64 {
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	x0, y0, x1, y1 := strip.PixelRect(w, h)
	if x1-x0 < 20 || y1-y0 < 4 {
		return 0
	}
	innerY0 := y0 + (y1-y0)/5
	innerY1 := y0 + (y1-y0)*4/5
	rowH := innerY1 - innerY0
	profile := columnDarkProfile(gray, x0, innerY0, x1, innerY1)
	smooth := smoothProfile(profile, 2)
	maxP := maxInt(smooth)
	if maxP < 3 {
		return 0
	}
	threshold := maxP / 4
	if threshold < 2 {
		threshold = 2
	}
	runs := filterStripRuns(extractRuns(smooth, threshold), rowH)
	if len(runs) < 8 || len(runs) > 14 {
		return 0
	}
	gapCV := coefficientOfVariationInt(gapSizes(runs))
	widthCV := coefficientOfVariationInt(runWidths(runs))
	if gapCV > 0.55 || widthCV > 0.85 {
		return 0
	}
	return 1.0 - (gapCV+widthCV)*0.5
}

func runWidths(runs []digitRun) []int {
	out := make([]int, len(runs))
	for i, r := range runs {
		out[i] = r.e - r.s
	}
	return out
}

func gapSizes(runs []digitRun) []int {
	if len(runs) < 2 {
		return nil
	}
	out := make([]int, 0, len(runs)-1)
	for i := 0; i < len(runs)-1; i++ {
		out = append(out, runs[i+1].s-runs[i].e)
	}
	return out
}

func coefficientOfVariationInt(vals []int) float64 {
	if len(vals) == 0 {
		return 1
	}
	sum := 0
	for _, v := range vals {
		sum += v
	}
	mean := float64(sum) / float64(len(vals))
	if mean < 1 {
		return 1
	}
	var sq float64
	for _, v := range vals {
		d := float64(v) - mean
		sq += d * d
	}
	return math.Sqrt(sq/float64(len(vals))) / mean
}

// StripPeakCount returns how many digit slots were found in the reference strip.
func StripPeakCount(gray *image.Gray, strip mask.Rect) int {
	return len(StripDigitRects(gray, strip))
}

// StripTemplateCoverage counts digit rects and how many yield usable segment templates (0..1).
func StripTemplateCoverage(gray *image.Gray, strip mask.Rect) (peaks int, coverage float64) {
	rects := StripDigitRects(gray, strip)
	if len(rects) == 0 {
		return 0, 0
	}
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	good := 0
	for _, r := range rects {
		x0, y0, x1, y1 := r.PixelRect(w, h)
		if x1-x0 < 3 || y1-y0 < 3 {
			continue
		}
		if stripDigitTemplateOK(gray, x0, y0, x1, y1) {
			good++
		}
	}
	return len(rects), float64(good) / 10.0
}

func stripDigitTemplateOK(gray *image.Gray, x0, y0, x1, y1 int) bool {
	raw := cropGrayStrip(gray, x0, y0, x1, y1)
	if raw.Bounds().Dx() < 3 || raw.Bounds().Dy() < 3 {
		return false
	}
	norm := normalizeStripDigit(raw)
	ratio := stripInkRatio(norm)
	if ratio < 0.08 {
		return false
	}
	b := norm.Bounds()
	aspect := float64(b.Dx()) / float64(b.Dy())
	if aspect < 0.22 {
		return false
	}
	spread := stripHorizontalSpread(norm)
	if spread < 0.20 && aspect < 0.42 {
		return false
	}
	return spread >= 0.18 || aspect >= 0.45 || ratio >= 0.18
}

func cropGrayStrip(src *image.Gray, x0, y0, x1, y1 int) *image.Gray {
	if x1 <= x0 || y1 <= y0 {
		return image.NewGray(image.Rect(0, 0, 1, 1))
	}
	dst := image.NewGray(image.Rect(0, 0, x1-x0, y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dst.SetGray(x-x0, y-y0, src.GrayAt(x, y))
		}
	}
	return dst
}

func normalizeStripDigit(raw *image.Gray) *image.Gray {
	const tw, th = 24, 32
	bg := stripCornerBG(raw)
	out := image.NewGray(raw.Bounds())
	for y := raw.Bounds().Min.Y; y < raw.Bounds().Max.Y; y++ {
		for x := raw.Bounds().Min.X; x < raw.Bounds().Max.X; x++ {
			v := int(raw.GrayAt(x, y).Y)
			if v > int(bg) {
				v = int(bg)
			}
			pix := uint8(255)
			if v < 150 {
				pix = 0
			}
			out.SetGray(x, y, color.Gray{Y: pix})
		}
	}
	return resizeGrayStrip(out, tw, th)
}

func stripCornerBG(crop *image.Gray) uint8 {
	b := crop.Bounds()
	corners := []image.Point{
		{b.Min.X, b.Min.Y}, {b.Max.X - 1, b.Min.Y},
		{b.Min.X, b.Max.Y - 1}, {b.Max.X - 1, b.Max.Y - 1},
	}
	bg := uint8(255)
	for _, p := range corners {
		if crop.GrayAt(p.X, p.Y).Y > bg {
			bg = crop.GrayAt(p.X, p.Y).Y
		}
	}
	return bg
}

func resizeGrayStrip(src *image.Gray, tw, th int) *image.Gray {
	b := src.Bounds()
	dst := image.NewGray(image.Rect(0, 0, tw, th))
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			sx := b.Min.X + x*b.Dx()/tw
			sy := b.Min.Y + y*b.Dy()/th
			dst.SetGray(x, y, src.GrayAt(sx, sy))
		}
	}
	return dst
}

func stripInkRatio(g *image.Gray) float64 {
	b := g.Bounds()
	var dark, total int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			total++
			if g.GrayAt(x, y).Y > 128 {
				dark++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(dark) / float64(total)
}

func stripHorizontalSpread(g *image.Gray) float64 {
	b := g.Bounds()
	if b.Dx() == 0 {
		return 0
	}
	cols := 0
	for x := b.Min.X; x < b.Max.X; x++ {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if g.GrayAt(x, y).Y > 128 {
				cols++
				break
			}
		}
	}
	return float64(cols) / float64(b.Dx())
}

func stripCandidateScore(gray *image.Gray, strip mask.Rect, hint mask.Rect) float64 {
	peaks, coverage := StripTemplateCoverage(gray, strip)
	if peaks < 8 {
		return -1
	}
	reg := stripDigitRegularity(gray, strip)
	score := coverage*3.0 + reg*1.5 + float64(peaks)*0.03
	dy := math.Abs(strip.Y - hint.Y)
	dx := math.Abs(strip.X - hint.X)
	score -= dy * 12
	score -= dx * 2
	if strip.Y > 0.12 {
		score -= (strip.Y - 0.12) * 20
	}
	if coverage < 0.5 {
		score -= (0.5 - coverage) * 4
	}
	return score
}

// ScanReferenceStrip searches the top band for the printed 1234567890 strip.
func ScanReferenceStrip(gray *image.Gray, hint mask.Rect) (mask.Rect, int, float64) {
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	rw := int(float64(w) * hint.W)
	rh := int(float64(h) * hint.H)
	if rw < 40 {
		rw = int(float64(w) * 0.22)
	}
	if rh < 8 {
		rh = int(float64(h) * 0.04)
	}
	best := hint
	bestPeaks := 0
	bestCov := 0.0
	bestScore := -1.0
	yEnd := int(float64(h) * 0.14)
	if yEnd < 40 {
		yEnd = 40
	}
	for yy := int(float64(h) * 0.008); yy < yEnd; yy += 2 {
		for xx := int(float64(w) * 0.52); xx < int(float64(w)*0.94); xx += 3 {
			strip := mask.Rect{
				X: float64(xx) / float64(w),
				Y: float64(yy) / float64(h),
				W: float64(rw) / float64(w),
				H: float64(rh) / float64(h),
			}
			score := stripCandidateScore(gray, strip, hint)
			if score < 0 {
				continue
			}
			peaks, cov := StripTemplateCoverage(gray, strip)
			if score > bestScore {
				bestScore = score
				best = strip
				bestPeaks = peaks
				bestCov = cov
			}
		}
	}
	return best, bestPeaks, bestCov
}

// LocateReferenceStrip finds the printed 1234567890 strip rect.
func LocateReferenceStrip(gray *image.Gray, tmpl *mask.Template) (mask.Rect, int) {
	exp := tmpl.Anchors.DigitReferenceStrip
	best := exp
	bestPeaks := 0
	bestScore := -1.0
	consider := func(strip mask.Rect) {
		score := stripCandidateScore(gray, strip, exp)
		if score < 0 {
			return
		}
		peaks, cov := StripTemplateCoverage(gray, strip)
		if cov < 0.45 && score < 1.0 {
			return
		}
		if score > bestScore {
			best, bestPeaks, bestScore = strip, peaks, score
		}
	}
	if sx, sy, sw, sh, ok := FindPrintedDigitStripNear(gray, exp.X, exp.Y, exp.W, exp.H); ok {
		consider(mask.Rect{X: sx, Y: sy, W: sw, H: sh})
	}
	if x, y, w, h, ok := FindPrintedDigitStrip(gray); ok {
		consider(mask.Rect{X: x, Y: y, W: w, H: h})
	}
	consider(exp)
	if scanned, peaks, cov := ScanReferenceStrip(gray, exp); peaks >= 8 && cov >= 0.45 {
		scanScore := stripCandidateScore(gray, scanned, exp)
		if scanScore > bestScore {
			best, bestPeaks = scanned, peaks
		}
	}
	if bestPeaks == 0 {
		return exp, StripPeakCount(gray, exp)
	}
	return best, bestPeaks
}

func UpdateDigitReferenceCells(tmpl *mask.Template, gray *image.Gray) {
	rects := StripDigitRects(gray, tmpl.Anchors.DigitReferenceStrip)
	if len(rects) < 8 {
		return
	}
	order := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	cells := make([]mask.DigitCell, 0, len(rects))
	for i, r := range rects {
		if i >= len(order) {
			break
		}
		cells = append(cells, mask.DigitCell{Digit: order[i], X: r.X, Y: r.Y, W: r.W, H: r.H})
	}
	if len(cells) >= 8 {
		tmpl.DigitReference.Cells = cells
	}
}
