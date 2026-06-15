// Package cli implements the driftdetector command-line interface using only
// the standard library flag package. Subcommands: scan, serve, schedule,
// providers.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/adeel450/terraform-drift-detector/internal/provider"
)

// Run dispatches a subcommand. It returns a process exit code.
func Run(ctx context.Context, args []string) int {
	if len(args) < 1 {
		usage(os.Stderr)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "scan":
		return runScan(ctx, rest)
	case "serve":
		return runServe(ctx, rest)
	case "schedule":
		return runSchedule(ctx, rest)
	case "providers":
		return runProviders(rest)
	case "help", "-h", "--help":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage(os.Stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `driftdetector — Terraform drift detection

Usage:
  driftdetector <command> [flags]

Commands:
  scan        Run a one-off drift scan and print/emit a report
  serve       Serve the web dashboard (optionally with a scheduler)
  schedule    Run scheduled scans headlessly from a config file
  providers   List registered cloud providers and resource types

Run 'driftdetector <command> -h' for command-specific flags.
`)
}

func runProviders(args []string) int {
	_ = args
	fmt.Println("Registered providers:")
	for _, name := range provider.Names() {
		p, err := provider.New(context.Background(), name, provider.Options{})
		if err != nil {
			// Providers that need credentials/config to construct still appear
			// by name; their resource types are listed when constructable.
			fmt.Printf("  - %s\n", name)
			continue
		}
		types := make([]string, 0)
		for _, m := range p.Mappers() {
			types = append(types, m.TerraformType())
		}
		fmt.Printf("  - %s: %s\n", name, strings.Join(types, ", "))
	}
	return 0
}

// parseProviderOpts converts repeated --provider-opt key=value flags into the
// provider.Options.Extra map.
func parseProviderOpts(pairs []string) map[string]string {
	m := map[string]string{}
	for _, p := range pairs {
		if k, v, ok := strings.Cut(p, "="); ok {
			m[strings.TrimSpace(k)] = v
		}
	}
	return m
}
