package align

import (
	"image"
	"image/color"
	"math"

	"tl/fuel-statement-ocr/internal/calibrate"
	"tl/fuel-statement-ocr/internal/mask"
)

// Result holds warped image and alignment confidence.
type Result struct {
	Image      image.Image
	Confidence float64
	ScaleX     float64
	ScaleY     float64
	// ContentDX/DY and ContentSX/SY map template coords (full reference canvas)
	// to the letterboxed image: x' = ContentDX + x*ContentSX, y' = ContentDY + y*ContentSY.
	ContentDX float64
	ContentDY float64
	ContentSX float64
	ContentSY float64
}

// WarpToReference uniformly scales the image to reference dimensions (letterbox if needed),
// then shifts so the printed digit strip aligns with the template anchor.
func WarpToReference(img image.Image, tmpl *mask.Template) Result {
	refW := tmpl.ReferenceSize.Width
	refH := tmpl.ReferenceSize.Height
	if refW <= 0 || refH <= 0 {
		refW = 2480
		refH = 3507
	}

	srcW, srcH := bounds(img)
	scale := math.Min(float64(refW)/float64(srcW), float64(refH)/float64(srcH))
	scaledW := int(math.Round(float64(srcW) * scale))
	scaledH := int(math.Round(float64(srcH) * scale))
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}
	scaled := resizeBilinear(img, scaledW, scaledH)
	canvas := pasteCenter(scaled, refW, refH)
	offX := (refW - scaledW) / 2
	offY := (refH - scaledH) / 2
	contentDX := float64(offX) / float64(refW)
	contentDY := float64(offY) / float64(refH)
	contentSX := float64(scaledW) / float64(refW)
	contentSY := float64(scaledH) / float64(refH)

	gray := calibrate.ToGray(canvas)
	confidence := 0.5
	exp := tmpl.Anchors.DigitReferenceStrip
	expX := contentDX + exp.X*contentSX
	expY := contentDY + exp.Y*contentSY
	if _, _, _, _, ok := calibrate.FindPrintedDigitStripNear(gray, expX, expY, exp.W, exp.H); ok {
		confidence = 0.75
	}

	if confidence < 0.3 {
		confidence = 0.3
	}
	return Result{
		Image:      canvas,
		Confidence: confidence,
		ScaleX:     scale,
		ScaleY:     scale,
		ContentDX:  contentDX,
		ContentDY:  contentDY,
		ContentSX:  contentSX,
		ContentSY:  contentSY,
	}
}

func cropCenter(src image.Image, width, height int) image.Image {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= width && sh <= height {
		return pasteCenter(src, width, height)
	}
	cropX := (sw - width) / 2
	cropY := (sh - height) / 2
	if cropX < 0 {
		cropX = 0
	}
	if cropY < 0 {
		cropY = 0
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dst.Set(x, y, src.At(sb.Min.X+cropX+x, sb.Min.Y+cropY+y))
		}
	}
	return dst
}

func pasteCenter(src image.Image, width, height int) image.Image {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	fill := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dst.Set(x, y, fill)
		}
	}
	offX := (width - sw) / 2
	offY := (height - sh) / 2
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			dst.Set(offX+x, offY+y, src.At(sb.Min.X+x, sb.Min.Y+y))
		}
	}
	return dst
}

func shiftImage(img image.Image, dx, dy int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	fill := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, y, fill)
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx, sy := x-dx, y-dy
			if sx >= 0 && sx < w && sy >= 0 && sy < h {
				dst.Set(x, y, img.At(b.Min.X+sx, b.Min.Y+sy))
			}
		}
	}
	return dst
}

func resizeBilinear(src image.Image, width, height int) image.Image {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sy := float64(sb.Min.Y) + (float64(y)+0.5)*float64(sh)/float64(height) - 0.5
		for x := 0; x < width; x++ {
			sx := float64(sb.Min.X) + (float64(x)+0.5)*float64(sw)/float64(width) - 0.5
			dst.Set(x, y, bilinearSample(src, sx, sy))
		}
	}
	return dst
}

func bilinearSample(src image.Image, x, y float64) color.Color {
	b := src.Bounds()
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 >= b.Max.X {
		x1 = b.Max.X - 1
	}
	if y1 >= b.Max.Y {
		y1 = b.Max.Y - 1
	}
	dx := x - float64(x0)
	dy := y - float64(y0)
	c00 := src.At(x0, y0)
	c10 := src.At(x1, y0)
	c01 := src.At(x0, y1)
	c11 := src.At(x1, y1)
	return lerpColor(c00, c10, c01, c11, dx, dy)
}

func lerpColor(c00, c10, c01, c11 color.Color, dx, dy float64) color.Color {
	r00, g00, b00, a00 := c00.RGBA()
	r10, g10, b10, a10 := c10.RGBA()
	r01, g01, b01, a01 := c01.RGBA()
	r11, g11, b11, a11 := c11.RGBA()
	lerp := func(a, b, c, d uint32) uint8 {
		top := float64(a)*(1-dx) + float64(b)*dx
		bot := float64(c)*(1-dx) + float64(d)*dx
		v := top*(1-dy) + bot*dy
		return uint8(v / 256)
	}
	return color.RGBA{
		R: lerp(r00, r10, r01, r11),
		G: lerp(g00, g10, g01, g11),
		B: lerp(b00, b10, b01, b11),
		A: lerp(a00, a10, a01, a11),
	}
}

func bounds(img image.Image) (int, int) {
	b := img.Bounds()
	return b.Dx(), b.Dy()
}

// SyncTemplateAnchors snaps digitReference cells to the detected printed digit strip.
func SyncTemplateAnchors(tmpl *mask.Template, gray *image.Gray, ink *image.Gray) {
	_ = ink
	calibrate.AlignDigitReference(tmpl, gray)
}
