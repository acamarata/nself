// Package server builds a chi-based HTTP server with the nSelf plugin
// default middleware stack: metrics, recovery, request ID, readiness / liveness.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/nself-org/cli/sdk/go/metrics"
)

// HealthChecker is any type that can report readiness. Plugins typically pass
// a closure that checks DB + upstream APIs.
type HealthChecker interface {
	Ready(ctx context.Context) error
}

// Options configures the default server.
type Options struct {
	Plugin    string
	Version   string
	Metrics   *metrics.Registry // optional; if nil, a new one is created
	Ready     HealthChecker     // optional; used by /readyz
	Timeout   time.Duration     // request timeout; default 30s
	Routes    func(r chi.Router, m *metrics.Registry)
}

// New returns a *chi.Mux with the default nSelf plugin stack mounted:
//
//	GET /healthz  - liveness (always 200 once the server is up)
//	GET /readyz   - readiness (delegates to Options.Ready)
//	GET /metrics  - Prometheus metrics
//	GET /version  - plugin version info
//
// Consumers supply Options.Routes to register their own handlers under /v1 etc.
func New(opts Options) *chi.Mux {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Metrics == nil {
		opts.Metrics = metrics.NewRegistry(opts.Plugin, opts.Version)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(opts.Timeout))

	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"plugin":  opts.Plugin,
			"version": opts.Version,
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
		defer cancel()
		if opts.Ready != nil {
			if err := opts.Ready.Ready(ctx); err != nil {
				opts.Metrics.IncError("not_ready")
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"status": "not_ready",
					"error":  err.Error(),
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
	})

	r.Handle("/metrics", opts.Metrics.Handler())

	r.Get("/version", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"plugin":  opts.Plugin,
			"version": opts.Version,
			"sdk":     "plugin-sdk-go",
		})
	})

	if opts.Routes != nil {
		opts.Routes(r, opts.Metrics)
	}

	return r
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
