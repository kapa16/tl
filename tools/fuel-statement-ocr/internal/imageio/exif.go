package imageio

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/rwcarlsen/goexif/exif"
)

func Load(path string) (image.Image, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 1, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 1, fmt.Errorf("decode image: %w", err)
	}
	orient := readExifOrientation(data)
	if orient > 1 {
		img = applyExifOrientation(img, orient)
	}
	return img, orient, nil
}

func Bounds(img image.Image) (width, height int) {
	b := img.Bounds()
	return b.Dx(), b.Dy()
}

func readExifOrientation(data []byte) int {
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return 1
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	v, err := tag.Int(0)
	if err != nil || v < 1 || v > 8 {
		return 1
	}
	return v
}

// applyExifOrientation rotates/flips per EXIF orientation tag.
func applyExifOrientation(img image.Image, orient int) image.Image {
	switch orient {
	case 2:
		return flipH(img)
	case 3:
		return rotateTimes(img, 2)
	case 4:
		return flipV(img)
	case 5:
		return flipH(rotateTimes(img, 3))
	case 6:
		return rotateTimes(img, 1)
	case 7:
		return flipH(rotateTimes(img, 1))
	case 8:
		return rotateTimes(img, 3)
	default:
		return img
	}
}

func rotateTimes(img image.Image, times int) image.Image {
	out := img
	for i := 0; i < times; i++ {
		out = rotate90CW(out)
	}
	return out
}

func rotate90CW(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(h-1-(y-b.Min.Y), x-b.Min.X, img.At(x, y))
		}
	}
	return dst
}

func flipH(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(w-1-x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func flipV(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, h-1-y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
