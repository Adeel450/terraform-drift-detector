// Package gcp implements the Google Cloud provider using the cloud.google.com/go
// client libraries. It supports a representative set of resource types. As with
// the other providers, the resource model ID is mapper-defined so the API call
// can recover the parts it needs (e.g. "zone/instance").
package gcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"

	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/provider"
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

func init() {
	provider.Register("gcp", New)
}

// Provider is the GCP provider.
type Provider struct {
	project   string
	instances *compute.InstancesClient
	storage   *storage.Client
}

// New constructs the GCP provider using Application Default Credentials. A
// project id is required.
func New(ctx context.Context, opts provider.Options) (provider.Provider, error) {
	if opts.Project == "" {
		return nil, fmt.Errorf("gcp provider requires --project <id>")
	}
	instances, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp compute client: %w", err)
	}
	store, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp storage client: %w", err)
	}
	return &Provider{project: opts.Project, instances: instances, storage: store}, nil
}

// Name implements provider.Provider.
func (p *Provider) Name() string { return "gcp" }

// Mappers implements provider.Provider.
func (p *Provider) Mappers() []provider.ResourceMapper {
	return []provider.ResourceMapper{
		&computeInstanceMapper{p: p},
		&storageBucketMapper{p: p},
	}
}

// --- google_compute_instance ---

type computeInstanceMapper struct{ p *Provider }

func (m *computeInstanceMapper) TerraformType() string { return "google_compute_instance" }

func (m *computeInstanceMapper) FromState(inst tfstate.Instance) (model.Resource, error) {
	zone := tfstate.AttrString(inst.Attributes, "zone")
	name := tfstate.AttrString(inst.Attributes, "name")
	return model.Resource{
		Provider: "gcp",
		Type:     "google_compute_instance",
		ID:       zone + "/" + name,
		Name:     inst.Name,
		Attributes: map[string]string{
			"machine_type": lastSegment(tfstate.AttrString(inst.Attributes, "machine_type")),
		},
		Tags: tfstate.AttrTags(inst.Attributes, "labels"),
	}, nil
}

func (m *computeInstanceMapper) FetchActual(ctx context.Context, id string) (model.Resource, bool, error) {
	zone, name, ok := strings.Cut(id, "/")
	if !ok {
		return model.Resource{}, false, fmt.Errorf("invalid instance id %q (want zone/name)", id)
	}
	inst, err := m.p.instances.Get(ctx, &computepb.GetInstanceRequest{
		Project:  m.p.project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		if isNotFound(err) {
			return model.Resource{}, false, nil
		}
		return model.Resource{}, false, err
	}
	return model.Resource{
		Provider: "gcp",
		Type:     "google_compute_instance",
		ID:       id,
		Attributes: map[string]string{
			"machine_type": lastSegment(inst.GetMachineType()),
		},
		Tags: inst.GetLabels(),
	}, true, nil
}

// --- google_storage_bucket ---

type storageBucketMapper struct{ p *Provider }

func (m *storageBucketMapper) TerraformType() string { return "google_storage_bucket" }

func (m *storageBucketMapper) FromState(inst tfstate.Instance) (model.Resource, error) {
	return model.Resource{
		Provider: "gcp",
		Type:     "google_storage_bucket",
		ID:       tfstate.AttrString(inst.Attributes, "name"),
		Name:     inst.Name,
		Attributes: map[string]string{
			"location": strings.ToUpper(tfstate.AttrString(inst.Attributes, "location")),
		},
		Tags: tfstate.AttrTags(inst.Attributes, "labels"),
	}, nil
}

func (m *storageBucketMapper) FetchActual(ctx context.Context, id string) (model.Resource, bool, error) {
	attrs, err := m.p.storage.Bucket(id).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) || isNotFound(err) {
			return model.Resource{}, false, nil
		}
		return model.Resource{}, false, err
	}
	return model.Resource{
		Provider: "gcp",
		Type:     "google_storage_bucket",
		ID:       id,
		Attributes: map[string]string{
			"location": strings.ToUpper(attrs.Location),
		},
		Tags: attrs.Labels,
	}, true, nil
}

// --- helpers ---

// lastSegment returns the final path/URL segment, normalizing GCP self-links
// like ".../machineTypes/e2-medium" to "e2-medium".
func lastSegment(s string) string {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// isNotFound reports whether err is a Google API 404.
func isNotFound(err error) bool {
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return gerr.Code == 404
	}
	return false
}
