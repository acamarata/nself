// Package metrics provides a shared Prometheus registry and a universal
// /metrics endpoint factory for nSelf plugins. Every plugin exposes the same
// base set of counters + histograms so observability tooling (Prometheus,
// Grafana, the nSelf monitoring bundle) can ingest them without per-plugin
// glue.
package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry wraps a prometheus.Registry plus the universal per-plugin metrics.
type Registry struct {
	Plugin string
	Reg    *prometheus.Registry

	// Universal metrics — every plugin gets these for free.
	RequestsTotal     *prometheus.CounterVec   // by route, method, status
	RequestDuration   *prometheus.HistogramVec // by route, method
	InFlightRequests  prometheus.Gauge
	ErrorsTotal       *prometheus.CounterVec // by kind
	BuildInfo         *prometheus.GaugeVec   // labels: version, go_version

	// Consumers can register their own metrics via Reg.MustRegister().
}

var (
	defaultRegistry *Registry
	defaultMu       sync.Mutex
)

// NewRegistry builds a new metrics registry for a plugin. Each plugin should
// call this once at startup, then pass it to handlers that record metrics.
//
// The returned registry includes Go runtime collectors + process collectors so
// operators see GC pauses, memory, goroutines out of the box.
func NewRegistry(pluginName, version string) *Registry {
	r := prometheus.NewRegistry()

	r.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	labels := prometheus.Labels{"plugin": pluginName}

	reg := &Registry{
		Plugin: pluginName,
		Reg:    r,
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "nself_plugin_requests_total",
				Help:        "Total HTTP requests handled by the plugin.",
				ConstLabels: labels,
			},
			[]string{"route", "method", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "nself_plugin_request_duration_seconds",
				Help:        "HTTP request duration in seconds.",
				ConstLabels: labels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"route", "method"},
		),
		InFlightRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "nself_plugin_in_flight_requests",
				Help:        "Number of in-flight HTTP requests.",
				ConstLabels: labels,
			},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "nself_plugin_errors_total",
				Help:        "Total plugin errors by kind.",
				ConstLabels: labels,
			},
			[]string{"kind"},
		),
		BuildInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "nself_plugin_build_info",
				Help:        "Build info for the plugin. Always 1.",
				ConstLabels: labels,
			},
			[]string{"version"},
		),
	}

	r.MustRegister(
		reg.RequestsTotal,
		reg.RequestDuration,
		reg.InFlightRequests,
		reg.ErrorsTotal,
		reg.BuildInfo,
	)
	reg.BuildInfo.WithLabelValues(version).Set(1)

	return reg
}

// SetDefault sets the process-wide default registry. Useful for plugins that
// want a single package-level registry.
func SetDefault(r *Registry) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultRegistry = r
}

// Default returns the process-wide default registry or nil if none set.
func Default() *Registry {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	return defaultRegistry
}

// Handler returns an http.Handler that serves the /metrics endpoint for this
// registry. Mount under /metrics.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.Reg, promhttp.HandlerOpts{Registry: r.Reg})
}

// ObserveRequest records a completed HTTP request. route should be a stable
// string (e.g. "/v1/chat"), not the raw URL with path params, to avoid
// cardinality blowup.
func (r *Registry) ObserveRequest(route, method string, status int, duration time.Duration) {
	r.RequestsTotal.WithLabelValues(route, method, statusClass(status)).Inc()
	r.RequestDuration.WithLabelValues(route, method).Observe(duration.Seconds())
}

// IncError increments the errors_total counter by kind (e.g. "db", "upstream",
// "timeout", "auth").
func (r *Registry) IncError(kind string) {
	r.ErrorsTotal.WithLabelValues(kind).Inc()
}

// Middleware returns an http.Handler wrapper that records metrics for every
// request passing through. route is a fixed string; call from inside your
// router so each route binds its own wrapper.
func (r *Registry) Middleware(route string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			r.InFlightRequests.Inc()
			defer r.InFlightRequests.Dec()

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, req)

			r.ObserveRequest(route, req.Method, rec.status, time.Since(start))
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func statusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
