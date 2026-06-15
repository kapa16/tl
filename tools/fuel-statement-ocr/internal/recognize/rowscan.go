package recognize

import (
	"image"
	"sort"

	"tl/fuel-statement-ocr/internal/mask"
	"tl/fuel-statement-ocr/internal/types"
)

// ColumnRange defines horizontal search band for a table column.
type ColumnRange struct {
	ID            string
	X0            float64
	X1            float64
	MaxCells      int
	DecimalPlaces int
}

var columnRanges = map[string][]ColumnRange{
	"perelivnaya": {
		{ID: "quantity_liters", X0: 0.54, X1: 0.78, MaxCells: 5, DecimalPlaces: 0},
		{ID: "quantity_kg", X0: 0.82, X1: 0.92, MaxCells: 5, DecimalPlaces: 0},
	},
	"prihodnaya": {
		{ID: "balance_before", X0: 0.10, X1: 0.28, MaxCells: 5, DecimalPlaces: 0},
		{ID: "quantity_liters", X0: 0.48, X1: 0.72, MaxCells: 5, DecimalPlaces: 0},
		{ID: "quantity_kg", X0: 0.68, X1: 0.92, MaxCells: 5, DecimalPlaces: 0},
	},
	"zapravka": {
		{ID: "garage_number", X0: 0.15, X1: 0.30, MaxCells: 5, DecimalPlaces: 0},
		{ID: "quantity_liters", X0: 0.62, X1: 0.82, MaxCells: 5, DecimalPlaces: 0},
	},
}

var headerRanges = map[string][]ColumnRange{
	"perelivnaya": {
		{ID: "date_day", X0: 0.47, X1: 0.52, MaxCells: 2, DecimalPlaces: 0},
		{ID: "date_month", X0: 0.53, X1: 0.60, MaxCells: 2, DecimalPlaces: 0},
	},
	"prihodnaya": {
		{ID: "date_day", X0: 0.14, X1: 0.20, MaxCells: 2, DecimalPlaces: 0},
		{ID: "date_month", X0: 0.20, X1: 0.28, MaxCells: 2, DecimalPlaces: 0},
	},
	"zapravka": {
		{ID: "date_day", X0: 0.47, X1: 0.52, MaxCells: 2, DecimalPlaces: 0},
		{ID: "date_month", X0: 0.53, X1: 0.60, MaxCells: 2, DecimalPlaces: 0},
	},
}

func RecognizeByRanges(ink *image.Gray, gray *image.Gray, tmpl *mask.Template, templateType string) (map[string]types.Field, []types.Row, float64) {
	templates, refConf := BuildTemplates(ink, gray, tmpl)
	header := map[string]types.Field{}
	for _, col := range headerRanges[templateType] {
		y0 := int(float64(ink.Bounds().Dy()) * 0.095)
		y1 := int(float64(ink.Bounds().Dy()) * 0.135)
		header[col.ID] = recognizeBand(ink, templates, col, y0, y1)
	}
	var rows = make([]types.Row, 0)
	w, h := ink.Bounds().Dx(), ink.Bounds().Dy()
	firstY := tmpl.Table.FirstRowY
	rowH := tmpl.Table.RowHeight
	cols := columnRanges[templateType]
	for row := 1; row <= tmpl.Table.RowCount; row++ {
		y0 := int(float64(h) * (firstY + float64(row-1)*rowH))
		y1 := int(float64(h) * (firstY + float64(row)*rowH))
		rowFields := map[string]types.Field{}
		hasData := false
		for _, col := range cols {
			f := recognizeBand(ink, templates, col, y0, y1)
			rowFields[col.ID] = f
			if f.Status == "ok" || f.Status == "partial" {
				hasData = true
			}
		}
		if hasData {
			rows = append(rows, types.Row{RowIndex: row, Fields: rowFields})
		}
	}
	_ = w
	return header, rows, refConf
}

func recognizeBand(ink *image.Gray, templates DigitTemplates, col ColumnRange, y0, y1 int) types.Field {
	w := ink.Bounds().Dx()
	x0 := int(col.X0 * float64(w))
	x1 := int(col.X1 * float64(w))
	clusters := findClusters(ink, x0, y0, x1, y1)
	if len(clusters) == 0 {
		return types.Field{ID: col.ID, Status: "empty", Confidence: 0}
	}
	var digits []int
	var confs []float64
	var cells []types.Cell
	for i, c := range clusters {
		if i >= col.MaxCells {
			break
		}
		rect := mask.Rect{X: float64(c.x0) / float64(w), Y: float64(c.y0) / float64(ink.Bounds().Dy()),
			W: float64(c.x1-c.x0) / float64(w), H: float64(c.y1-c.y0) / float64(ink.Bounds().Dy())}
		d, conf, st := RecognizeCell(ink, templates, rect)
		cells = append(cells, types.Cell{Index: i, Digit: d, Confidence: conf, Status: st})
		if d != nil && (st == "ok" || st == "partial") {
			digits = append(digits, *d)
			confs = append(confs, conf)
		}
	}
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
	if len(digits) < len(clusters) {
		st = "partial"
	}
	return types.Field{ID: col.ID, Status: st, Confidence: conf, Value: &val, ValueString: vs, Cells: cells}
}

type cluster struct {
	x0, y0, x1, y1 int
}

func findClusters(ink *image.Gray, x0, y0, x1, y1 int) []cluster {
	var xs []int
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x < 0 || y < 0 || x >= ink.Bounds().Dx() || y >= ink.Bounds().Dy() {
				continue
			}
			if ink.GrayAt(x, y).Y > 128 {
				xs = append(xs, x)
			}
		}
	}
	if len(xs) == 0 {
		return nil
	}
	sort.Ints(xs)
	gap := 25
	var groups [][]int
	cur := []int{xs[0]}
	for _, x := range xs[1:] {
		if x-cur[len(cur)-1] > gap {
			groups = append(groups, cur)
			cur = []int{x}
		} else {
			cur = append(cur, x)
		}
	}
	groups = append(groups, cur)
	var out []cluster
	for _, g := range groups {
		if len(g) < 8 {
			continue
		}
		out = append(out, cluster{x0: g[0] - 2, x1: g[len(g)-1] + 2, y0: y0, y1: y1})
	}
	return out
}
