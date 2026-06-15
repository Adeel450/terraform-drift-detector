# terraform-drift-detector

A cloud-agnostic **Terraform drift detection** platform. It compares the
**expected** infrastructure recorded in Terraform state against the **actual**
infrastructure reported by cloud provider APIs, and surfaces drift — deleted
resources, modified attributes, and tag changes — through a CLI, JSON output,
and a web dashboard. Scans run on demand or on a schedule.

It does **not** run `terraform plan` or `apply`. It reads the state file
directly and queries cloud APIs, so scans are fast and side-effect free.

## How it works

```
read tfstate ─┐
              ├─► []Resource (expected) ─┐
provider map ─┘                          ├─► diff ─► DriftReport ─► CLI / JSON / dashboard
cloud APIs ──► []Resource (actual) ──────┘                              (+ filesystem store)
```

1. **Read state** — parse a Terraform `.tfstate` (local file or S3) into resource instances.
2. **Normalize** — each resource type's *mapper* turns both the state side and the
   live cloud side into a common `Resource{Type, ID, Attributes, Tags}` model.
3. **Diff** — join expected vs actual by `(type, id)` and emit findings:
   - **deleted** — in state, not found in the cloud
   - **modified** — a comparable attribute differs
   - **tag** — a tag/label was added, removed, or changed
4. **Report** — render as text/JSON, persist to the store, and serve on the dashboard.

### Extensibility

The system is kept cloud-agnostic by a single seam, `provider.ResourceMapper`
(`internal/provider/provider.go`):

```go
type ResourceMapper interface {
    TerraformType() string                                   // e.g. "aws_instance"
    FromState(inst tfstate.Instance) (model.Resource, error) // expected, from state
    FetchActual(ctx, id string) (model.Resource, bool, error)// actual, from cloud
}
```

Adding a resource type or a whole cloud means implementing mappers and
registering a `Provider` — nothing in the diff engine, scanner, reporters, or
CLI changes. State backends register the same way (`tfstate.RegisterBackend`).

## Project layout

| Path | Responsibility |
|------|----------------|
| `cmd/driftdetector` | CLI entrypoint |
| `internal/model` | normalized `Resource` + `DriftReport` model |
| `internal/tfstate` | state parsing, `StateSource`, backend registry (local + s3) |
| `internal/provider` | `Provider`/`ResourceMapper` interfaces + registry |
| `internal/provider/{aws,azure,gcp,mock}` | provider implementations |
| `internal/diff` | expected-vs-actual comparison |
| `internal/scan` | scan orchestrator (bounded concurrency) |
| `internal/report` | text / JSON / HTML reporters |
| `internal/store` | filesystem report store (history) |
| `internal/schedule` | cron scheduler |
| `internal/server` | web dashboard + JSON API |
| `internal/config` | YAML config loader |
| `internal/runner` | shared "run a target" used by CLI, scheduler, dashboard |

## Supported providers

| Provider | Resource types (representative) | State backends |
|----------|----------------------------------|----------------|
| `aws`    | `aws_instance`, `aws_security_group`, `aws_s3_bucket` | local, `s3://` |
| `azure`  | `azurerm_resource_group`, `azurerm_storage_account`   | local |
| `gcp`    | `google_compute_instance`, `google_storage_bucket`    | local |
| `mock`   | any type declared in a fixture (no credentials)       | local |

Coverage is intentionally representative; extend it by adding mappers.

## Build

Requires Go 1.23+.

```bash
go build -o driftdetector ./cmd/driftdetector
```

## Usage

### Scan (on demand)

```bash
# Mock provider — no cloud credentials needed (great for a demo / CI of the tool itself)
driftdetector scan \
  --state testdata/sample.tfstate \
  --provider mock \
  --provider-opt fixture=testdata/mock_actual.json \
  --output text

# AWS, state on disk
driftdetector scan --state ./terraform.tfstate --provider aws --region us-east-1 --output json

# AWS, state in S3
driftdetector scan --state s3://my-bucket/prod/terraform.tfstate --provider aws --region us-east-1

# Azure / GCP
driftdetector scan --state ./azure.tfstate --provider azure --subscription <sub-id>
driftdetector scan --state ./gcp.tfstate   --provider gcp   --project <project-id>
```

Useful flags: `--out <file>` (write report to a file), `--store <dir>` (persist
to the store), `--exit-code` (exit `3` when drift is found, for CI gates),
`--concurrency N`.

Cloud credentials use each SDK's default chain (AWS: env/profile/role; Azure:
`DefaultAzureCredential`; GCP: Application Default Credentials).

### Dashboard

```bash
driftdetector serve --addr :8080 --store ./reports
# with scheduling + a "Run scan now" button:
driftdetector serve --config configs/config.example.yaml
```

- `/` — list of scans with drift status
- `/scan/<id>` — drift detail
- `/api/scans`, `/api/scans/<id>` — JSON

### Scheduled scans (headless)

```bash
driftdetector schedule --config configs/config.example.yaml
```

Each target with a `schedule` (cron expression) is scanned automatically and the
report is persisted to the store. See `configs/config.example.yaml`.

### Discover providers

```bash
driftdetector providers
```

## Configuration

`configs/config.example.yaml` defines the store directory, dashboard address,
and a list of scan targets (state source + provider + optional cron schedule).

## Testing & verification

```bash
go test ./...      # unit + end-to-end pipeline tests (no cloud credentials needed)
go vet ./...
go build ./...
```

The `mock` provider drives the full parse → diff → report pipeline against
`testdata/`, so correctness is verified without any cloud account. Live cloud
scans require credentials and are exercised manually with the `scan` command.

## Exit codes

`0` success/no drift · `1` error · `2` usage error · `3` drift detected (with `--exit-code`).
