package engine

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"tl/fuel-statement-ocr/internal/calibrate"
	"tl/fuel-statement-ocr/internal/imageio"
	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/preprocess"
	"tl/fuel-statement-ocr/internal/recognize"
	"tl/fuel-statement-ocr/internal/types"
)

type Options struct {
	ImagePath string
	Type      string
	DumpCrops string
}

func Run(opts Options) (*types.Result, error) {
	start := time.Now()
	tmpl, err := mask.LoadByType(opts.Type)
	if err != nil {
		return nil, err
	}
	img, err := imageio.Load(opts.ImagePath)
	if err != nil {
		return nil, err
	}
	strip := preprocess.StripFromTemplate(
		tmpl.Anchors.DigitReferenceStrip.X,
		tmpl.Anchors.DigitReferenceStrip.Y,
		tmpl.Anchors.DigitReferenceStrip.W,
		tmpl.Anchors.DigitReferenceStrip.H,
	)
	img, _ = preprocess.PickOrientation(img, strip)
	w, h := imageio.Bounds(img)
	gray := calibrate.ToGray(img)
	ink := preprocess.InkMask(img)
	if sx, sy, sw, sh, ok := calibrate.FindPrintedDigitStrip(gray); ok {
		dx := sx - tmpl.Anchors.DigitReferenceStrip.X
		dy := sy - tmpl.Anchors.DigitReferenceStrip.Y
		tmpl.Anchors.DigitReferenceStrip.X = sx
		tmpl.Anchors.DigitReferenceStrip.Y = sy
		tmpl.Anchors.DigitReferenceStrip.W = sw
		tmpl.Anchors.DigitReferenceStrip.H = sh
		calibrate.ShiftTemplatePublic(tmpl, dx, dy)
		cells := make([]mask.DigitCell, 10)
		cellW := sw / 10.0
		for i := 0; i < 10; i++ {
			d := i + 1
			if d == 10 {
				d = 0
			}
			cells[i] = mask.DigitCell{
				Digit: d,
				X:     sx + float64(i)*cellW + cellW*0.1,
				Y:     sy + sh*0.1,
				W:     cellW * 0.8,
				H:     sh * 0.8,
			}
		}
		tmpl.DigitReference.Cells = cells
	} else {
		calibrate.AdjustTemplate(tmpl, ink)
	}
	headerFields, rowFields, refConf := recognize.RecognizeByRanges(ink, gray, tmpl, opts.Type)
	res := &types.Result{
		Version:      "1",
		TemplateType: opts.Type,
		TemplateID:   tmpl.ID,
		ImageWidth:   w,
		ImageHeight:  h,
		AnchorsFound: refConf >= 0.25,
		Header:       headerFields,
		Footer:       map[string]types.Field{},
		Rows:         rowFields,
		Warnings:     []types.Warning{},
		Errors:       []types.Warning{},
	}
	res.ReferenceDigits.Found = refConf >= 0.25
	res.ReferenceDigits.Confidence = refConf
	if refConf < recognize.MinReferenceConfidence {
		res.Warnings = append(res.Warnings, types.Warning{
			Code:    "ANCHOR_LOW_CONF",
			Message: fmt.Sprintf("Reference digit strip confidence %.2f", refConf),
		})
	}
	res.ProcessingMs = time.Since(start).Milliseconds()
	return res, nil
}

func dumpRect(ink *image.Gray, r mask.Rect, path string) {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	x0, y0, x1, y1 := r.PixelRect(w, h)
	crop := imageCrop(ink, x0, y0, x1, y1)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.Create(path + ".png")
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, recognize.GrayToRGBA(crop))
}

func imageCrop(src *image.Gray, x0, y0, x1, y1 int) *image.Gray {
	if x1 <= x0 || y1 <= y0 {
		return src
	}
	dst := image.NewGray(image.Rect(0, 0, x1-x0, y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dst.SetGray(x-x0, y-y0, src.GrayAt(x, y))
		}
	}
	return dst
}
