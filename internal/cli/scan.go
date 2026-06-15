package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/adeel450/terraform-drift-detector/internal/provider"
	"github.com/adeel450/terraform-drift-detector/internal/report"
	"github.com/adeel450/terraform-drift-detector/internal/scan"
	"github.com/adeel450/terraform-drift-detector/internal/store"
)

// stringSlice is a flag.Value collecting repeated string flags.
type stringSlice []string

func (s *stringSlice) String() string { return fmt.Sprintf("%v", *s) }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runScan(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	var (
		stateRef     = fs.String("state", "", "Terraform state location: a file path or s3://bucket/key")
		providerNm   = fs.String("provider", "", "cloud provider: "+fmt.Sprintf("%v", provider.Names()))
		output       = fs.String("output", "text", "output format: text|json")
		outFile      = fs.String("out", "", "write the report to this file instead of stdout")
		storeDir     = fs.String("store", "", "if set, also persist the report to this store directory")
		region       = fs.String("region", "", "provider region (e.g. AWS region)")
		project      = fs.String("project", "", "GCP project id")
		subscription = fs.String("subscription", "", "Azure subscription id")
		concurrency  = fs.Int("concurrency", 8, "max concurrent cloud lookups")
		exitOnDrift  = fs.Bool("exit-code", false, "exit with code 3 when drift is detected (for CI)")
		opts         stringSlice
	)
	fs.Var(&opts, "provider-opt", "provider-specific option key=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *stateRef == "" || *providerNm == "" {
		fmt.Fprintln(os.Stderr, "scan: --state and --provider are required")
		fs.Usage()
		return 2
	}

	src, err := stateSourceFor(*stateRef, *region)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		return 1
	}

	p, err := provider.New(ctx, *providerNm, provider.Options{
		Region:         *region,
		Project:        *project,
		SubscriptionID: *subscription,
		Extra:          parseProviderOpts(opts),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		return 1
	}

	rep, err := scan.Scan(ctx, src, p, scan.Options{Concurrency: *concurrency})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		return 1
	}

	if *storeDir != "" {
		st, err := store.New(*storeDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "scan:", err)
			return 1
		}
		if err := st.Save(rep); err != nil {
			fmt.Fprintln(os.Stderr, "scan:", err)
			return 1
		}
	}

	out := os.Stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "scan:", err)
			return 1
		}
		defer f.Close()
		out = f
	}

	switch *output {
	case "json":
		err = report.JSON(out, rep)
	case "text":
		err = report.Text(out, rep)
	default:
		fmt.Fprintf(os.Stderr, "scan: unknown output format %q\n", *output)
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		return 1
	}

	if *exitOnDrift && rep.HasDrift() {
		return 3
	}
	return 0
}
