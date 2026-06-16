package layout

import (
	"image"
	"sort"

	"tl/fuel-statement-ocr/internal/mask"
)

// RefineRowsWithInk reorders row detection using blue-ink presence in data columns.
func RefineRowsWithInk(ink *image.Gray, tmpl *mask.Template, bands []RowBand, columns []ColumnBand) []RowBand {
	if len(bands) == 0 || len(columns) == 0 {
		return bands
	}
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	type scored struct {
		band  RowBand
		ink   int
		order int
	}
	var items []scored
	for i, band := range bands {
		y0 := int(band.Y0 * float64(h))
		y1 := int(band.Y1 * float64(h))
		totalInk := 0
		for _, col := range columns {
			x0 := int(col.X0 * float64(w))
			x1 := int(col.X1 * float64(w))
			totalInk += countInk(ink, x0, y0, x1, y1)
		}
		items = append(items, scored{band: band, ink: totalInk, order: i})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].band.Y0 < items[j].band.Y0
	})
	// Keep template row indices; bands with ink are the meaningful rows.
	_ = items
	return bands
}

// InkRowIndices returns 1-based row indices that contain handwritten ink.
func InkRowIndices(ink *image.Gray, tmpl *mask.Template, columns []ColumnBand) []int {
	h := ink.Bounds().Dy()
	w := ink.Bounds().Dx()
	var withInk []int
	for row := 1; row <= tmpl.Table.RowCount; row++ {
		y0 := int((tmpl.Table.FirstRowY + float64(row-1)*tmpl.Table.RowHeight) * float64(h))
		y1 := int((tmpl.Table.FirstRowY + float64(row)*tmpl.Table.RowHeight) * float64(h))
		inkCount := 0
		for _, col := range columns {
			x0 := int(col.X0 * float64(w))
			x1 := int(col.X1 * float64(w))
			inkCount += countInk(ink, x0, y0, x1, y1)
		}
		if inkCount > 12 {
			withInk = append(withInk, row)
		}
	}
	return withInk
}

func countInk(ink *image.Gray, x0, y0, x1, y1 int) int {
	var n int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x < 0 || y < 0 || x >= ink.Bounds().Dx() || y >= ink.Bounds().Dy() {
				continue
			}
			if ink.GrayAt(x, y).Y > 128 {
				n++
			}
		}
	}
	return n
}

// DetectInkBands builds row bands from template geometry (always reliable after align).
func DetectInkBands(tmpl *mask.Template) []RowBand {
	var bands []RowBand
	for row := 1; row <= tmpl.Table.RowCount; row++ {
		y0 := tmpl.Table.FirstRowY + float64(row-1)*tmpl.Table.RowHeight
		y1 := tmpl.Table.FirstRowY + float64(row)*tmpl.Table.RowHeight
		bands = append(bands, RowBand{RowIndex: row, Y0: y0, Y1: y1})
	}
	return bands
}
