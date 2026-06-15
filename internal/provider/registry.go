package provider

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// registry holds the known provider factories keyed by provider name.
var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

// Register makes a provider factory available by name. It is intended to be
// called from provider package init() functions. Registering the same name
// twice panics, surfacing wiring mistakes at startup.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := factories[name]; exists {
		panic(fmt.Sprintf("provider %q already registered", name))
	}
	factories[name] = f
}

// New constructs the named provider.
func New(ctx context.Context, name string, opts Options) (Provider, error) {
	mu.RLock()
	f, ok := factories[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown provider %q (known: %v)", name, Names())
	}
	return f(ctx, opts)
}

// Names returns the sorted list of registered provider names.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(factories))
	for n := range factories {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// MapperIndex builds a lookup of Terraform type -> ResourceMapper for a
// provider, so the scanner can resolve the right mapper per state instance.
func MapperIndex(p Provider) map[string]ResourceMapper {
	idx := map[string]ResourceMapper{}
	for _, m := range p.Mappers() {
		idx[m.TerraformType()] = m
	}
	return idx
}
