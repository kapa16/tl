package types

// OCR JSON output contract (version 1).

type Result struct {
	Version          string          `json:"version"`
	TemplateType     string          `json:"templateType"`
	TemplateID       string          `json:"templateId"`
	ImageWidth       int             `json:"imageWidth"`
	ImageHeight      int             `json:"imageHeight"`
	AnchorsFound     bool            `json:"anchorsFound"`
	ProcessingMs     int64           `json:"processingMs"`
	ReferenceDigits  ReferenceDigits `json:"referenceDigits"`
	Header           map[string]Field `json:"header"`
	Footer           map[string]Field `json:"footer"`
	Rows             []Row           `json:"rows"`
	Layout           *LayoutInfo     `json:"layout,omitempty"`
	Warnings         []Warning       `json:"warnings"`
	Errors           []Warning       `json:"errors"`
}

type ReferenceDigits struct {
	Found      bool    `json:"found"`
	Confidence float64 `json:"confidence"`
}

type LayoutInfo struct {
	OrientationApplied   int             `json:"orientationApplied"`
	ExifOrientation      int             `json:"exifOrientation,omitempty"`
	HomographyConfidence float64         `json:"homographyConfidence"`
	TableFound           bool            `json:"tableFound"`
	Columns              []LayoutColumn  `json:"columns,omitempty"`
	RowBands             []LayoutRowBand `json:"rowBands,omitempty"`
	RecognitionDebug     []FieldDebug    `json:"recognitionDebug,omitempty"`
}

type LayoutColumn struct {
	ID     string  `json:"id"`
	X0     float64 `json:"x0"`
	X1     float64 `json:"x1"`
	Source string  `json:"source"`
}

type LayoutRowBand struct {
	RowIndex int     `json:"rowIndex"`
	Y0       float64 `json:"y0"`
	Y1       float64 `json:"y1"`
}

type Field struct {
	ID          string   `json:"id"`
	Label       string   `json:"label,omitempty"`
	Cells       []Cell   `json:"cells,omitempty"`
	Value       *float64 `json:"value,omitempty"`
	ValueString string   `json:"valueString,omitempty"`
	Status      string   `json:"status"`
	Confidence  float64  `json:"confidence"`
}

type Cell struct {
	Index      int     `json:"index"`
	Digit      *int    `json:"digit"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
}

type Row struct {
	RowIndex int              `json:"rowIndex"`
	Fields   map[string]Field `json:"fields"`
}

type Warning struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type FieldDebug struct {
	RowIndex int            `json:"rowIndex"`
	FieldID  string         `json:"fieldId"`
	Selected string         `json:"selected"`
	Attempts []FieldAttempt `json:"attempts"`
}

type FieldAttempt struct {
	Method     string  `json:"method"`
	Value      string  `json:"value,omitempty"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
	Accepted   bool    `json:"accepted"`
	Reason     string  `json:"reason,omitempty"`
	Rect       *DebugRect `json:"rect,omitempty"`
}

type DebugRect struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
}
