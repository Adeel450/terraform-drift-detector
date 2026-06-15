package server

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/store"
)

func seededServer(t *testing.T) *Server {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rep := model.DriftReport{
		ScanID: "20260101T000000Z", Timestamp: time.Unix(0, 0).UTC(), Provider: "mock", Source: "file://x",
		Items: []model.DriftItem{
			{Key: model.ResourceKey{Type: "aws_instance", ID: "i-1"}, Kind: model.DriftModified, Field: "instance_type", Expected: "t3.small", Actual: "t3.large", Message: "changed"},
		},
	}
	rep.Summary.ResourcesChecked = 1
	rep.Finalize()
	if err := st.Save(rep); err != nil {
		t.Fatal(err)
	}
	return New(st, nil, log.New(io.Discard, "", 0))
}

func TestServer_IndexAndDetailAndAPI(t *testing.T) {
	h := seededServer(t).Handler()

	cases := []struct {
		path     string
		wantCode int
		contains string
	}{
		{"/", http.StatusOK, "Terraform Drift Dashboard"},
		{"/scan/20260101T000000Z", http.StatusOK, "instance_type"},
		{"/api/scans", http.StatusOK, `"scan_id"`},
		{"/api/scans/20260101T000000Z", http.StatusOK, `"total": 1`},
		{"/scan/missing", http.StatusNotFound, ""},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.wantCode {
			t.Errorf("%s: code = %d, want %d", c.path, rec.Code, c.wantCode)
		}
		if c.contains != "" && !strings.Contains(rec.Body.String(), c.contains) {
			t.Errorf("%s: body missing %q", c.path, c.contains)
		}
	}
}

func TestServer_RunScanWithoutConfig(t *testing.T) {
	h := seededServer(t).Handler()
	req := httptest.NewRequest(http.MethodPost, "/scan", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("run-now without config: code = %d, want 400", rec.Code)
	}
}
