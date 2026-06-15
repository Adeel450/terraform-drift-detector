// Package mock implements an in-memory Provider driven by a JSON fixture that
// describes the "actual" cloud state. It enables end-to-end testing and local
// demos of the full scan pipeline without any cloud credentials.
//
// Fixture format:
//
//	{
//	  "resources": [
//	    {"type":"aws_instance","id":"i-1","attributes":{"instance_type":"t3.small"},"tags":{"env":"prod"}}
//	  ]
//	}
//
// A resource present in Terraform state but absent from the fixture is reported
// as deleted. The set of comparable attributes for a resource is the set of
// attribute keys present in its fixture entry; the same keys are read from
// state, keeping both sides aligned.
package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/provider"
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

func init() {
	provider.Register("mock", func(_ context.Context, opts provider.Options) (provider.Provider, error) {
		path := opts.Extra["fixture"]
		if path == "" {
			return nil, fmt.Errorf("mock provider requires --provider-opt fixture=<path.json>")
		}
		return Load(path)
	})
}

// actualResource is one fixture entry describing a live resource.
type actualResource struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Attributes map[string]string `json:"attributes"`
	Tags       map[string]string `json:"tags"`
}

type fixture struct {
	// Types optionally enumerates all Terraform types this mock cloud manages.
	// It is needed so that a managed resource present in state but absent from
	// Resources can be detected as deleted (a deleted resource has no entry to
	// infer its type from).
	Types     []string         `json:"types"`
	Resources []actualResource `json:"resources"`
}

// Provider is the mock cloud provider.
type Provider struct {
	// byID maps cloud id -> actual resource.
	byID map[string]actualResource
	// types is the set of Terraform types seen in the fixture.
	types map[string]bool
}

// Load builds a mock provider from a fixture file.
func Load(path string) (*Provider, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mock fixture %q: %w", path, err)
	}
	return FromBytes(data)
}

// FromBytes builds a mock provider from fixture JSON bytes.
func FromBytes(data []byte) (*Provider, error) {
	var f fixture
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse mock fixture: %w", err)
	}
	p := &Provider{byID: map[string]actualResource{}, types: map[string]bool{}}
	for _, t := range f.Types {
		p.types[t] = true
	}
	for _, r := range f.Resources {
		p.byID[r.ID] = r
		p.types[r.Type] = true
	}
	return p, nil
}

// Name implements provider.Provider.
func (p *Provider) Name() string { return "mock" }

// Mappers returns one mapper per Terraform type present in the fixture.
func (p *Provider) Mappers() []provider.ResourceMapper {
	var out []provider.ResourceMapper
	for t := range p.types {
		out = append(out, &mapper{tfType: t, p: p})
	}
	return out
}

// mapper is a generic mock mapper for a single Terraform type.
type mapper struct {
	tfType string
	p      *Provider
}

func (m *mapper) TerraformType() string { return m.tfType }

func (m *mapper) FromState(inst tfstate.Instance) (model.Resource, error) {
	id := tfstate.AttrString(inst.Attributes, "id")
	res := model.Resource{
		Provider:   "mock",
		Type:       inst.Type,
		ID:         id,
		Name:       inst.Name,
		Attributes: map[string]string{},
		Tags:       tfstate.AttrTags(inst.Attributes, "tags"),
	}
	// Compare exactly the attribute keys declared for this resource in the
	// fixture, so expected/actual key sets line up.
	if actual, ok := m.p.byID[id]; ok {
		for k := range actual.Attributes {
			res.Attributes[k] = tfstate.AttrString(inst.Attributes, k)
		}
	} else {
		// No fixture entry: surface all string attributes so a present-vs-
		// deleted comparison still carries some context.
		for k, v := range inst.Attributes {
			if s, ok := v.(string); ok {
				res.Attributes[k] = s
			}
		}
	}
	return res, nil
}

func (m *mapper) FetchActual(_ context.Context, id string) (model.Resource, bool, error) {
	actual, ok := m.p.byID[id]
	if !ok {
		return model.Resource{}, false, nil
	}
	attrs := map[string]string{}
	for k, v := range actual.Attributes {
		attrs[k] = v
	}
	tags := map[string]string{}
	for k, v := range actual.Tags {
		tags[k] = v
	}
	return model.Resource{
		Provider:   "mock",
		Type:       actual.Type,
		ID:         actual.ID,
		Name:       "",
		Attributes: attrs,
		Tags:       tags,
	}, true, nil
}
