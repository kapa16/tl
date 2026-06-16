package layout

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"tl/fuel-statement-ocr/internal/mask"
)

// DumpOverlay saves an image with column and row bands drawn.
func DumpOverlay(img image.Image, table TableLayout, dir string) error {
	if dir == "" {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}
	colColor := color.RGBA{R: 0, G: 180, B: 0, A: 180}
	rowColor := color.RGBA{R: 255, G: 0, B: 0, A: 120}
	for _, band := range table.RowBands {
		y0 := int(band.Y0 * float64(h))
		y1 := int(band.Y1 * float64(h))
		drawHLine(out, 0, w-1, y0, rowColor)
		drawHLine(out, 0, w-1, y1, rowColor)
	}
	for _, col := range table.Columns {
		x0 := int(col.X0 * float64(w))
		x1 := int(col.X1 * float64(w))
		drawVLine(out, x0, 0, h-1, colColor)
		drawVLine(out, x1, 0, h-1, colColor)
	}
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.Create(filepath.Join(dir, "layout_overlay.png"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

func drawHLine(img *image.RGBA, x0, x1, y int, c color.Color) {
	if y < 0 || y >= img.Bounds().Dy() {
		return
	}
	for x := x0; x <= x1; x++ {
		img.Set(x, y, c)
	}
}

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.Color) {
	if x < 0 || x >= img.Bounds().Dx() {
		return
	}
	for y := y0; y <= y1; y++ {
		img.Set(x, y, c)
	}
}

// DumpCrops saves ink mask crops per row/column for debugging.
func DumpCrops(ink *image.Gray, table TableLayout, dir string) error {
	if dir == "" {
		return nil
	}
	_ = os.MkdirAll(dir, 0o755)
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	for _, band := range table.RowBands {
		y0 := int(band.Y0 * float64(h))
		y1 := int(band.Y1 * float64(h))
		for _, col := range table.Columns {
			x0 := int(col.X0 * float64(w))
			x1 := int(col.X1 * float64(w))
			r := mask.Rect{
				X: float64(x0) / float64(w),
				Y: float64(y0) / float64(h),
				W: float64(x1-x0) / float64(w),
				H: float64(y1-y0) / float64(h),
			}
			path := filepath.Join(dir, "row"+itoa(band.RowIndex)+"_"+col.ID)
			if err := saveCrop(ink, r, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func saveCrop(ink *image.Gray, r mask.Rect, path string) error {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	x0, y0, x1, y1 := r.PixelRect(w, h)
	crop := image.NewGray(image.Rect(0, 0, x1-x0, y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			crop.SetGray(x-x0, y-y0, ink.GrayAt(x, y))
		}
	}
	f, err := os.Create(path + ".png")
	if err != nil {
		return err
	}
	defer f.Close()
	rgba := image.NewRGBA(crop.Bounds())
	for y := crop.Bounds().Min.Y; y < crop.Bounds().Max.Y; y++ {
		for x := crop.Bounds().Min.X; x < crop.Bounds().Max.X; x++ {
			v := crop.GrayAt(x, y).Y
			rgba.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return png.Encode(f, rgba)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
