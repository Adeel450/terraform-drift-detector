// Package provider defines the extension seam that keeps the drift detector
// cloud-agnostic. A Provider groups a set of ResourceMappers, each of which
// knows how to (a) normalize a Terraform state instance into the common model
// and (b) fetch the corresponding live resource from a cloud API.
//
// Adding support for a new cloud or resource type means implementing a
// ResourceMapper and registering a Provider — no changes to the diff engine,
// scanner, reporters, or CLI are required.
package provider

import (
	"context"

	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

// ResourceMapper translates a single Terraform resource type between the state
// representation and the live cloud representation.
type ResourceMapper interface {
	// TerraformType is the Terraform type this mapper handles, e.g.
	// "aws_instance".
	TerraformType() string

	// FromState normalizes a state instance into the expected Resource. The
	// mapper selects which attributes are comparable, keeping computed/derived
	// noise out of drift detection.
	FromState(inst tfstate.Instance) (model.Resource, error)

	// FetchActual retrieves the live resource by its cloud ID. found is false
	// when the resource no longer exists (which the diff engine reports as
	// deleted). The returned Resource should expose the same attribute and tag
	// keys as FromState so they compare cleanly.
	FetchActual(ctx context.Context, id string) (resource model.Resource, found bool, err error)
}

// Provider is a named collection of resource mappers for one cloud.
type Provider interface {
	// Name is the logical provider name, e.g. "aws", "azure", "gcp", "mock".
	Name() string
	// Mappers returns the resource mappers this provider supports.
	Mappers() []ResourceMapper
}

// Factory constructs a Provider. Construction may require credentials or config
// and can fail, hence the error return and context.
type Factory func(ctx context.Context, opts Options) (Provider, error)

// Options carries provider construction settings sourced from CLI flags or
// config. Fields are optional; providers use what applies to them.
type Options struct {
	// Region for providers that are region-scoped (e.g. AWS).
	Region string
	// Project for GCP.
	Project string
	// SubscriptionID for Azure.
	SubscriptionID string
	// Extra holds provider-specific key/values for extensibility.
	Extra map[string]string
}
