package preprocess

import (
	"image"
	"image/color"
)

// Rotate90 rotates image clockwise n times (n=1..3).
func Rotate90(img image.Image, times int) image.Image {
	times = ((times % 4) + 4) % 4
	if times == 0 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var out image.Image
	switch times {
	case 1:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.Set(h-1-(y-b.Min.Y), x-b.Min.X, img.At(x, y))
			}
		}
		out = dst
	case 2:
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.Set(w-1-(x-b.Min.X), h-1-(y-b.Min.Y), img.At(x, y))
			}
		}
		out = dst
	case 3:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				dst.Set(y-b.Min.Y, w-1-(x-b.Min.X), img.At(x, y))
			}
		}
		out = dst
	}
	return out
}

// InkMask extracts handwritten ink as binary mask (255 = ink).
// Supports blue and black/gray pen in various shades via chroma + local contrast.
func InkMask(img image.Image) *image.Gray {
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := uint8(0)
			if IsInkPixel(img, x, y) {
				v = 255
			}
			out.SetGray(x, y, color.Gray{Y: v})
		}
	}
	return out
}

// InkMaskFull combines color-based ink detection with adaptive gray fallback in table data rows.
func InkMaskFull(img image.Image, gray *image.Gray) *image.Gray {
	ink := InkMask(img)
	if gray == nil {
		return ink
	}
	h := gray.Bounds().Dy()
	dataY := int(float64(h) * 0.28)
	adaptive := AdaptiveInkOnGray(gray, dataY)
	return UnionInkMask(ink, adaptive)
}

// isHorizontalGridStroke detects printed table lines (long horizontal runs).
func isHorizontalGridStroke(img image.Image, cx, cy int) bool {
	b := img.Bounds()
	hRun := 0
	for x := cx - 12; x <= cx+12; x++ {
		if x < b.Min.X || x >= b.Max.X {
			continue
		}
		r, g, bl, _ := img.At(x, cy).RGBA()
		gray := uint8((299*r + 587*g + 114*bl) / 1000 >> 8)
		if gray < 120 {
			hRun++
		}
	}
	vRun := 0
	for y := cy - 8; y <= cy+8; y++ {
		if y < b.Min.Y || y >= b.Max.Y {
			continue
		}
		r, g, bl, _ := img.At(cx, y).RGBA()
		gray := uint8((299*r + 587*g + 114*bl) / 1000 >> 8)
		if gray < 120 {
			vRun++
		}
	}
	return hRun >= 18 && hRun > vRun*3
}

func localMeanGray(img image.Image, cx, cy, radius int) int {
	b := img.Bounds()
	var sum, n int
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
				continue
			}
			r, g, bl, _ := img.At(x, y).RGBA()
			sum += int((299*r + 587*g + 114*bl) / 1000 >> 8)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / n
}

// ScoreOrientation estimates how well digit reference strip region has structure.
func ScoreOrientation(img image.Image, stripX, stripY, stripW, stripH float64) float64 {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	x0 := int(stripX * float64(w))
	y0 := int(stripY * float64(h))
	x1 := int((stripX + stripW) * float64(w))
	y1 := int((stripY + stripH) * float64(h))
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
	ink := InkMask(img)
	var dark int
	var total int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			total++
			if ink.GrayAt(x, y).Y > 128 {
				dark++
			}
		}
	}
	if total == 0 {
		return 0
	}
	ratio := float64(dark) / float64(total)
	if ratio < 0.02 || ratio > 0.45 {
		return ratio * 0.3
	}
	return ratio
}

func PickOrientation(img image.Image, strip maskStrip) (image.Image, int) {
	bestScore := -1.0
	bestRot := 0
	var bestImg image.Image = img
	for rot := 0; rot < 4; rot++ {
		candidate := img
		if rot > 0 {
			candidate = Rotate90(img, rot)
		}
		score := ScoreOrientation(candidate, strip.X, strip.Y, strip.W, strip.H)
		if score > bestScore {
			bestScore = score
			bestRot = rot
			bestImg = candidate
		}
	}
	return bestImg, bestRot
}

type maskStrip struct {
	X, Y, W, H float64
}

func StripFromTemplate(stripX, stripY, stripW, stripH float64) maskStrip {
	return maskStrip{X: stripX, Y: stripY, W: stripW, H: stripH}
}
