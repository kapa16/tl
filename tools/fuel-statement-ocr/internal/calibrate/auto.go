package calibrate

import (
	"image"
	"image/color"

	"tl/fuel-statement-ocr/internal/mask"
)

// AdjustTemplate shifts mask coordinates using detected digit reference strip.
func AdjustTemplate(tmpl *mask.Template, ink *image.Gray) {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	exp := tmpl.Anchors.DigitReferenceStrip
	bestDx, bestDy, bestScore := 0.0, 0.0, -1.0
	for dy := -0.04; dy <= 0.04; dy += 0.005 {
		for dx := -0.08; dx <= 0.08; dx += 0.005 {
			score := stripScore(ink, w, h, exp.X+dx, exp.Y+dy, exp.W, exp.H)
			if score > bestScore {
				bestScore = score
				bestDx, bestDy = dx, dy
			}
		}
	}
	if bestScore < 0.01 {
		return
	}
	shiftAll(tmpl, bestDx, bestDy)
}

func stripScore(ink *image.Gray, w, h int, x, y, rw, rh float64) float64 {
	x0 := int(x * float64(w))
	y0 := int(y * float64(h))
	x1 := int((x + rw) * float64(w))
	y1 := int((y + rh) * float64(h))
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
	var dark int
	var total int
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			total++
			if ink.GrayAt(xx, yy).Y > 0 {
				dark++
			}
		}
	}
	if total == 0 {
		return 0
	}
	r := float64(dark) / float64(total)
	if r < 0.03 || r > 0.5 {
		return r * 0.2
	}
	return r
}

func shiftAll(tmpl *mask.Template, dx, dy float64) {
	ShiftTemplatePublic(tmpl, dx, dy)
}

// ShiftTemplatePublic moves all mask regions by delta.
func ShiftTemplatePublic(tmpl *mask.Template, dx, dy float64) {
	tmpl.Anchors.DigitReferenceStrip.X += dx
	tmpl.Anchors.DigitReferenceStrip.Y += dy
	for i := range tmpl.DigitReference.Cells {
		tmpl.DigitReference.Cells[i].X += dx
		tmpl.DigitReference.Cells[i].Y += dy
	}
	shiftFields(tmpl.Header, dx, dy)
	shiftFields(tmpl.Footer, dx, dy)
	tmpl.Table.FirstRowY += dy
	for i := range tmpl.Table.Columns {
		tmpl.Table.Columns[i].X += dx
	}
}

func shiftFields(m map[string]mask.FieldDef, dx, dy float64) {
	for k, f := range m {
		f.X += dx
		f.Y += dy
		m[k] = f
	}
}

// FindPrintedDigitStrip locates printed reference digits (dark, not only blue).
func FindPrintedDigitStrip(gray *image.Gray) (x, y, w, h float64, ok bool) {
	b := gray.Bounds()
	width, height := b.Dx(), b.Dy()
	bestScore := -1.0
	var bx, by, bw, bh float64
	for yy := int(float64(height) * 0.06); yy < int(float64(height)*0.16); yy += 4 {
		for xx := int(float64(width) * 0.65); xx < int(float64(width)*0.92); xx += 4 {
			rw := int(float64(width) * 0.22)
			rh := int(float64(height) * 0.035)
			score := darkScore(gray, xx, yy, rw, rh)
			if score > bestScore {
				bestScore = score
				bx = float64(xx) / float64(width)
				by = float64(yy) / float64(height)
				bw = float64(rw) / float64(width)
				bh = float64(rh) / float64(height)
			}
		}
	}
	return bx, by, bw, bh, bestScore > 0.08
}

func darkScore(gray *image.Gray, x0, y0, rw, rh int) float64 {
	var dark, total int
	for y := y0; y < y0+rh; y++ {
		for x := x0; x < x0+rw; x++ {
			if x < 0 || y < 0 || x >= gray.Bounds().Dx() || y >= gray.Bounds().Dy() {
				continue
			}
			total++
			if gray.GrayAt(x, y).Y < 128 {
				dark++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(dark) / float64(total)
}

func ToGray(img image.Image) *image.Gray {
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			gray := uint8((299*r + 587*g + 114*bl) / 1000 >> 8)
			out.SetGray(x, y, color.Gray{Y: gray})
		}
	}
	return out
}
