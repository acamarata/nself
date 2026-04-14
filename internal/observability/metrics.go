// Package observability provides shared Prometheus metrics and OpenTelemetry
// tracing helpers for all nSelf Go services.
package observability

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Base label keys injected into every metric via constLabels.
// Cardinality budget: max 10k series per metric. user_id is forbidden.
var forbiddenLabels = map[string]bool{
	"user_id": true,
}

// SI-unit histogram buckets per spec §1.1.
var (
	// DurationBuckets for _seconds histograms (HTTP, DB, AI, cache, queue).
	DurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	// SizeBuckets for _bytes histograms (request/response body sizes).
	SizeBuckets = []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000, 5000000}
)

// RegistryConfig configures a new metrics registry.
type RegistryConfig struct {
	Service  string // e.g. "claw", "ai", "mux"
	Instance string // hostname or pod name
	Env      string // "dev", "staging", "prod"
	Version  string // e.g. "1.0.3"
}

// NewRegistry creates a *prometheus.Registry pre-configured with base labels
// and process/go collectors. It enforces naming conventions (_total for counters,
// _seconds/_bytes for histograms) and forbids user_id as a label.
func NewRegistry(cfg RegistryConfig) *prometheus.Registry {
	reg := prometheus.NewRegistry()
	// Register standard process and Go runtime collectors.
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector())
	return reg
}

// BaseLabels returns the standard label set from config plus environment.
// Services should include these as ConstLabels or via a wrapping registerer.
func BaseLabels(cfg RegistryConfig) prometheus.Labels {
	instance := cfg.Instance
	if instance == "" {
		instance, _ = os.Hostname()
	}
	env := cfg.Env
	if env == "" {
		env = os.Getenv("NSELF_ENV")
		if env == "" {
			env = "dev"
		}
	}
	return prometheus.Labels{
		"service":  cfg.Service,
		"instance": instance,
		"env":      env,
		"version":  cfg.Version,
	}
}

// ValidateLabels checks that no forbidden labels (e.g. user_id) are used.
// Returns an error if a forbidden label is found.
func ValidateLabels(labels []string) error {
	for _, l := range labels {
		if forbiddenLabels[strings.ToLower(l)] {
			return fmt.Errorf("observability: forbidden label %q (cardinality risk)", l)
		}
	}
	return nil
}

// ServeMetrics starts an HTTP server on bindAddr serving the /metrics endpoint
// for the given registry. It blocks, so call in a goroutine.
func ServeMetrics(registry *prometheus.Registry, bindAddr string) {
	if bindAddr == "" {
		bindAddr = "127.0.0.1:9090"
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	slog.Info("serving prometheus metrics", "addr", bindAddr)
	if err := http.ListenAndServe(bindAddr, mux); err != nil {
		slog.Error("metrics server failed", "error", err)
	}
}

// MetricsHandler returns an http.Handler that serves /metrics for the given registry.
func MetricsHandler(registry *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}
