package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tl/fuel-statement-ocr/internal/align"
	"tl/fuel-statement-ocr/internal/calibrate"
	"tl/fuel-statement-ocr/internal/imageio"
	"tl/fuel-statement-ocr/internal/layout"
	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/orient"
	"tl/fuel-statement-ocr/internal/preprocess"
	"tl/fuel-statement-ocr/internal/recognize"
	"tl/fuel-statement-ocr/internal/types"
)

type Options struct {
	ImagePath  string
	Type       string
	DumpCrops  string
	DumpLayout string
	DumpRef    string
}

func Run(opts Options) (*types.Result, error) {
	start := time.Now()
	tmpl, err := mask.LoadByType(opts.Type)
	if err != nil {
		return nil, err
	}
	img, exifOrient, err := imageio.Load(opts.ImagePath)
	if err != nil {
		return nil, err
	}

	orientRes := orient.Normalize(img, tmpl, exifOrient)
	img = orientRes.Image

	alignRes := align.WarpToReference(img, tmpl)
	img = alignRes.Image
	calibrate.AdjustTemplateForCanvas(tmpl, alignRes.ContentDX, alignRes.ContentDY, alignRes.ContentSX, alignRes.ContentSY)

	w, h := imageio.Bounds(img)
	gray := calibrate.ToGray(img)
	ink := preprocess.InkMaskFull(img, gray)
	layout.CalibrateTableFromInk(ink, tmpl)
	align.SyncTemplateAnchors(tmpl, gray, ink)

	tableLayout := layout.Detect(gray, tmpl)
	headerFields, rowFields, refConf, templates, recogDebug := recognize.RecognizeByLayout(ink, gray, tmpl, tableLayout)

	if opts.DumpRef != "" {
		_ = recognize.DumpRefTemplates(opts.DumpRef, templates)
	}

	layoutInfo := tableLayout.ToTypes()
	if layoutInfo == nil {
		layoutInfo = &types.LayoutInfo{}
	}
	layoutInfo.OrientationApplied = orientRes.AppliedRotation
	layoutInfo.ExifOrientation = exifOrient
	layoutInfo.HomographyConfidence = alignRes.Confidence
	layoutInfo.RecognitionDebug = recogDebug

	warnings := []types.Warning{}
	if orientRes.Score > 0 && orientRes.SecondBestScore > 0 {
		if orientRes.Score-orientRes.SecondBestScore < 0.15 {
			warnings = append(warnings, types.Warning{
				Code:    "ORIENTATION_LOW_CONF",
				Message: fmt.Sprintf("Orientation scores %.2f vs %.2f", orientRes.Score, orientRes.SecondBestScore),
			})
		}
	}
	if refConf < recognize.MinReferenceConfidence {
		warnings = append(warnings, types.Warning{
			Code:    "ANCHOR_LOW_CONF",
			Message: fmt.Sprintf("Reference digit strip confidence %.2f", refConf),
		})
	}
	if !layout.VerifyDocumentTitle(gray, tmpl) {
		warnings = append(warnings, types.Warning{
			Code:    "TYPE_MISMATCH",
			Message: fmt.Sprintf("Document title region does not match template type %s", opts.Type),
		})
	}
	for _, col := range layoutInfo.Columns {
		if col.Source == "fallback" {
			warnings = append(warnings, types.Warning{
				Code:    "COLUMN_FALLBACK",
				Field:   col.ID,
				Message: fmt.Sprintf("Column %s positioned by fallback coordinates", col.ID),
			})
		}
	}

	if opts.DumpCrops != "" {
		_ = layout.DumpCrops(ink, tableLayout, opts.DumpCrops)
		_ = dumpRecognitionDebug(opts.DumpCrops, layoutInfo)
	}
	if opts.DumpLayout != "" {
		_ = layout.DumpOverlay(img, tableLayout, opts.DumpLayout)
	}

	res := &types.Result{
		Version:      "1.1",
		TemplateType: opts.Type,
		TemplateID:   tmpl.ID,
		ImageWidth:   w,
		ImageHeight:  h,
		AnchorsFound: refConf >= 0.25,
		Header:       headerFields,
		Footer:       map[string]types.Field{},
		Rows:         rowFields,
		Layout:       layoutInfo,
		Warnings:     warnings,
		Errors:       []types.Warning{},
	}
	res.ReferenceDigits.Found = refConf >= 0.25
	res.ReferenceDigits.Confidence = refConf
	res.ProcessingMs = time.Since(start).Milliseconds()
	return res, nil
}

func dumpRecognitionDebug(dir string, info *types.LayoutInfo) error {
	if dir == "" || info == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "recognition_debug.json"), data, 0o644)
}
