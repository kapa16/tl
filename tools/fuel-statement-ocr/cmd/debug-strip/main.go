package main

import (
	"fmt"
	"os"

	"tl/fuel-statement-ocr/internal/align"
	"tl/fuel-statement-ocr/internal/calibrate"
	"tl/fuel-statement-ocr/internal/imageio"
	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/orient"
	"tl/fuel-statement-ocr/internal/recognize"
)

func main() {
	path := "../../scans/ведомости.4.jpg"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	tmpl, _ := mask.LoadByType("prihodnaya")
	img, exif, _ := imageio.Load(path)
	o := orient.Normalize(img, tmpl, exif)
	fmt.Println("rot", o.AppliedRotation)
	a := align.WarpToReference(o.Image, tmpl)
	calibrate.AdjustTemplateForCanvas(tmpl, a.ContentDX, a.ContentDY, a.ContentSX, a.ContentSY)
	gray := calibrate.ToGray(a.Image)
	strip, peaks := calibrate.LocateReferenceStrip(gray, tmpl)
	_, cov := calibrate.StripTemplateCoverage(gray, strip)
	fmt.Println("strip", strip, "peaks", peaks, "cov", cov)
	rects := calibrate.StripDigitRects(gray, strip)
	order := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	for i, r := range rects {
		if i >= len(order) {
			break
		}
		d := order[i]
		x0, y0, x1, y1 := r.PixelRect(w, h)
		norm := recognize.NormalizePrintedDigit(gray, x0, y0, x1, y1)
		fmt.Printf("d%d tq=%v seg=%v ink=%.3f box=%d,%d-%d,%d\n", d,
			recognize.TemplateQualityPublic(norm),
			recognize.SegmentTemplateOK(gray, x0, y0, x1, y1),
			recognize.InkRatioPublic(norm), x0, y0, x1, y1)
	}
	templates, refConf := recognize.BuildTemplates(nil, gray, tmpl)
	fmt.Println("refConf", refConf, "d8", templates[8] != nil)
}
