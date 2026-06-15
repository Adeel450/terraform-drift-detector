// Package runner ties a configured scan target to the scan engine: it builds
// the state source and provider, runs the scan, and persists the report. It is
// shared by the CLI scheduler, the headless schedule command, and the
// dashboard's "run now" action so they behave identically.
package runner

import (
	"context"
	"fmt"

	"github.com/adeel450/terraform-drift-detector/internal/config"
	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/provider"
	"github.com/adeel450/terraform-drift-detector/internal/scan"
	"github.com/adeel450/terraform-drift-detector/internal/store"
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

// Run executes a single scan target and, if st is non-nil, persists the report.
func Run(ctx context.Context, t config.Target, st *store.Store) (model.DriftReport, error) {
	src, err := tfstate.Open(t.State.Path, stateRegion(t))
	if err != nil {
		return model.DriftReport{}, fmt.Errorf("target %q: %w", t.Name, err)
	}
	p, err := provider.New(ctx, t.Provider, provider.Options{
		Region:         t.Region,
		Project:        t.Project,
		SubscriptionID: t.Subscription,
		Extra:          t.Options,
	})
	if err != nil {
		return model.DriftReport{}, fmt.Errorf("target %q: %w", t.Name, err)
	}
	rep, err := scan.Scan(ctx, src, p, scan.Options{})
	if err != nil {
		return model.DriftReport{}, fmt.Errorf("target %q: %w", t.Name, err)
	}
	if st != nil {
		if err := st.Save(rep); err != nil {
			return rep, fmt.Errorf("target %q: save report: %w", t.Name, err)
		}
	}
	return rep, nil
}

// stateRegion returns the region a state backend should use, preferring the
// state-specific region and falling back to the provider region.
func stateRegion(t config.Target) string {
	if t.State.Region != "" {
		return t.State.Region
	}
	return t.Region
}
