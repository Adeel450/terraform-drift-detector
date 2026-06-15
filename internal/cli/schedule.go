package cli

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/adeel450/terraform-drift-detector/internal/config"
	"github.com/adeel450/terraform-drift-detector/internal/schedule"
	"github.com/adeel450/terraform-drift-detector/internal/store"
)

func runSchedule(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("schedule", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "config file defining scan targets and schedules (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *cfgPath == "" {
		fmt.Fprintln(os.Stderr, "schedule: --config is required")
		return 2
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "schedule:", err)
		return 1
	}
	st, err := store.New(cfg.StoreDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "schedule:", err)
		return 1
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	sched := schedule.New(cfg, st, logger)
	n, err := sched.Start(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "schedule:", err)
		return 1
	}
	if n == 0 {
		fmt.Fprintln(os.Stderr, "schedule: no targets have a schedule; nothing to run")
		return 1
	}
	logger.Printf("scheduler running with %d target(s); press Ctrl-C to stop", n)

	<-ctx.Done()
	logger.Println("shutting down scheduler...")
	sched.Stop()
	return 0
}
