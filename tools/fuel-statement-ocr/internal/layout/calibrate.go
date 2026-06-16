package layout

import (
	"image"

	"tl/fuel-statement-ocr/internal/mask"
)

// CalibrateTableFromInk aligns firstRowY so row index 1 matches the first handwritten data row.
func CalibrateTableFromInk(ink *image.Gray, tmpl *mask.Template) {
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	var x0, x1 float64
	for _, col := range tmpl.Table.Columns {
		if col.ID == "quantity_liters" || col.ID == "quantity_kg" {
			cx0, cx1 := col.ColumnXRange()
			if x1 == 0 {
				x0, x1 = cx0, cx1
			} else {
				if cx0 < x0 {
					x0 = cx0
				}
				if cx1 > x1 {
					x1 = cx1
				}
			}
		}
	}
	if x1 <= x0 {
		return
	}
	ix0 := int(x0 * float64(w))
	ix1 := int(x1 * float64(w))
	baseY := tmpl.Table.FirstRowY
	rowH := tmpl.Table.RowHeight
	bestShift := 0.0
	bestScore := -1
	for shift := -0.20; shift <= 0.20; shift += 0.003 {
		score := 0
		for _, rowOffset := range []float64{1, 2} {
			y0 := int((baseY + shift + rowOffset*rowH) * float64(h))
			y1 := int((baseY + shift + (rowOffset+1)*rowH) * float64(h))
			skip := (y1 - y0) * 38 / 100
			y0 += skip
			score += countInk(ink, ix0, y0, ix1, y1)
		}
		if score > bestScore {
			bestScore = score
			bestShift = shift
		}
	}
	if bestScore > 50 {
		tmpl.Table.FirstRowY = baseY + bestShift
	}
	if tmpl.Table.HeaderBand != nil {
		minY := tmpl.Table.HeaderBand.Y1 + rowH*0.8
		if tmpl.Table.FirstRowY < minY {
			tmpl.Table.FirstRowY = minY
		}
	}
}
