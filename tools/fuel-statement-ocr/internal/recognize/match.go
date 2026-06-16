package recognize

import (
	"image"
	"image/color"
	"math"
	"strings"

	"tl/fuel-statement-ocr/internal/calibrate"
	"tl/fuel-statement-ocr/internal/mask"
)

const (
	MinCellConfidence      = 0.22
	MinReferenceConfidence = 0.55
	EmptyCellMaxInk        = 0.012
	PartialCellMinScore    = 0.08
)

type DigitTemplates [10]image.Image

func BuildTemplates(ink *image.Gray, gray *image.Gray, tmpl *mask.Template) (DigitTemplates, float64) {
	embedded, _ := LoadEmbeddedTemplates(tmpl.Type)
	runtime, rtPeaks, rtCov := buildTemplatesFromStrip(gray, tmpl)
	out := embedded
	if rtPeaks >= 8 && rtCov >= 0.55 {
		for d := 0; d <= 9; d++ {
			if runtime[d] != nil {
				out[d] = runtime[d]
			}
		}
	} else if rtPeaks >= 8 {
		for d := 0; d <= 9; d++ {
			if out[d] == nil && runtime[d] != nil {
				out[d] = runtime[d]
			}
		}
	}
	refConf := referenceConfidence(gray, tmpl, out, rtPeaks, rtCov)
	return out, refConf
}

func referenceConfidence(gray *image.Gray, tmpl *mask.Template, templates DigitTemplates, peaks int, coverage float64) float64 {
	if peaks < 8 {
		peaks, coverage = calibrate.StripTemplateCoverage(gray, tmpl.Anchors.DigitReferenceStrip)
	}
	filled := 0
	var inkScores []float64
	for d := 0; d <= 9; d++ {
		if templates[d] == nil {
			continue
		}
		filled++
		if g, ok := templates[d].(*image.Gray); ok {
			inkScores = append(inkScores, inkRatio(g))
		}
	}
	if filled < 7 {
		return coverage * 0.4
	}
	avgInk := average(inkScores)
	conf := coverage*0.55 + float64(filled)/10.0*0.25 + avgInk*0.2
	if peaks >= 9 {
		conf += 0.05
	}
	reg := calibrate.StripDigitRegularity(gray, tmpl.Anchors.DigitReferenceStrip)
	conf += reg * 0.1
	if conf > 1 {
		conf = 1
	}
	return conf
}

func buildTemplatesFromStrip(gray *image.Gray, tmpl *mask.Template) (DigitTemplates, int, float64) {
	var templates DigitTemplates
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	calibrate.AlignDigitReference(tmpl, gray)
	strip := tmpl.Anchors.DigitReferenceStrip
	peaks, coverage := calibrate.StripTemplateCoverage(gray, strip)
	rects := calibrate.StripDigitRects(gray, strip)
	order := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	for i, r := range rects {
		if i >= len(order) {
			break
		}
		ix0, iy0, ix1, iy1 := r.PixelRect(w, h)
		crop := NormalizePrintedDigit(gray, ix0, iy0, ix1, iy1)
		if !templateQuality(crop) && !SegmentTemplateOK(gray, ix0, iy0, ix1, iy1) {
			continue
		}
		templates[order[i]] = crop
	}
	return templates, peaks, coverage
}

func grayImage(img image.Image) *image.Gray {
	if g, ok := img.(*image.Gray); ok {
		return g
	}
	b := img.Bounds()
	g := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g.SetGray(x, y, color.Gray{Y: grayAt(img, x, y)})
		}
	}
	return g
}

// TemplateQualityPublic reports whether a normalized digit crop is usable as a template.
func TemplateQualityPublic(crop *image.Gray) bool {
	return templateQuality(crop)
}

// SegmentTemplateOK is a relaxed quality gate for segment-style printed digits.
func SegmentTemplateOK(gray *image.Gray, x0, y0, x1, y1 int) bool {
	norm := NormalizePrintedDigit(gray, x0, y0, x1, y1)
	ratio := inkRatio(norm)
	if ratio < 0.05 {
		return false
	}
	b := norm.Bounds()
	if b.Dx() < 2 || b.Dy() < 4 {
		return false
	}
	aspect := float64(b.Dx()) / float64(b.Dy())
	if aspect < 0.18 {
		return false
	}
	spread := horizontalInkSpread(norm)
	return spread >= 0.12 || aspect >= 0.35 || ratio >= 0.15
}

func InkRatioPublic(crop image.Image) float64 {
	if g, ok := crop.(*image.Gray); ok {
		return inkRatio(g)
	}
	return 0
}

func templateQuality(crop *image.Gray) bool {
	ratio := inkRatio(crop)
	if ratio < 0.06 {
		return false
	}
	b := crop.Bounds()
	if b.Dy() < 4 || b.Dx() < 3 {
		return false
	}
	// Reject vertical grid-line "sticks": narrow width, tall aspect.
	aspect := float64(b.Dx()) / float64(b.Dy())
	if aspect < 0.28 {
		return false
	}
	spread := horizontalInkSpread(crop)
	if spread < 0.22 {
		return false
	}
	// Sticks have ink in one column but almost no horizontal extent.
	if spread < 0.34 && aspect < 0.40 {
		return false
	}
	if verticalDominance(crop) > 0.72 && spread < 0.50 {
		return false
	}
	return true
}

func verticalDominance(g *image.Gray) float64 {
	b := g.Bounds()
	if b.Dx() == 0 {
		return 0
	}
	tall := 0
	for x := b.Min.X; x < b.Max.X; x++ {
		dark := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if g.GrayAt(x, y).Y > 128 {
				dark++
			}
		}
		if dark*3 > b.Dy()*2 {
			tall++
		}
	}
	return float64(tall) / float64(b.Dx())
}

func horizontalInkSpread(g *image.Gray) float64 {
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

func mergeTemplates(base, override DigitTemplates) DigitTemplates {
	out := base
	for d := 0; d <= 9; d++ {
		if override[d] != nil {
			if out[d] == nil {
				out[d] = override[d]
			}
		}
	}
	return out
}

func invertGray(g *image.Gray) *image.Gray {
	b := g.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.SetGray(x, y, color.Gray{Y: 255 - g.GrayAt(x, y).Y})
		}
	}
	return out
}

func RecognizeCell(ink *image.Gray, gray *image.Gray, templates DigitTemplates, rect mask.Rect) (digit *int, confidence float64, status string) {
	return recognizeCellPrinted(ink, gray, templates, rect)
}

// RecognizeCellHandwritten matches pen ink (blue/black shades) against digit templates.
func RecognizeCellHandwritten(ink *image.Gray, gray *image.Gray, templates DigitTemplates, rect mask.Rect) (digit *int, confidence float64, status string) {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	x0, y0, x1, y1 := rect.PixelRect(w, h)
	padX := (x1 - x0) / 5
	padY := (y1 - y0) / 4
	if padX < 2 {
		padX = 2
	}
	if padY < 2 {
		padY = 2
	}
	x0 -= padX
	y0 -= padY
	x1 += padX
	y1 += padY
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	inkCrop := cropGray(ink, x0, y0, x1, y1)
	grayCrop := cropGray(gray, x0, y0, x1, y1)
	if handwritingRatio(inkCrop, grayCrop) < EmptyCellMaxInk {
		return nil, 0, "empty"
	}
	if cellLooksLikeGridStick(inkCrop, grayCrop) {
		return nil, 0, "empty"
	}
	bestD := -1
	bestScore := -1.0
	for _, norm := range handwritingNorms(inkCrop, grayCrop) {
		d, s := matchDigit(norm, templates)
		if s > bestScore {
			bestD, bestScore = d, s
		}
	}
	if bestD < 0 || bestScore < MinCellConfidence {
		if bestD >= 0 && bestScore > PartialCellMinScore {
			return &bestD, bestScore, "partial"
		}
		return nil, bestScore, "low_confidence"
	}
	return &bestD, bestScore, "ok"
}

func handwritingNorms(inkCrop, grayCrop *image.Gray) []*image.Gray {
	out := []*image.Gray{normalizeInkDigit(inkCrop, grayCrop)}
	if grayCrop != nil {
		adaptive := adaptiveHandwritingBinary(grayCrop)
		seg := normalizeSegmentHandwriting(grayCrop, adaptive)
		out = append(out,
			seg,
			normalizeSize(invertGray(adaptive), digitTplW, digitTplH),
			normalizeSize(invertGray(binarizeOtsu(subtractBackground(grayCrop))), digitTplW, digitTplH),
		)
	}
	return out
}

func normalizeSegmentHandwriting(grayCrop, inkMask *image.Gray) *image.Gray {
	b := grayCrop.Bounds()
	xMin, yMin, xMax, yMax := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			isInk := inkMask.GrayAt(x, y).Y < 128
			if !isInk {
				g := int(grayCrop.GrayAt(x, y).Y)
				mean := localMeanGrayCrop(grayCrop, x, y, 3)
				isInk = mean-g >= 18 && g < 165
			}
			if !isInk {
				continue
			}
			found = true
			if x < xMin {
				xMin = x
			}
			if x > xMax {
				xMax = x
			}
			if y < yMin {
				yMin = y
			}
			if y > yMax {
				yMax = y
			}
		}
	}
	if !found {
		return normalizeSize(invertGray(inkMask), digitTplW, digitTplH)
	}
	pad := 2
	xMin -= pad
	yMin -= pad
	xMax += pad
	yMax += pad
	if xMin < b.Min.X {
		xMin = b.Min.X
	}
	if yMin < b.Min.Y {
		yMin = b.Min.Y
	}
	if xMax >= b.Max.X {
		xMax = b.Max.X - 1
	}
	if yMax >= b.Max.Y {
		yMax = b.Max.Y - 1
	}
	raw := cropGray(grayCrop, xMin, yMin, xMax+1, yMax+1)
	return normalizeSize(invertGray(binarizeOtsu(subtractBackground(raw))), digitTplW, digitTplH)
}

func adaptiveHandwritingBinary(gray *image.Gray) *image.Gray {
	b := gray.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g := int(gray.GrayAt(x, y).Y)
			mean := localMeanGrayCrop(gray, x, y, 4)
			if mean-g >= 22 && g < 160 {
				out.SetGray(x, y, color.Gray{Y: 0})
			} else {
				out.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return out
}

func localMeanGrayCrop(g *image.Gray, cx, cy, radius int) int {
	b := g.Bounds()
	var sum, n int
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
				continue
			}
			sum += int(g.GrayAt(x, y).Y)
			n++
		}
	}
	if n == 0 {
		return 255
	}
	return sum / n
}

func recognizeCellPrinted(ink *image.Gray, gray *image.Gray, templates DigitTemplates, rect mask.Rect) (digit *int, confidence float64, status string) {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	x0, y0, x1, y1 := rect.PixelRect(w, h)
	padX := (x1 - x0) / 5
	padY := (y1 - y0) / 4
	if padX < 2 {
		padX = 2
	}
	if padY < 2 {
		padY = 2
	}
	x0 -= padX
	y0 -= padY
	x1 += padX
	y1 += padY
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	crop := cropGray(ink, x0, y0, x1, y1)
	grayCrop := cropGray(gray, x0, y0, x1, y1)
	ratio := handwritingRatio(crop, grayCrop)
	if ratio < EmptyCellMaxInk {
		return nil, 0, "empty"
	}
	norm := normalizeSize(invertGray(subtractBackground(grayCrop)), digitTplW, digitTplH)
	bestD := -1
	bestScore := -1.0
	for d := 0; d <= 9; d++ {
		if templates[d] == nil {
			continue
		}
		score := correlation(norm, templates[d])
		if score > bestScore {
			bestScore = score
			bestD = d
		}
	}
	if bestD < 0 || bestScore < MinCellConfidence {
		bin := normalizeSize(invertGray(binarizeOtsu(grayCrop)), digitTplW, digitTplH)
		for d := 0; d <= 9; d++ {
			if templates[d] == nil {
				continue
			}
			score := correlation(bin, templates[d])
			if score > bestScore {
				bestScore = score
				bestD = d
			}
		}
	}
	if bestD < 0 || bestScore < MinCellConfidence {
		if bestD >= 0 && bestScore > PartialCellMinScore {
			return &bestD, bestScore, "partial"
		}
		return nil, bestScore, "low_confidence"
	}
	return &bestD, bestScore, "ok"
}

func subtractBackground(crop *image.Gray) *image.Gray {
	b := crop.Bounds()
	var corners []uint8
	for _, p := range []image.Point{
		{b.Min.X, b.Min.Y}, {b.Max.X - 1, b.Min.Y},
		{b.Min.X, b.Max.Y - 1}, {b.Max.X - 1, b.Max.Y - 1},
	} {
		corners = append(corners, crop.GrayAt(p.X, p.Y).Y)
	}
	bg := corners[0]
	for _, c := range corners[1:] {
		if c > bg {
			bg = c
		}
	}
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := int(crop.GrayAt(x, y).Y)
			if v > int(bg) {
				v = int(bg)
			}
			out.SetGray(x, y, color.Gray{Y: uint8(v)})
		}
	}
	return out
}

func RecognizeField(ink *image.Gray, templates DigitTemplates, rects []mask.Rect, decimalPlaces int) (value *float64, valueString string, status string, confidence float64, cells []CellResult) {
	var digits []string
	var confs []float64
	hasPartial := false
	hasEmpty := false
	for i, r := range rects {
		d, c, st := RecognizeCell(ink, nil, templates, r)
		cr := CellResult{Index: i, Confidence: c, Status: st}
		cr.Digit = d
		cells = append(cells, cr)
		if st == "empty" {
			hasEmpty = true
			digits = append(digits, "")
			continue
		}
		if st == "low_confidence" {
			hasPartial = true
			digits = append(digits, "?")
			confs = append(confs, c)
			continue
		}
		digits = append(digits, string(rune('0'+*d)))
		confs = append(confs, c)
	}
	valueString = strings.Join(digits, "")
	valueString = strings.ReplaceAll(valueString, "?", "")
	if valueString == "" {
		return nil, "", "empty", 0, cells
	}
	val := parseNumber(valueString, decimalPlaces)
	confidence = average(confs)
	status = "ok"
	if hasPartial || hasEmpty {
		status = "partial"
	}
	return &val, strings.TrimLeft(valueString, "0"), status, confidence, cells
}

type CellResult struct {
	Index      int
	Digit      *int
	Confidence float64
	Status     string
}

func parseNumber(s string, decimalPlaces int) float64 {
	if decimalPlaces <= 0 {
		var n float64
		for _, ch := range s {
			if ch < '0' || ch > '9' {
				continue
			}
			n = n*10 + float64(ch-'0')
		}
		return n
	}
	if len(s) <= decimalPlaces {
		return 0
	}
	intPart := s[:len(s)-decimalPlaces]
	fracPart := s[len(s)-decimalPlaces:]
	var a, b float64
	for _, ch := range intPart {
		if ch >= '0' && ch <= '9' {
			a = a*10 + float64(ch-'0')
		}
	}
	for _, ch := range fracPart {
		if ch >= '0' && ch <= '9' {
			b = b*10 + float64(ch-'0')
		}
	}
	div := math.Pow(10, float64(decimalPlaces))
	return a + b/div
}

func cropGray(src *image.Gray, x0, y0, x1, y1 int) *image.Gray {
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

func normalizeSize(src *image.Gray, tw, th int) *image.Gray {
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

func handwritingRatio(ink, gray *image.Gray) float64 {
	b := ink.Bounds()
	var dark, total int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			total++
			if ink.GrayAt(x, y).Y > 128 {
				dark++
				continue
			}
			if gray == nil {
				continue
			}
			g := int(gray.GrayAt(x, y).Y)
			mean := localMeanGrayCrop(gray, x, y, 4)
			if mean-g >= 20 && g < 165 {
				dark++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(dark) / float64(total)
}

func inkRatio(g *image.Gray) float64 {
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

func correlation(a, b image.Image) float64 {
	ba := a.Bounds()
	bb := b.Bounds()
	if ba.Dx() != bb.Dx() || ba.Dy() != bb.Dy() {
		return 0
	}
	var sumA, sumB, sumAB, sumA2, sumB2 float64
	n := 0
	for y := ba.Min.Y; y < ba.Max.Y; y++ {
		for x := ba.Min.X; x < ba.Max.X; x++ {
			av := float64(grayAt(a, x, y))
			bv := float64(grayAt(b, x, y))
			sumA += av
			sumB += bv
			sumAB += av * bv
			sumA2 += av * av
			sumB2 += bv * bv
			n++
		}
	}
	if n == 0 {
		return 0
	}
	num := float64(n)*sumAB - sumA*sumB
	den := math.Sqrt((float64(n)*sumA2-sumA*sumA) * (float64(n)*sumB2-sumB*sumB))
	if den == 0 {
		return 0
	}
	return num / den
}

func binarizeOtsu(src *image.Gray) *image.Gray {
	b := src.Bounds()
	var hist [256]int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			hist[int(src.GrayAt(x, y).Y)]++
		}
	}
	total := b.Dx() * b.Dy()
	if total == 0 {
		return src
	}
	var sum int
	for i := 0; i < 256; i++ {
		sum += i * hist[i]
	}
	var sumB, wB, maxVar int
	threshold := 128
	for t := 0; t < 256; t++ {
		wB += hist[t]
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += t * hist[t]
		mB := sumB / wB
		mF := (sum - sumB) / wF
		varBetween := wB * wF * (mB - mF) * (mB - mF)
		if varBetween > maxVar {
			maxVar = varBetween
			threshold = t
		}
	}
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := uint8(255)
			if int(src.GrayAt(x, y).Y) < threshold {
				v = 0
			}
			out.SetGray(x, y, color.Gray{Y: v})
		}
	}
	return out
}

func grayAt(img image.Image, x, y int) uint8 {
	if g, ok := img.(*image.Gray); ok {
		return g.GrayAt(x, y).Y
	}
	r, _, _, _ := img.At(x, y).RGBA()
	return uint8(r >> 8)
}

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var s float64
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

func SaveCropPNG(path string, ink *image.Gray, rect mask.Rect) error {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	x0, y0, x1, y1 := rect.PixelRect(w, h)
	crop := cropGray(ink, x0, y0, x1, y1)
	return savePNG(path, crop)
}

func savePNG(path string, img image.Image) error {
	// minimal: use image package via internal helper in dump only
	_ = path
	_ = img
	return nil
}

func normalizeInkDigit(inkCrop, grayCrop *image.Gray) *image.Gray {
	b := inkCrop.Bounds()
	xMin, yMin, xMax, yMax := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			isInk := inkCrop.GrayAt(x, y).Y > 128
			if !isInk && grayCrop != nil {
				g := int(grayCrop.GrayAt(x, y).Y)
				mean := localMeanGrayCrop(grayCrop, x, y, 3)
				isInk = mean-g >= 20 && g < 165
			}
			if !isInk {
				continue
			}
			found = true
			if x < xMin {
				xMin = x
			}
			if x > xMax {
				xMax = x
			}
			if y < yMin {
				yMin = y
			}
			if y > yMax {
				yMax = y
			}
		}
	}
	if !found {
		return normalizeSize(inkCrop, digitTplW, digitTplH)
	}
	pad := 2
	xMin -= pad
	yMin -= pad
	xMax += pad
	yMax += pad
	if xMin < b.Min.X {
		xMin = b.Min.X
	}
	if yMin < b.Min.Y {
		yMin = b.Min.Y
	}
	if xMax >= b.Max.X {
		xMax = b.Max.X - 1
	}
	if yMax >= b.Max.Y {
		yMax = b.Max.Y - 1
	}
	crop := cropGray(inkCrop, xMin, yMin, xMax+1, yMax+1)
	out := image.NewGray(image.Rect(0, 0, digitTplW, digitTplH))
	cw := crop.Bounds().Dx()
	ch := crop.Bounds().Dy()
	if cw < 1 || ch < 1 {
		return out
	}
	scale := math.Min(float64(digitTplW-4)/float64(cw), float64(digitTplH-4)/float64(ch))
	nw := int(float64(cw) * scale)
	nh := int(float64(ch) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	offX := (digitTplW - nw) / 2
	offY := (digitTplH - nh) / 2
	for y := 0; y < nh; y++ {
		sy := crop.Bounds().Min.Y + y*crop.Bounds().Dy()/nh
		for x := 0; x < nw; x++ {
			sx := crop.Bounds().Min.X + x*crop.Bounds().Dx()/nw
			isInk := inkCrop.GrayAt(sx, sy).Y > 128
			if !isInk && grayCrop != nil {
				g := int(grayCrop.GrayAt(sx, sy).Y)
				mean := localMeanGrayCrop(grayCrop, sx, sy, 3)
				isInk = mean-g >= 20 && g < 165
			}
			if isInk {
				out.SetGray(offX+x, offY+y, color.Gray{Y: 255})
			}
		}
	}
	return out
}

func matchDigit(norm *image.Gray, templates DigitTemplates) (int, float64) {
	sig := segmentSignature(norm)
	bestD := -1
	bestScore := -1.0
	for d := 0; d <= 9; d++ {
		if templates[d] == nil {
			continue
		}
		tg, ok := templates[d].(*image.Gray)
		if !ok {
			continue
		}
		corr := correlation(norm, tg)
		overlap := maskOverlap(norm, tg)
		bands := segmentBandOverlap(norm, tg)
		sigMatch := segmentSignatureMatch(sig, segmentSignature(tg))
		score := corr*0.32 + overlap*0.28 + bands*0.22 + sigMatch*0.18
		score *= digitShapePenalty(d, sig)
		if score > bestScore {
			bestScore = score
			bestD = d
		}
	}
	return bestD, bestScore
}

func segmentSignature(g *image.Gray) [5]bool {
	var sig [5]bool
	if g == nil {
		return sig
	}
	b := g.Bounds()
	for band := 0; band < 5; band++ {
		y0 := b.Min.Y + band*b.Dy()/5
		y1 := b.Min.Y + (band+1)*b.Dy()/5
		sig[band] = bandHasInk(g, y0, y1)
	}
	return sig
}

func segmentSignatureMatch(a, b [5]bool) float64 {
	match, total := 0, 0
	for i := 0; i < 5; i++ {
		if a[i] || b[i] {
			total++
		}
		if a[i] == b[i] {
			match++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(match) / float64(total)
}

func digitShapePenalty(d int, sig [5]bool) float64 {
	verticalStick := !sig[0] && !sig[2] && !sig[4] && sig[1] && !sig[3]
	topBar := sig[0]
	switch d {
	case 1:
		if topBar {
			return 0.55
		}
		if verticalStick {
			return 1.0
		}
	case 7:
		if !topBar {
			return 0.45
		}
		if verticalStick {
			return 0.5
		}
	case 0, 8:
		if !sig[0] || !sig[4] {
			return 0.6
		}
	case 4:
		if !sig[2] {
			return 0.65
		}
	}
	if verticalStick && d != 1 {
		return 0.55
	}
	return 1.0
}

// segmentBandOverlap compares horizontal ink bands (segment-display style).
func segmentBandOverlap(a, b *image.Gray) float64 {
	if a == nil || b == nil {
		return 0
	}
	ba := a.Bounds()
	bb := b.Bounds()
	if ba.Dx() != bb.Dx() || ba.Dy() != bb.Dy() {
		return 0
	}
	const bands = 5
	var inter, union int
	for band := 0; band < bands; band++ {
		y0 := ba.Min.Y + band*ba.Dy()/bands
		y1 := ba.Min.Y + (band+1)*ba.Dy()/bands
		aInk := bandHasInk(a, y0, y1)
		bInk := bandHasInk(b, y0, y1)
		if aInk || bInk {
			union++
		}
		if aInk && bInk {
			inter++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func cellLooksLikeGridStick(inkCrop, grayCrop *image.Gray) bool {
	b := inkCrop.Bounds()
	if b.Dx() < 4 || b.Dy() < 6 {
		return false
	}
	x0 := b.Min.X + b.Dx()*2/10
	x1 := b.Min.X + b.Dx()*8/10
	darkCols := 0
	for x := x0; x < x1; x++ {
		colDark := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			if inkCrop.GrayAt(x, y).Y > 128 {
				colDark++
				continue
			}
			if grayCrop != nil {
				g := int(grayCrop.GrayAt(x, y).Y)
				mean := localMeanGrayCrop(grayCrop, x, y, 3)
				if mean-g >= 18 && g < 165 {
					colDark++
				}
			}
		}
		if colDark*3 > b.Dy()*2 {
			darkCols++
		}
	}
	innerW := x1 - x0
	if innerW <= 0 {
		return false
	}
	spread := float64(darkCols) / float64(innerW)
	return spread < 0.22 && b.Dx()*4 < b.Dy()*3
}

func bandHasInk(g *image.Gray, y0, y1 int) bool {
	b := g.Bounds()
	threshold := (b.Dx() * (y1 - y0)) / 12
	if threshold < 2 {
		threshold = 2
	}
	var bright int
	for y := y0; y < y1; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if g.GrayAt(x, y).Y > 128 {
				bright++
			}
		}
	}
	return bright >= threshold
}

func maskOverlap(a, b *image.Gray) float64 {
	if a == nil || b == nil {
		return 0
	}
	ba := a.Bounds()
	bb := b.Bounds()
	if ba.Dx() != bb.Dx() || ba.Dy() != bb.Dy() {
		return 0
	}
	var inter, union int
	for y := ba.Min.Y; y < ba.Max.Y; y++ {
		for x := ba.Min.X; x < ba.Max.X; x++ {
			av := a.GrayAt(x, y).Y > 128
			bv := b.GrayAt(x, y).Y > 128
			if av || bv {
				union++
			}
			if av && bv {
				inter++
			}
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// NormalizePrintedDigit crops and normalizes a printed digit from gray image.
func NormalizePrintedDigit(gray *image.Gray, x0, y0, x1, y1 int) *image.Gray {
	raw := cropGray(gray, x0, y0, x1, y1)
	return normalizeSize(invertGray(binarizeOtsu(subtractBackground(raw))), digitTplW, digitTplH)
}

func GrayToRGBA(g *image.Gray) *image.RGBA {
	b := g.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := g.GrayAt(x, y).Y
			out.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return out
}
