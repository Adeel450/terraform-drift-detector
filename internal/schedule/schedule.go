// Package schedule runs configured scan targets on cron schedules, persisting
// each resulting report to the store.
package schedule

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"

	"github.com/adeel450/terraform-drift-detector/internal/config"
	"github.com/adeel450/terraform-drift-detector/internal/runner"
	"github.com/adeel450/terraform-drift-detector/internal/store"
)

// Scheduler runs scan targets on their cron schedules.
type Scheduler struct {
	cfg   *config.Config
	store *store.Store
	cron  *cron.Cron
	log   *log.Logger
}

// New builds a Scheduler. Targets without a Schedule are ignored (they are
// on-demand only).
func New(cfg *config.Config, st *store.Store, logger *log.Logger) *Scheduler {
	return &Scheduler{
		cfg:   cfg,
		store: st,
		cron:  cron.New(),
		log:   logger,
	}
}

// Start registers all scheduled targets and begins the cron loop. It returns
// the number of scheduled targets, or an error if a cron expression is invalid.
func (s *Scheduler) Start(ctx context.Context) (int, error) {
	count := 0
	for _, t := range s.cfg.Targets {
		if t.Schedule == "" {
			continue
		}
		t := t // capture
		_, err := s.cron.AddFunc(t.Schedule, func() { s.runOne(ctx, t) })
		if err != nil {
			return 0, err
		}
		count++
		s.log.Printf("scheduled target %q with cron %q", t.Name, t.Schedule)
	}
	s.cron.Start()
	return count, nil
}

// Stop halts the scheduler, waiting for running jobs to finish.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

func (s *Scheduler) runOne(ctx context.Context, t config.Target) {
	rep, err := runner.Run(ctx, t, s.store)
	if err != nil {
		s.log.Printf("scan target %q failed: %v", t.Name, err)
		return
	}
	s.log.Printf("scan target %q complete: %d finding(s) (%s)", t.Name, rep.Summary.Total, rep.ScanID)
}
