package preprocess

import (
	"image"
	"image/color"
)

func luma(r, g, b uint8) int {
	return int((299*uint32(r) + 587*uint32(g) + 114*uint32(b)) / 1000)
}

// IsInkPixel reports pen ink (blue or black/gray shades), not paper or printed grid.
func IsInkPixel(img image.Image, x, y int) bool {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return false
	}
	r, g, bl, _ := img.At(x, y).RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(bl>>8)
	gray := uint8(luma(r8, g8, b8))
	localMean := LocalMeanGray(img, x, y, 5)
	tableY := int(float64(b.Dy()) * 0.17)
	dataY := int(float64(b.Dy()) * 0.28)

	if isBlueInk(r8, g8, b8) {
		return true
	}
	// Black/gray pen only in table data area (skip printed headers above).
	if y < dataY {
		return false
	}
	if !isDarkInk(gray, localMean) {
		return false
	}
	if y >= tableY && isHorizontalGridStroke(img, x, y) {
		return false
	}
	return true
}

// isBlueInk matches blue pen ink in various shades (navy, royal, light blue).
func isBlueInk(r, g, b uint8) bool {
	maxRG := r
	if g > maxRG {
		maxRG = g
	}
	if int(b) < int(maxRG)+8 {
		return false
	}
	chroma := int(b) - int(maxRG)
	if chroma < 6 {
		return false
	}
	lum := luma(r, g, b)
	if lum > 218 {
		return false
	}
	// purple/magenta pens: R competes with B
	if int(r) > int(b)-8 && r > g+10 {
		return false
	}
	return true
}

// isDarkInk matches black/gray pen strokes via local contrast (shade-invariant).
func isDarkInk(gray uint8, localMean int) bool {
	g := int(gray)
	if g > 150 {
		return false
	}
	contrast := localMean - g
	if contrast < 16 {
		return false
	}
	if g < 95 && localMean > 125 {
		return true
	}
	return contrast >= 20
}

// LocalMeanGray returns mean luma in a square neighborhood.
func LocalMeanGray(img image.Image, cx, cy, radius int) int {
	return localMeanGray(img, cx, cy, radius)
}

// AdaptiveInkOnGray marks pixels darker than local background (black/gray pen fallback).
func AdaptiveInkOnGray(gray *image.Gray, y0 int) *image.Gray {
	b := gray.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if y < y0 {
				continue
			}
			g := int(gray.GrayAt(x, y).Y)
			mean := localMeanGray(gray, x, y, 4)
			if mean-g >= 28 && g < 145 {
				out.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return out
}

// UnionInkMask merges binary ink masks (255 = ink).
func UnionInkMask(a, b *image.Gray) *image.Gray {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	ba := a.Bounds()
	out := image.NewGray(ba)
	for y := ba.Min.Y; y < ba.Max.Y; y++ {
		for x := ba.Min.X; x < ba.Max.X; x++ {
			v := uint8(0)
			if a.GrayAt(x, y).Y > 128 || b.GrayAt(x, y).Y > 128 {
				v = 255
			}
			out.SetGray(x, y, color.Gray{Y: v})
		}
	}
	return out
}
