// Package report renders a DriftReport in various formats: machine-readable
// JSON, a human-readable text summary for the CLI, and HTML for the dashboard.
package report

import (
	"encoding/json"
	"io"

	"github.com/adeel450/terraform-drift-detector/internal/model"
)

// JSON writes the report as indented JSON.
func JSON(w io.Writer, r model.DriftReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
