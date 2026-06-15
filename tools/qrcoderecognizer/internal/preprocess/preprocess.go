package preprocess

import (
	"image"
	"image/color"
	"math"
)

// EnhanceContrast усиливает контраст относительно середины шкалы (0.5).
func EnhanceContrast(img image.Image, factor float64) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			rf := enhanceChannel(float64(r>>8)/255.0, factor)
			gf := enhanceChannel(float64(g>>8)/255.0, factor)
			bf := enhanceChannel(float64(b>>8)/255.0, factor)
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(rf * 255)),
				G: uint8(math.Round(gf * 255)),
				B: uint8(math.Round(bf * 255)),
				A: uint8(a >> 8),
			})
		}
	}
	return out
}

func enhanceChannel(v, factor float64) float64 {
	v = (v-0.5)*factor + 0.5
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Threshold бинаризует изображение по порогу яркости.
func Threshold(img image.Image, threshold uint8) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := luminance(img.At(x, y))
			var v uint8 = 0
			if gray > threshold {
				v = 255
			}
			out.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return out
}

func luminance(c color.Color) uint8 {
	r, g, b, _ := c.RGBA()
	// ITU-R BT.601
	l := (299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000
	return uint8(l)
}

// CornerCrops возвращает фрагменты четырёх углов изображения.
func CornerCrops(img image.Image) []image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	size := int(math.Round(float64(min(w, h)) * 0.38))
	if size < 400 {
		size = 400
	}
	if size > min(w, h) {
		size = min(w, h)
	}

	type cropSpec struct {
		x0, y0 int
	}
	specs := []cropSpec{
		{bounds.Min.X, bounds.Min.Y},
		{bounds.Max.X - size, bounds.Min.Y},
		{bounds.Min.X, bounds.Max.Y - size},
		{bounds.Max.X - size, bounds.Max.Y - size},
	}

	crops := make([]image.Image, 0, len(specs))
	for _, s := range specs {
		rect := image.Rect(s.x0, s.y0, s.x0+size, s.y0+size)
		if sub, ok := img.(interface {
			SubImage(r image.Rectangle) image.Image
		}); ok {
			crops = append(crops, sub.SubImage(rect))
		}
	}
	return crops
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
