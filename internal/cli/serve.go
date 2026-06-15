package cli

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/adeel450/terraform-drift-detector/internal/config"
	"github.com/adeel450/terraform-drift-detector/internal/schedule"
	"github.com/adeel450/terraform-drift-detector/internal/server"
	"github.com/adeel450/terraform-drift-detector/internal/store"
)

func runServe(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	var (
		addr     = fs.String("addr", ":8080", "dashboard listen address")
		storeDir = fs.String("store", "./reports", "report store directory")
		cfgPath  = fs.String("config", "", "optional config file; enables scheduled scans and the 'run now' button")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)

	var cfg *config.Config
	if *cfgPath != "" {
		c, err := config.Load(*cfgPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "serve:", err)
			return 1
		}
		cfg = c
		// Config store dir wins when a config is supplied.
		*storeDir = c.StoreDir
		if !isFlagSet(fs, "addr") {
			*addr = c.Addr
		}
	}

	st, err := store.New(*storeDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		return 1
	}

	// Start the scheduler in-process when a config defines schedules.
	if cfg != nil {
		sched := schedule.New(cfg, st, logger)
		n, err := sched.Start(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "serve: scheduler:", err)
			return 1
		}
		if n > 0 {
			logger.Printf("started scheduler with %d target(s)", n)
		}
		defer sched.Stop()
	}

	srv := server.New(st, cfg, logger)
	if err := srv.ListenAndServe(ctx, *addr); err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		return 1
	}
	return 0
}

// isFlagSet reports whether the named flag was explicitly provided.
func isFlagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
