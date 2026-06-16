package orient

import (
	"image"

	"tl/fuel-statement-ocr/internal/mask"
)

// EnsureUpright is a no-op: upside-down detection is handled inside Normalize (0/180 scoring).
func EnsureUpright(img image.Image, tmpl *mask.Template) (image.Image, int) {
	_ = img
	_ = tmpl
	return img, 0
}
