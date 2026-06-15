// Package server serves the drift dashboard: an HTML index of scans, per-scan
// detail pages, and a JSON API over the report store.
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/adeel450/terraform-drift-detector/internal/config"
	"github.com/adeel450/terraform-drift-detector/internal/report"
	"github.com/adeel450/terraform-drift-detector/internal/runner"
	"github.com/adeel450/terraform-drift-detector/internal/store"
)

// shutdownTimeout bounds graceful shutdown when the context is canceled.
const shutdownTimeout = 5 * time.Second

// Server is the dashboard HTTP server.
type Server struct {
	store *store.Store
	cfg   *config.Config // optional; enables "run scan now" of the first target
	log   *log.Logger
}

// New constructs a dashboard server. cfg may be nil (the "run now" button then
// reports that no targets are configured).
func New(st *store.Store, cfg *config.Config, logger *log.Logger) *Server {
	return &Server{store: st, cfg: cfg, log: logger}
}

// Handler returns the configured HTTP handler (exposed for testing).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/scan/", s.handleDetail)
	mux.HandleFunc("/scan", s.handleRunScan)
	mux.HandleFunc("/api/scans", s.handleAPIList)
	mux.HandleFunc("/api/scans/", s.handleAPIGet)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	reports, err := s.store.List()
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := report.HTMLIndex(w, reports); err != nil {
		s.log.Printf("render index: %v", err)
	}
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/scan/")
	if id == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	rep, err := s.store.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := report.HTMLDetail(w, rep); err != nil {
		s.log.Printf("render detail: %v", err)
	}
}

// handleRunScan runs the first configured target on demand (POST /scan).
func (s *Server) handleRunScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg == nil || len(s.cfg.Targets) == 0 {
		http.Error(w, "no scan targets configured (start serve with --config)", http.StatusBadRequest)
		return
	}
	rep, err := runner.Run(r.Context(), s.cfg.Targets[0], s.store)
	if err != nil {
		s.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/scan/"+rep.ScanID, http.StatusSeeOther)
}

func (s *Server) handleAPIList(w http.ResponseWriter, r *http.Request) {
	reports, err := s.store.List()
	if err != nil {
		s.serverError(w, err)
		return
	}
	writeJSON(w, reports)
}

func (s *Server) handleAPIGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/scans/")
	rep, err := s.store.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, rep)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	s.log.Printf("server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// ListenAndServe starts the HTTP server and shuts it down when ctx is canceled.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{Addr: addr, Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	s.log.Printf("dashboard listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
