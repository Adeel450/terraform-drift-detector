// Package azure implements the Azure cloud provider using the Azure SDK for Go
// (track 2). It supports a representative set of resource types; the resource
// model ID is mapper-defined (e.g. "resourceGroup/account") so that FromState
// and FetchActual agree on the join key and the API call can recover the parts
// it needs.
package azure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/provider"
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

func init() {
	provider.Register("azure", New)
}

// Provider is the Azure provider.
type Provider struct {
	groups   *armresources.ResourceGroupsClient
	accounts *armstorage.AccountsClient
}

// New constructs the Azure provider using DefaultAzureCredential (environment,
// managed identity, Azure CLI, etc.). A subscription id is required.
func New(_ context.Context, opts provider.Options) (provider.Provider, error) {
	if opts.SubscriptionID == "" {
		return nil, fmt.Errorf("azure provider requires --subscription <id>")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure credentials: %w", err)
	}
	groups, err := armresources.NewResourceGroupsClient(opts.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure resource groups client: %w", err)
	}
	accounts, err := armstorage.NewAccountsClient(opts.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure storage accounts client: %w", err)
	}
	return &Provider{groups: groups, accounts: accounts}, nil
}

// Name implements provider.Provider.
func (p *Provider) Name() string { return "azure" }

// Mappers implements provider.Provider.
func (p *Provider) Mappers() []provider.ResourceMapper {
	return []provider.ResourceMapper{
		&resourceGroupMapper{p: p},
		&storageAccountMapper{p: p},
	}
}

// --- azurerm_resource_group ---

type resourceGroupMapper struct{ p *Provider }

func (m *resourceGroupMapper) TerraformType() string { return "azurerm_resource_group" }

func (m *resourceGroupMapper) FromState(inst tfstate.Instance) (model.Resource, error) {
	name := tfstate.AttrString(inst.Attributes, "name")
	return model.Resource{
		Provider:   "azure",
		Type:       "azurerm_resource_group",
		ID:         name,
		Name:       inst.Name,
		Attributes: map[string]string{"location": tfstate.AttrString(inst.Attributes, "location")},
		Tags:       tfstate.AttrTags(inst.Attributes, "tags"),
	}, nil
}

func (m *resourceGroupMapper) FetchActual(ctx context.Context, id string) (model.Resource, bool, error) {
	resp, err := m.p.groups.Get(ctx, id, nil)
	if err != nil {
		if isNotFound(err) {
			return model.Resource{}, false, nil
		}
		return model.Resource{}, false, err
	}
	return model.Resource{
		Provider:   "azure",
		Type:       "azurerm_resource_group",
		ID:         id,
		Attributes: map[string]string{"location": deref(resp.Location)},
		Tags:       azureTags(resp.Tags),
	}, true, nil
}

// --- azurerm_storage_account ---

type storageAccountMapper struct{ p *Provider }

func (m *storageAccountMapper) TerraformType() string { return "azurerm_storage_account" }

func (m *storageAccountMapper) FromState(inst tfstate.Instance) (model.Resource, error) {
	rg := tfstate.AttrString(inst.Attributes, "resource_group_name")
	name := tfstate.AttrString(inst.Attributes, "name")
	return model.Resource{
		Provider: "azure",
		Type:     "azurerm_storage_account",
		ID:       rg + "/" + name,
		Name:     inst.Name,
		Attributes: map[string]string{
			"location":     tfstate.AttrString(inst.Attributes, "location"),
			"account_kind": tfstate.AttrString(inst.Attributes, "account_kind"),
		},
		Tags: tfstate.AttrTags(inst.Attributes, "tags"),
	}, nil
}

func (m *storageAccountMapper) FetchActual(ctx context.Context, id string) (model.Resource, bool, error) {
	rg, name, ok := strings.Cut(id, "/")
	if !ok {
		return model.Resource{}, false, fmt.Errorf("invalid storage account id %q (want resourceGroup/account)", id)
	}
	resp, err := m.p.accounts.GetProperties(ctx, rg, name, nil)
	if err != nil {
		if isNotFound(err) {
			return model.Resource{}, false, nil
		}
		return model.Resource{}, false, err
	}
	kind := ""
	if resp.Kind != nil {
		kind = string(*resp.Kind)
	}
	return model.Resource{
		Provider: "azure",
		Type:     "azurerm_storage_account",
		ID:       id,
		Attributes: map[string]string{
			"location":     deref(resp.Location),
			"account_kind": kind,
		},
		Tags: azureTags(resp.Tags),
	}, true, nil
}

// --- helpers ---

func azureTags(tags map[string]*string) map[string]string {
	out := map[string]string{}
	for k, v := range tags {
		out[k] = deref(v)
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// isNotFound reports whether err is an Azure 404 response.
func isNotFound(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusNotFound
	}
	return false
}
