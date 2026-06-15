package report

import (
	"html/template"
	"io"

	"github.com/adeel450/terraform-drift-detector/internal/model"
)

// kindClass maps a drift kind to a CSS class for badge coloring.
func kindClass(k model.DriftKind) string {
	switch k {
	case model.DriftDeleted:
		return "badge-deleted"
	case model.DriftModified:
		return "badge-modified"
	case model.DriftTag:
		return "badge-tag"
	default:
		return ""
	}
}

var funcs = template.FuncMap{"kindClass": kindClass}

var detailTmpl = template.Must(template.New("detail").Funcs(funcs).Parse(detailHTML))

// HTMLDetail renders the full drift report detail page.
func HTMLDetail(w io.Writer, r model.DriftReport) error {
	return detailTmpl.Execute(w, r)
}

const detailHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>Drift report {{.ScanID}}</title>
<style>` + sharedCSS + `</style></head>
<body>
<header><a href="/" class="back">&larr; All scans</a><h1>Drift report</h1></header>
<section class="meta">
  <div><span>Scan</span>{{.ScanID}}</div>
  <div><span>Provider</span>{{.Provider}}</div>
  <div><span>Source</span>{{.Source}}</div>
  <div><span>Time</span>{{.Timestamp.Format "2006-01-02 15:04:05 MST"}}</div>
  <div><span>Checked</span>{{.Summary.ResourcesChecked}} resource(s)</div>
</section>
{{if .HasDrift}}
<section class="summary">
  <span class="pill">{{.Summary.Total}} findings</span>
  <span class="pill badge-deleted">{{.Summary.Deleted}} deleted</span>
  <span class="pill badge-modified">{{.Summary.Modified}} modified</span>
  <span class="pill badge-tag">{{.Summary.TagChanges}} tag</span>
</section>
<table>
<thead><tr><th>Resource</th><th>Kind</th><th>Field</th><th>Expected</th><th>Actual</th><th>Detail</th></tr></thead>
<tbody>
{{range .Items}}
<tr>
  <td class="mono">{{.Key.String}}</td>
  <td><span class="badge {{kindClass .Kind}}">{{.Kind}}</span></td>
  <td class="mono">{{if .Field}}{{.Field}}{{else}}&mdash;{{end}}</td>
  <td class="mono">{{if .Expected}}{{.Expected}}{{else}}&mdash;{{end}}</td>
  <td class="mono">{{if .Actual}}{{.Actual}}{{else}}&mdash;{{end}}</td>
  <td>{{.Message}}</td>
</tr>
{{end}}
</tbody>
</table>
{{else}}
<section class="ok">No drift detected. Infrastructure matches Terraform state.</section>
{{end}}
</body></html>`

var indexTmpl = template.Must(template.New("index").Funcs(funcs).Parse(indexHTML))

// IndexData is the view model for the dashboard index.
type IndexData struct {
	Reports []model.DriftReport
}

// HTMLIndex renders the dashboard index listing all scans.
func HTMLIndex(w io.Writer, reports []model.DriftReport) error {
	return indexTmpl.Execute(w, IndexData{Reports: reports})
}

const indexHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>Drift dashboard</title>
<style>` + sharedCSS + `</style></head>
<body>
<header><h1>Terraform Drift Dashboard</h1>
<form method="post" action="/scan"><button class="run" type="submit">Run scan now</button></form>
</header>
{{if .Reports}}
<table>
<thead><tr><th>Scan</th><th>Provider</th><th>Source</th><th>Status</th><th>Findings</th></tr></thead>
<tbody>
{{range .Reports}}
<tr onclick="location='/scan/{{.ScanID}}'" class="clickable">
  <td class="mono">{{.ScanID}}</td>
  <td>{{.Provider}}</td>
  <td class="mono">{{.Source}}</td>
  <td>{{if .HasDrift}}<span class="badge badge-deleted">drift</span>{{else}}<span class="badge badge-ok">clean</span>{{end}}</td>
  <td>{{.Summary.Total}} ({{.Summary.Deleted}}d / {{.Summary.Modified}}m / {{.Summary.TagChanges}}t)</td>
</tr>
{{end}}
</tbody>
</table>
{{else}}
<section class="ok">No scans yet. Run a scan from the CLI or the button above.</section>
{{end}}
</body></html>`

const sharedCSS = `
:root{font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;color:#1c2230}
body{margin:0;background:#f5f6f8}
header{display:flex;align-items:center;gap:1rem;padding:1rem 1.5rem;background:#1c2230;color:#fff}
header h1{font-size:1.1rem;margin:0;flex:1}
a.back{color:#9fb4ff;text-decoration:none}
.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:.85rem}
table{width:calc(100% - 3rem);margin:1.5rem;border-collapse:collapse;background:#fff;box-shadow:0 1px 3px rgba(0,0,0,.08)}
th,td{text-align:left;padding:.55rem .8rem;border-bottom:1px solid #eceef2;font-size:.9rem}
th{background:#fafbfc;font-size:.75rem;text-transform:uppercase;letter-spacing:.04em;color:#6b7280}
tr.clickable{cursor:pointer}
tr.clickable:hover{background:#f0f4ff}
.badge{display:inline-block;padding:.1rem .5rem;border-radius:999px;font-size:.72rem;font-weight:600;color:#fff}
.badge-deleted{background:#d64545}.badge-modified{background:#d99100}.badge-tag{background:#3b7dd8}.badge-ok{background:#2e9e5b}
.summary,.meta{margin:1.5rem 1.5rem 0;display:flex;flex-wrap:wrap;gap:.6rem}
.meta div{background:#fff;padding:.5rem .8rem;border-radius:6px;font-size:.85rem}
.meta span{display:block;color:#6b7280;font-size:.7rem;text-transform:uppercase}
.pill{display:inline-block;padding:.25rem .7rem;border-radius:999px;background:#e7eaf0;font-size:.8rem;font-weight:600}
.pill.badge-deleted,.pill.badge-modified,.pill.badge-tag{color:#fff}
.ok{margin:1.5rem;padding:1.2rem;background:#e7f6ed;border:1px solid #b6e2c6;border-radius:8px;color:#1d6b3c}
button.run{background:#2e9e5b;color:#fff;border:0;padding:.45rem .9rem;border-radius:6px;cursor:pointer;font-weight:600}
`
