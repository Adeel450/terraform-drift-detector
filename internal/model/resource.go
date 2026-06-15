// Package model defines the cloud-agnostic normalized representation of
// infrastructure resources and the drift report produced by comparing the
// expected state (from Terraform) against the actual state (from a cloud API).
package model

import "fmt"

// Resource is the normalized, provider-agnostic view of a single piece of
// infrastructure. Both the Terraform state side ("expected") and the cloud API
// side ("actual") are normalized into this shape so they can be compared
// directly.
type Resource struct {
	// Provider is the logical provider name, e.g. "aws", "azure", "gcp".
	Provider string `json:"provider"`
	// Type is the Terraform resource type, e.g. "aws_instance".
	Type string `json:"type"`
	// ID is the canonical cloud identifier used to join expected and actual.
	ID string `json:"id"`
	// Name is the Terraform resource name (the local label), for display.
	Name string `json:"name"`
	// Attributes holds the comparable attributes selected by the resource
	// mapper. Only attributes present here participate in drift detection,
	// which keeps computed/derived noise out of reports.
	Attributes map[string]string `json:"attributes"`
	// Tags holds resource tags/labels, compared independently of Attributes.
	Tags map[string]string `json:"tags"`
}

// Key returns the join key used to match an expected resource with its actual
// counterpart.
func (r Resource) Key() ResourceKey {
	return ResourceKey{Type: r.Type, ID: r.ID}
}

// ResourceKey uniquely identifies a resource across the expected/actual sets.
type ResourceKey struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// String renders the key as "type[id]".
func (k ResourceKey) String() string {
	return fmt.Sprintf("%s[%s]", k.Type, k.ID)
}
