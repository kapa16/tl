package mask

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed templates/*.json
var embeddedFS embed.FS

// Template describes normalized cell layout for one statement type.
type Template struct {
	ID                   string  `json:"id"`
	Type                 string  `json:"type"`
	EnumName             string  `json:"enumName"`
	ReferenceSize        Size    `json:"referenceSize"`
	CanonicalOrientation string  `json:"canonicalOrientation"`
	Anchors              Anchors `json:"anchors"`
	DigitReference       DigitRef `json:"digitReference"`
	Header               map[string]FieldDef `json:"header"`
	Footer               map[string]FieldDef `json:"footer"`
	Table                TableDef `json:"table"`
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Anchors struct {
	QRTopLeft           Rect `json:"qrTopLeft"`
	DigitReferenceStrip Rect `json:"digitReferenceStrip"`
}

type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type DigitRef struct {
	CellCount int        `json:"cellCount"`
	Cells     []DigitCell `json:"cells"`
}

type DigitCell struct {
	Digit int     `json:"digit"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
}

type FieldDef struct {
	ID            string  `json:"id"`
	Cells         int     `json:"cells"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	CellW         float64 `json:"cellW"`
	CellH         float64 `json:"cellH"`
	Gap           float64 `json:"gap"`
	DecimalPlaces int     `json:"decimalPlaces"`
}

type TableDef struct {
	FirstRowY float64            `json:"firstRowY"`
	RowHeight float64            `json:"rowHeight"`
	RowCount  int                `json:"rowCount"`
	Columns   []TableColumnDef   `json:"columns"`
}

type TableColumnDef struct {
	ID            string  `json:"id"`
	X             float64 `json:"x"`
	Cells         int     `json:"cells"`
	CellW         float64 `json:"cellW"`
	CellH         float64 `json:"cellH"`
	Gap           float64 `json:"gap"`
	DecimalPlaces int     `json:"decimalPlaces"`
}

func LoadByType(templateType string) (*Template, error) {
	name := map[string]string{
		"zapravka":    "templates/zapravka.json",
		"prihodnaya":  "templates/prihodnaya.json",
		"perelivnaya": "templates/perelivnaya.json",
	}[templateType]
	if name == "" {
		return nil, fmt.Errorf("unknown template type: %s", templateType)
	}
	data, err := embeddedFS.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var t Template
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (t *Template) CellRects(field FieldDef) []Rect {
	rects := make([]Rect, field.Cells)
	x := field.X
	for i := 0; i < field.Cells; i++ {
		rects[i] = Rect{X: x, Y: field.Y, W: field.CellW, H: field.CellH}
		x += field.CellW + field.Gap
	}
	return rects
}

func (t *Template) RowFieldRects(col TableColumnDef, rowIndex int) []Rect {
	y := t.Table.FirstRowY + float64(rowIndex-1)*t.Table.RowHeight
	rects := make([]Rect, col.Cells)
	x := col.X
	for i := 0; i < col.Cells; i++ {
		rects[i] = Rect{X: x, Y: y, W: col.CellW, H: col.CellH}
		x += col.CellW + col.Gap
	}
	return rects
}

func (r Rect) PixelRect(width, height int) (x0, y0, x1, y1 int) {
	x0 = int(r.X * float64(width))
	y0 = int(r.Y * float64(height))
	x1 = int((r.X + r.W) * float64(width))
	y1 = int((r.Y + r.H) * float64(height))
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > width {
		x1 = width
	}
	if y1 > height {
		y1 = height
	}
	return
}
