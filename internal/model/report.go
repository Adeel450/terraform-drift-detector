package model

import "time"

// DriftKind classifies a single drift finding.
type DriftKind string

const (
	// DriftDeleted means a resource present in Terraform state was not found
	// in the cloud (it was deleted or never created out-of-band).
	DriftDeleted DriftKind = "deleted"
	// DriftModified means a comparable attribute differs between state and cloud.
	DriftModified DriftKind = "modified"
	// DriftTag means a tag/label was added, removed, or changed.
	DriftTag DriftKind = "tag"
)

// DriftItem is a single drift finding for one resource.
type DriftItem struct {
	Key  ResourceKey `json:"key"`
	Name string      `json:"name"`
	Kind DriftKind   `json:"kind"`
	// Field is the attribute or tag key that drifted. Empty for DriftDeleted.
	Field    string `json:"field,omitempty"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Message  string `json:"message"`
}

// Summary aggregates counts for a report.
type Summary struct {
	ResourcesChecked int `json:"resources_checked"`
	Deleted          int `json:"deleted"`
	Modified         int `json:"modified"`
	TagChanges       int `json:"tag_changes"`
	Total            int `json:"total"`
}

// DriftReport is the full result of one scan.
type DriftReport struct {
	ScanID    string      `json:"scan_id"`
	Timestamp time.Time   `json:"timestamp"`
	Provider  string      `json:"provider"`
	Source    string      `json:"source"`
	Summary   Summary     `json:"summary"`
	Items     []DriftItem `json:"items"`
}

// HasDrift reports whether any drift was detected.
func (r DriftReport) HasDrift() bool {
	return r.Summary.Total > 0
}

// recount recomputes the summary counters from Items. ResourcesChecked is set
// by the caller (the scanner) since it is not derivable from Items alone.
func (r *DriftReport) recount() {
	var s Summary
	s.ResourcesChecked = r.Summary.ResourcesChecked
	for _, it := range r.Items {
		switch it.Kind {
		case DriftDeleted:
			s.Deleted++
		case DriftModified:
			s.Modified++
		case DriftTag:
			s.TagChanges++
		}
	}
	s.Total = len(r.Items)
	r.Summary = s
}

// Finalize sorts/normalizes and recomputes the summary. Call after all items
// have been appended.
func (r *DriftReport) Finalize() {
	r.recount()
}
