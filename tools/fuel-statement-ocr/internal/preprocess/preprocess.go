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

// InkMask extracts blue handwritten ink as binary mask (255 = ink).
func InkMask(img image.Image) *image.Gray {
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(bl >> 8)
			if b8 > 80 && int(b8) > int(r8)+30 && int(b8) > int(g8)+20 && r8 < 200 {
				out.SetGray(x, y, color.Gray{Y: 255})
			} else {
				out.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}
	return out
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
	// printed reference digits ~ moderate ink density
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
