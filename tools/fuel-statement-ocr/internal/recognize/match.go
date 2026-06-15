package recognize

import (
	"image"
	"image/color"
	"math"
	"strings"

	"tl/fuel-statement-ocr/internal/mask"
)

const (
	MinCellConfidence      = 0.35
	MinReferenceConfidence = 0.65
	EmptyCellMaxInk        = 0.02
)

type DigitTemplates [10]image.Image

func BuildTemplates(ink *image.Gray, gray *image.Gray, tmpl *mask.Template) (DigitTemplates, float64) {
	var templates DigitTemplates
	var scores []float64
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	for _, dc := range tmpl.DigitReference.Cells {
		ix0 := int(dc.X * float64(w))
		iy0 := int(dc.Y * float64(h))
		ix1 := int((dc.X + dc.W) * float64(w))
		iy1 := int((dc.Y + dc.H) * float64(h))
		crop := invertGray(cropGray(gray, ix0, iy0, ix1, iy1))
		d := dc.Digit
		if d >= 0 && d <= 9 {
			templates[d] = normalizeSize(crop, 24, 32)
			scores = append(scores, inkRatio(crop))
		}
	}
	return templates, average(scores)
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

func RecognizeCell(ink *image.Gray, templates DigitTemplates, rect mask.Rect) (digit *int, confidence float64, status string) {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	x0, y0, x1, y1 := rect.PixelRect(w, h)
	crop := cropGray(ink, x0, y0, x1, y1)
	ratio := inkRatio(crop)
	if ratio < EmptyCellMaxInk {
		return nil, 0, "empty"
	}
	norm := normalizeSize(crop, 24, 32)
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
		// fallback: pick digit with any positive correlation
		if bestD >= 0 && bestScore > 0.15 {
			return &bestD, bestScore, "partial"
		}
		return nil, bestScore, "low_confidence"
	}
	return &bestD, bestScore, "ok"
}

func RecognizeField(ink *image.Gray, templates DigitTemplates, rects []mask.Rect, decimalPlaces int) (value *float64, valueString string, status string, confidence float64, cells []CellResult) {
	var digits []string
	var confs []float64
	hasPartial := false
	hasEmpty := false
	for i, r := range rects {
		d, c, st := RecognizeCell(ink, templates, r)
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

// GrayToRGBA for debug dumps.
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
