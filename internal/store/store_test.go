package store

import (
	"testing"
	"time"

	"github.com/adeel450/terraform-drift-detector/internal/model"
)

func TestStore_SaveListGet(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	r1 := model.DriftReport{ScanID: "20260101T000000Z", Timestamp: time.Unix(0, 0).UTC(), Provider: "mock"}
	r2 := model.DriftReport{ScanID: "20260102T000000Z", Timestamp: time.Unix(0, 0).UTC(), Provider: "mock"}
	for _, r := range []model.DriftReport{r1, r2} {
		if err := st.Save(r); err != nil {
			t.Fatal(err)
		}
	}

	list, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 reports, got %d", len(list))
	}
	// Newest first.
	if list[0].ScanID != "20260102T000000Z" {
		t.Errorf("expected newest first, got %s", list[0].ScanID)
	}

	got, err := st.Get("20260101T000000Z")
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "mock" {
		t.Errorf("provider = %q", got.Provider)
	}

	if _, err := st.Get("does-not-exist"); err == nil {
		t.Error("expected error getting missing report")
	}
}
