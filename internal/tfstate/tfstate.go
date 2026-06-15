// Package tfstate reads and parses Terraform state into a flat list of
// resource instances. It deliberately does NOT shell out to the terraform
// binary or run plan/apply — it parses the state JSON directly so scans are
// fast and side-effect free.
package tfstate

import (
	"context"
	"encoding/json"
	"fmt"
)

// Instance is one materialized resource instance from Terraform state, with the
// raw attribute bag preserved for provider-specific normalization.
type Instance struct {
	// ProviderType is the provider prefix derived from Type, e.g. "aws".
	ProviderType string
	// Type is the Terraform resource type, e.g. "aws_instance".
	Type string
	// Name is the Terraform resource local name.
	Name string
	// Module is the module path ("" for root).
	Module string
	// Attributes is the raw attribute map from the state instance.
	Attributes map[string]any
}

// StateSource yields raw Terraform state bytes from some backend (local file,
// S3, etc.). Implementations live alongside this package.
type StateSource interface {
	// Read returns the raw state JSON. Description is a human-readable label
	// for the source (used in reports).
	Read(ctx context.Context) (data []byte, description string, err error)
}

// stateFile mirrors the subset of the Terraform state JSON schema we need.
type stateFile struct {
	Version          int    `json:"version"`
	TerraformVersion string `json:"terraform_version"`
	Resources        []struct {
		Module    string `json:"module"`
		Mode      string `json:"mode"`
		Type      string `json:"type"`
		Name      string `json:"name"`
		Provider  string `json:"provider"`
		Instances []struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"instances"`
	} `json:"resources"`
}

// Parse converts raw state JSON into a flat list of managed resource instances.
// Data resources (mode != "managed") are skipped — they are read-only and not
// candidates for drift.
func Parse(data []byte) ([]Instance, error) {
	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse terraform state: %w", err)
	}
	var out []Instance
	for _, r := range sf.Resources {
		if r.Mode != "" && r.Mode != "managed" {
			continue
		}
		for _, inst := range r.Instances {
			out = append(out, Instance{
				ProviderType: providerPrefix(r.Type),
				Type:         r.Type,
				Name:         r.Name,
				Module:       r.Module,
				Attributes:   inst.Attributes,
			})
		}
	}
	return out, nil
}

// providerPrefix extracts the provider portion of a Terraform type, e.g.
// "aws_instance" -> "aws", "google_storage_bucket" -> "google".
func providerPrefix(tfType string) string {
	for i := 0; i < len(tfType); i++ {
		if tfType[i] == '_' {
			return tfType[:i]
		}
	}
	return tfType
}

// String helpers for pulling typed values out of the raw attribute bag. These
// are exported for use by provider resource mappers.

// AttrString returns attrs[key] as a string, or "" if absent/not a string.
func AttrString(attrs map[string]any, key string) string {
	v, ok := attrs[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		// JSON numbers decode as float64; render integers without a decimal.
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// AttrTags returns attrs[key] interpreted as a string->string tag map. Handles
// the common Terraform shape where tags is a map[string]any of strings.
func AttrTags(attrs map[string]any, key string) map[string]string {
	out := map[string]string{}
	raw, ok := attrs[key]
	if !ok || raw == nil {
		return out
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
