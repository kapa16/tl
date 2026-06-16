package recognize

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

//go:embed refdigits/prihodnaya/*.png
var prihodnayaRefDigits embed.FS

const digitTplW = 24
const digitTplH = 32

// LoadEmbeddedTemplates returns pre-built digit templates for a statement type.
func LoadEmbeddedTemplates(templateType string) (DigitTemplates, float64) {
	var fs embed.FS
	switch templateType {
	case "prihodnaya":
		fs = prihodnayaRefDigits
	default:
		return DigitTemplates{}, 0
	}
	var templates DigitTemplates
	var scores []float64
	for d := 0; d <= 9; d++ {
		data, err := fs.ReadFile(fmt.Sprintf("refdigits/prihodnaya/%d.png", d))
		if err != nil {
			continue
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		g := toGray(img)
		if inkRatio(g) < 0.06 || !templateQuality(g) {
			continue
		}
		templates[d] = normalizeSize(g, digitTplW, digitTplH)
		scores = append(scores, inkRatio(templates[d].(*image.Gray)))
	}
	if len(scores) < 7 {
		return DigitTemplates{}, 0
	}
	return templates, average(scores)
}

func toGray(img image.Image) *image.Gray {
	if g, ok := img.(*image.Gray); ok {
		return g
	}
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			out.SetGray(x, y, color.Gray{Y: uint8(r >> 8)})
		}
	}
	return out
}

// DumpRefTemplates saves active digit templates as PNG files.
func DumpRefTemplates(dir string, templates DigitTemplates) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for d := 0; d <= 9; d++ {
		if templates[d] == nil {
			continue
		}
		path := fmt.Sprintf("%s/%d.png", dir, d)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := png.Encode(f, templates[d]); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}
