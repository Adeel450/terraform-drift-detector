// Package store persists drift reports to the filesystem as JSON, providing the
// scan history that powers the dashboard and scheduled runs. It uses only the
// standard library (no database, no CGO).
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adeel450/terraform-drift-detector/internal/model"
)

// Store is a filesystem-backed report store rooted at Dir.
type Store struct {
	Dir string
}

// New returns a Store rooted at dir, creating the directory if needed.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir %q: %w", dir, err)
	}
	return &Store{Dir: dir}, nil
}

// Save writes a report as <ScanID>.json.
func (s *Store) Save(r model.DriftReport) error {
	if r.ScanID == "" {
		return fmt.Errorf("report has no scan id")
	}
	path := filepath.Join(s.Dir, r.ScanID+".json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report %q: %w", path, err)
	}
	return nil
}

// Get loads a single report by scan id.
func (s *Store) Get(scanID string) (model.DriftReport, error) {
	var r model.DriftReport
	path := filepath.Join(s.Dir, scanID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return r, fmt.Errorf("read report %q: %w", scanID, err)
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("parse report %q: %w", scanID, err)
	}
	return r, nil
}

// List returns all stored reports, newest first (scan ids are timestamp-sorted).
func (s *Store) List() ([]model.DriftReport, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("list store dir %q: %w", s.Dir, err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))

	var out []model.DriftReport
	for _, id := range ids {
		r, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
