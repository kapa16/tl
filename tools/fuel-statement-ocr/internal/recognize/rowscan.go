package recognize

import (
	"image"

	"tl/fuel-statement-ocr/internal/layout"
	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/types"
)

// RecognizeByLayout runs digit recognition using template cell geometry.
func RecognizeByLayout(ink *image.Gray, gray *image.Gray, tmpl *mask.Template, table layout.TableLayout) (map[string]types.Field, []types.Row, float64, DigitTemplates) {
	templates, refConf := BuildTemplates(ink, gray, tmpl)
	header := recognizeHeader(ink, gray, templates, tmpl)

	colByID := map[string]mask.TableColumnDef{}
	for _, c := range tmpl.Table.Columns {
		if c.ShouldRecognize() {
			colByID[c.ID] = c
		}
	}

	rows := make([]types.Row, 0)
	for _, band := range table.RowBands {
		rowFields := map[string]types.Field{}
		for _, colDef := range colByID {
			f := recognizeRowField(ink, gray, templates, colDef, band)
			rowFields[colDef.ID] = f
		}
		if !rowHasQuantityData(rowFields) {
			continue
		}
		rows = append(rows, types.Row{RowIndex: band.RowIndex, Fields: rowFields})
	}
	return header, rows, refConf, templates
}

func rowHasQuantityData(fields map[string]types.Field) bool {
	lit, okL := fields["quantity_liters"]
	kg, okK := fields["quantity_kg"]
	litLen := len(lit.ValueString)
	kgLen := len(kg.ValueString)
	if litLen >= 2 && lit.Confidence >= 0.12 {
		return true
	}
	if kgLen >= 2 && kg.Confidence >= 0.12 {
		return true
	}
	if okL && okK && litLen >= 1 && kgLen >= 1 {
		return lit.Confidence+kg.Confidence > 0.22
	}
	if okK && kgLen >= 1 && kg.Confidence >= 0.15 {
		return true
	}
	if okL && litLen >= 1 && lit.Confidence >= 0.15 {
		return true
	}
	return false
}

func recognizeRowField(ink, gray *image.Gray, templates DigitTemplates, col mask.TableColumnDef, band layout.RowBand) types.Field {
	f := recognizeRowFieldFixedGrid(ink, gray, templates, col, band)
	if acceptSegmentedField(f, col) {
		return f
	}
	f = recognizeRowFieldSegmented(ink, gray, templates, col, band)
	if acceptSegmentedField(f, col) {
		return f
	}
	return recognizeRowFieldByInkRuns(ink, gray, templates, col, band)
}

func acceptInkRunsField(f types.Field, col mask.TableColumnDef) bool {
	if f.Status != "ok" && f.Status != "partial" {
		return false
	}
	n := len(f.ValueString)
	if n == 0 {
		return false
	}
	if n > col.Cells {
		return false
	}
	if f.Confidence < 0.16 {
		return false
	}
	if f.Status == "partial" && (n < 2 || f.Confidence < 0.14) {
		return false
	}
	return true
}

func acceptSegmentedField(f types.Field, col mask.TableColumnDef) bool {
	if f.Status != "ok" && f.Status != "partial" {
		return false
	}
	n := len(f.ValueString)
	if n == 0 {
		return false
	}
	if n > col.Cells+1 {
		return false
	}
	if f.Confidence < 0.12 {
		return false
	}
	return true
}

func recognizeRowFieldFixedGrid(ink, gray *image.Gray, templates DigitTemplates, col mask.TableColumnDef, band layout.RowBand) types.Field {
	rects := rowFieldRectsInBand(col, band)
	var cells []types.Cell
	for i, rect := range rects {
		d, conf, st := RecognizeCellHandwritten(ink, gray, templates, rect)
		cells = append(cells, types.Cell{Index: i, Digit: d, Confidence: conf, Status: st})
	}
	digits, confs := extractDigitRun(cells, 0.08)
	if len(digits) == 0 {
		return types.Field{ID: col.ID, Status: "empty", Cells: cells, Confidence: 0}
	}
	vs := ""
	for _, d := range digits {
		vs += string(rune('0' + d))
	}
	val := parseNumber(vs, col.DecimalPlaces)
	conf := average(confs)
	st := "ok"
	if len(digits) < len(rects) {
		st = "partial"
	}
	return types.Field{ID: col.ID, Status: st, Confidence: conf, Value: &val, ValueString: vs, Cells: cells}
}

func rowFieldRectsInBand(col mask.TableColumnDef, band layout.RowBand) []mask.Rect {
	x0, x1 := col.ColumnXRange()
	cols := col.Cells
	gap := col.Gap
	if gap <= 0 {
		gap = 0.003
	}
	cellW := (x1 - x0 - gap*float64(cols-1)) / float64(cols)
	if cellW <= 0 {
		return rowFieldRectsAtY(col, band.Y0)
	}
	rowH := band.Y1 - band.Y0
	cellH := col.CellH
	if rowH > 0 {
		cellH = rowH * 0.48
	}
	y0 := band.Y0 + rowH*0.40
	rects := make([]mask.Rect, cols)
	x := x0
	for i := 0; i < cols; i++ {
		rects[i] = mask.Rect{X: x, Y: y0, W: cellW, H: cellH}
		x += cellW + gap
	}
	return rects
}

func rowFieldRectsAtY(col mask.TableColumnDef, y0 float64) []mask.Rect {
	rects := make([]mask.Rect, col.Cells)
	x := col.X
	for i := 0; i < col.Cells; i++ {
		rects[i] = mask.Rect{X: x, Y: y0, W: col.CellW, H: col.CellH}
		x += col.CellW + col.Gap
	}
	return rects
}

func recognizeHeader(ink *image.Gray, gray *image.Gray, templates DigitTemplates, tmpl *mask.Template) map[string]types.Field {
	header := map[string]types.Field{}
	for id, field := range tmpl.Header {
		var digits []int
		var confs []float64
		var cells []types.Cell
		for i, rect := range tmpl.CellRects(field) {
			d, conf, st := RecognizeCell(ink, gray, templates, rect)
			cells = append(cells, types.Cell{Index: i, Digit: d, Confidence: conf, Status: st})
			if d != nil && (st == "ok" || st == "partial") {
				digits = append(digits, *d)
				confs = append(confs, conf)
			}
		}
		if len(digits) == 0 {
			header[id] = types.Field{ID: id, Status: "empty", Cells: cells}
			continue
		}
		vs := ""
		for _, d := range digits {
			vs += string(rune('0' + d))
		}
		val := parseNumber(vs, field.DecimalPlaces)
		conf := average(confs)
		st := "ok"
		if len(digits) < field.Cells {
			st = "partial"
		}
		header[id] = types.Field{ID: id, Status: st, Confidence: conf, Value: &val, ValueString: vs, Cells: cells}
	}
	return header
}

// extractDigitRun keeps digits from the first to the last recognized cell (trim empty edges).
func extractDigitRun(cells []types.Cell, minConf float64) ([]int, []float64) {
	first, last := -1, -1
	for i, c := range cells {
		if cellRecognized(c, minConf) {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return nil, nil
	}
	var digits []int
	var confs []float64
	for i := first; i <= last; i++ {
		c := cells[i]
		if cellRecognized(c, minConf) {
			digits = append(digits, *c.Digit)
			confs = append(confs, c.Confidence)
		}
	}
	if len(digits) == 0 {
		return nil, nil
	}
	return digits, confs
}

func cellRecognized(c types.Cell, minConf float64) bool {
	return c.Digit != nil && (c.Status == "ok" || (c.Status == "partial" && c.Confidence >= minConf))
}
