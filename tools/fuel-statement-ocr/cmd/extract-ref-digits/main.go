// One-time tool: extract printed digit templates from a reference scan.
// Usage: go run ./cmd/extract-ref-digits --type=prihodnaya --out=internal/recognize/refdigits/prihodnaya scan.jpg
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"tl/fuel-statement-ocr/internal/align"
	"tl/fuel-statement-ocr/internal/calibrate"
	"tl/fuel-statement-ocr/internal/imageio"
	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/orient"
	"tl/fuel-statement-ocr/internal/recognize"
)

func main() {
	typeName := flag.String("type", "prihodnaya", "template type")
	outDir := flag.String("out", "", "output directory for digit PNGs")
	flag.Parse()
	if *outDir == "" || flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: extract-ref-digits --out=dir --type=prihodnaya scan.jpg")
		os.Exit(2)
	}
	tmpl, err := mask.LoadByType(*typeName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	img, exifOrient, err := imageio.Load(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	orientRes := orient.Normalize(img, tmpl, exifOrient)
	alignRes := align.WarpToReference(orientRes.Image, tmpl)
	calibrate.AdjustTemplateForCanvas(tmpl, alignRes.ContentDX, alignRes.ContentDY, alignRes.ContentSX, alignRes.ContentSY)
	gray := calibrate.ToGray(alignRes.Image)
	if !calibrate.AlignDigitReference(tmpl, gray) {
		fmt.Fprintln(os.Stderr, "reference digit strip not found")
		os.Exit(1)
	}
	rects := calibrate.StripDigitRects(gray, tmpl.Anchors.DigitReferenceStrip)
	if len(rects) < 8 {
		fmt.Fprintln(os.Stderr, "too few digit rects in strip")
		os.Exit(1)
	}
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	order := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i, r := range rects {
		if i >= len(order) {
			break
		}
		d := order[i]
		ix0, iy0, ix1, iy1 := r.PixelRect(w, h)
		norm := recognize.NormalizePrintedDigit(gray, ix0, iy0, ix1, iy1)
		if !recognize.TemplateQualityPublic(norm) && !recognize.SegmentTemplateOK(gray, ix0, iy0, ix1, iy1) {
			fmt.Fprintf(os.Stderr, "skip digit %d: low quality crop\n", d)
			continue
		}
		path := filepath.Join(*outDir, fmt.Sprintf("%d.png", d))
		f, err := os.Create(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := png.Encode(f, norm); err != nil {
			f.Close()
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		f.Close()
		fmt.Println("wrote", path)
	}
}
