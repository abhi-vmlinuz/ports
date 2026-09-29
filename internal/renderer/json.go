package renderer

import (
	"encoding/json"
	"io"

	"ports/internal/model"
)

// RenderJSON writes formatted JSON representation of records to w.
func RenderJSON(w io.Writer, records []model.PortRecord) error {
	if records == nil {
		records = []model.PortRecord{}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

// JSONRenderer provides compatibility for struct-based callers.
type JSONRenderer struct{}

func NewJSONRenderer() *JSONRenderer { return &JSONRenderer{} }

func (jr *JSONRenderer) Render(w io.Writer, records []model.PortRecord) error {
	return RenderJSON(w, records)
}
