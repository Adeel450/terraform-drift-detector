// Command driftdetector compares Terraform state against live cloud
// infrastructure and reports configuration drift.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/adeel450/terraform-drift-detector/internal/cli"

	// Register all providers (and their state backends) via init().
	_ "github.com/adeel450/terraform-drift-detector/internal/provider/all"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(cli.Run(ctx, os.Args[1:]))
}
