package report

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/adeel450/terraform-drift-detector/internal/model"
)

// Text writes a human-readable summary suitable for a terminal.
func Text(w io.Writer, r model.DriftReport) error {
	fmt.Fprintf(w, "Drift scan %s\n", r.ScanID)
	fmt.Fprintf(w, "  provider: %s\n", r.Provider)
	fmt.Fprintf(w, "  source:   %s\n", r.Source)
	fmt.Fprintf(w, "  time:     %s\n", r.Timestamp.Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(w, "  checked:  %d resource(s)\n\n", r.Summary.ResourcesChecked)

	if !r.HasDrift() {
		fmt.Fprintln(w, "No drift detected. Infrastructure matches Terraform state.")
		return nil
	}

	fmt.Fprintf(w, "DRIFT DETECTED: %d finding(s) — %d deleted, %d modified, %d tag change(s)\n\n",
		r.Summary.Total, r.Summary.Deleted, r.Summary.Modified, r.Summary.TagChanges)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "RESOURCE\tKIND\tFIELD\tEXPECTED\tACTUAL")
	for _, it := range r.Items {
		field := it.Field
		if field == "" {
			field = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			it.Key.String(), it.Kind, field, dash(it.Expected), dash(it.Actual))
	}
	return tw.Flush()
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
