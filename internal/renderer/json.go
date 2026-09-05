package renderer

import (
	"encoding/json"
	"io"

	"ports/internal/model"
)

// JSONRenderer outputs PortRecord structures as pure JSON.
type JSONRenderer struct{}

// NewJSONRenderer creates a JSONRenderer.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// Render writes formatted JSON to w.
func (jr *JSONRenderer) Render(w io.Writer, records []model.PortRecord) error {
	if records == nil {
		records = []model.PortRecord{}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}
