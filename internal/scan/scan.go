// Package scan orchestrates a drift scan: read state, normalize, fetch live
// resources concurrently, diff, and assemble a report.
package scan

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/adeel450/terraform-drift-detector/internal/diff"
	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/provider"
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

// Options tunes a scan.
type Options struct {
	// Concurrency bounds simultaneous cloud API lookups. Defaults to 8.
	Concurrency int
}

// Scan reads Terraform state from src, fetches the corresponding live resources
// from p, and returns a drift report. Resource types in state that the provider
// has no mapper for are skipped (and counted as skipped, not as drift).
func Scan(ctx context.Context, src tfstate.StateSource, p provider.Provider, opts Options) (model.DriftReport, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 8
	}

	data, sourceDesc, err := src.Read(ctx)
	if err != nil {
		return model.DriftReport{}, err
	}
	instances, err := tfstate.Parse(data)
	if err != nil {
		return model.DriftReport{}, err
	}

	mappers := provider.MapperIndex(p)

	type job struct {
		inst   tfstate.Instance
		mapper provider.ResourceMapper
	}
	var jobs []job
	for _, inst := range instances {
		m, ok := mappers[inst.Type]
		if !ok {
			continue // provider doesn't manage this type; skip silently
		}
		jobs = append(jobs, job{inst: inst, mapper: m})
	}

	var (
		mu       sync.Mutex
		items    []model.DriftItem
		firstErr error
		wg       sync.WaitGroup
		sem      = make(chan struct{}, opts.Concurrency)
	)

	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()

			expected, err := j.mapper.FromState(j.inst)
			if err != nil {
				recordErr(&mu, &firstErr, fmt.Errorf("normalize %s.%s: %w", j.inst.Type, j.inst.Name, err))
				return
			}
			actual, found, err := j.mapper.FetchActual(ctx, expected.ID)
			if err != nil {
				recordErr(&mu, &firstErr, fmt.Errorf("fetch %s: %w", expected.Key(), err))
				return
			}
			drift := diff.Compare(expected, actual, found)
			if len(drift) > 0 {
				mu.Lock()
				items = append(items, drift...)
				mu.Unlock()
			}
		}(j)
	}
	wg.Wait()

	if firstErr != nil {
		return model.DriftReport{}, firstErr
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Key.String() != items[j].Key.String() {
			return items[i].Key.String() < items[j].Key.String()
		}
		return items[i].Field < items[j].Field
	})

	now := time.Now().UTC()
	report := model.DriftReport{
		ScanID:    now.Format("20060102T150405Z"),
		Timestamp: now,
		Provider:  p.Name(),
		Source:    sourceDesc,
		Items:     items,
	}
	report.Summary.ResourcesChecked = len(jobs)
	report.Finalize()
	return report, nil
}

func recordErr(mu *sync.Mutex, dst *error, err error) {
	mu.Lock()
	if *dst == nil {
		*dst = err
	}
	mu.Unlock()
}
