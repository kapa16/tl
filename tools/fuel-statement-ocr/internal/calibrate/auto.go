package calibrate

import (
	"image"
	"image/color"
	"math"

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

// AdjustTemplateForCanvas maps template coordinates from reference canvas to letterboxed image.
func AdjustTemplateForCanvas(tmpl *mask.Template, dx, dy, sx, sy float64) {
	if dx == 0 && dy == 0 && sx == 1 && sy == 1 {
		return
	}
	transformRect := func(r *mask.Rect) {
		r.X = dx + r.X*sx
		r.Y = dy + r.Y*sy
		r.W *= sx
		r.H *= sy
	}
	transformRect(&tmpl.Anchors.DigitReferenceStrip)
	transformRect(&tmpl.Anchors.QRTopLeft)
	for i := range tmpl.DigitReference.Cells {
		dc := &tmpl.DigitReference.Cells[i]
		dc.X = dx + dc.X*sx
		dc.Y = dy + dc.Y*sy
		dc.W *= sx
		dc.H *= sy
	}
	scaleField := func(f *mask.FieldDef) {
		f.X = dx + f.X*sx
		f.Y = dy + f.Y*sy
		f.CellW *= sx
		f.CellH *= sy
		f.Gap *= sx
	}
	for k, f := range tmpl.Header {
		scaleField(&f)
		tmpl.Header[k] = f
	}
	for k, f := range tmpl.Footer {
		scaleField(&f)
		tmpl.Footer[k] = f
	}
	tmpl.Table.FirstRowY = dy + tmpl.Table.FirstRowY*sy
	tmpl.Table.RowHeight *= sy
	if tmpl.Table.HeaderBand != nil {
		tmpl.Table.HeaderBand.Y0 = dy + tmpl.Table.HeaderBand.Y0*sy
		tmpl.Table.HeaderBand.Y1 = dy + tmpl.Table.HeaderBand.Y1*sy
	}
	for i := range tmpl.Table.Columns {
		c := &tmpl.Table.Columns[i]
		c.X = dx + c.X*sx
		c.CellW *= sx
		c.CellH *= sy
		c.Gap *= sx
		if c.FallbackX0 > 0 {
			c.FallbackX0 = dx + c.FallbackX0*sx
			c.FallbackX1 = dx + c.FallbackX1*sx
		}
	}
	if tmpl.DocumentTitle != nil {
		transformRect(&tmpl.DocumentTitle.SearchRegion)
	}
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

// AlignDigitReference locates the printed 1234567890 strip and updates its anchor rect.
func AlignDigitReference(tmpl *mask.Template, gray *image.Gray) bool {
	strip, peaks := LocateReferenceStrip(gray, tmpl)
	if peaks >= 8 {
		tmpl.Anchors.DigitReferenceStrip = strip
		UpdateDigitReferenceCells(tmpl, gray)
	}
	return peaks >= 8
}

// FindPrintedDigitStripNear locates printed reference digits near an expected anchor.
func FindPrintedDigitStripNear(gray *image.Gray, expX, expY, expW, expH float64) (x, y, w, h float64, ok bool) {
	b := gray.Bounds()
	width, height := b.Dx(), b.Dy()
	rw := int(float64(width) * expW)
	rh := int(float64(height) * expH)
	if rw < 8 {
		rw = int(float64(width) * 0.22)
	}
	if rh < 4 {
		rh = int(float64(height) * 0.035)
	}
	bestScore := -1.0
	var bx, by, bw, bh float64
	centerX := int(expX * float64(width))
	centerY := int(expY * float64(height))
	xRad := int(float64(width) * 0.06)
	yRad := int(float64(height) * 0.03)
	if expY > 0.08 {
		xRad = int(float64(width) * 0.10)
		yRad = int(float64(height) * 0.06)
	}
	for yy := centerY - yRad; yy <= centerY+yRad; yy += 2 {
		for xx := centerX - xRad; xx <= centerX+xRad; xx += 2 {
			score := scoreStripCandidate(gray, xx, yy, rw, rh, centerX, centerY, width, height)
			if float64(yy)/float64(height) > 0.09 {
				score *= 0.02
			}
			if float64(xx)/float64(width) < 0.58 {
				score *= 0.15
			}
			if score > bestScore {
				bestScore = score
				bx = float64(xx) / float64(width)
				by = float64(yy) / float64(height)
				bw = float64(rw) / float64(width)
				bh = float64(rh) / float64(height)
			}
		}
	}
	// Always evaluate template anchor as fallback candidate.
	anchorScore := scoreStripCandidate(gray, centerX, centerY, rw, rh, centerX, centerY, width, height)
	if anchorScore > bestScore {
		bestScore = anchorScore
		bx, by = expX, expY
		bw = expW
		bh = expH
	}
	return bx, by, bw, bh, bestScore > 0.05
}

func scoreStripCandidate(gray *image.Gray, xx, yy, rw, rh, centerX, centerY, width, height int) float64 {
	score := darkScore(gray, xx, yy, rw, rh)
	dist := math.Abs(float64(xx-centerX)/float64(width)) + math.Abs(float64(yy-centerY)/float64(height))
	score -= dist * 0.25
	strip := mask.Rect{
		X: float64(xx) / float64(width),
		Y: float64(yy) / float64(height),
		W: float64(rw) / float64(width),
		H: float64(rh) / float64(height),
	}
	peaks := len(StripDigitRects(gray, strip))
	if peaks < 8 {
		score *= 0.25
	} else if peaks > 11 {
		score *= 0.5
	} else {
		score += float64(peaks) * 0.02
	}
	return score
}

// FindPrintedDigitStrip locates printed reference digits (dark, not only blue).
func FindPrintedDigitStrip(gray *image.Gray) (x, y, w, h float64, ok bool) {
	b := gray.Bounds()
	width, height := b.Dx(), b.Dy()
	bestScore := -1.0
	var bx, by, bw, bh float64
	for yy := int(float64(height) * 0.02); yy < int(float64(height)*0.085); yy += 3 {
		for xx := int(float64(width) * 0.58); xx < int(float64(width)*0.95); xx += 4 {
			rw := int(float64(width) * 0.22)
			rh := int(float64(height) * 0.035)
			score := darkScore(gray, xx, yy, rw, rh)
			strip := mask.Rect{
				X: float64(xx) / float64(width),
				Y: float64(yy) / float64(height),
				W: float64(rw) / float64(width),
				H: float64(rh) / float64(height),
			}
			peaks := len(StripDigitRects(gray, strip))
			if peaks < 8 {
				score *= 0.2
			} else {
				score += float64(peaks) * 0.03
			}
			if score > bestScore {
				bestScore = score
				bx = strip.X
				by = strip.Y
				bw = strip.W
				bh = strip.H
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
