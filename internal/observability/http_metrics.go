package observability

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics holds the five standard HTTP metrics per spec §1.2.
type HTTPMetrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestSize      *prometheus.HistogramVec
	ResponseSize     *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge

	inFlight int64 // atomic, mirrored to gauge
}

// NewHTTPMetrics creates and registers the five HTTP metrics on the given registry.
// service is the service name label value (e.g. "claw", "ai", "mux").
func NewHTTPMetrics(reg *prometheus.Registry, service string) *HTTPMetrics {
	m := &HTTPMetrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests.",
			},
			[]string{"method", "route", "status", "service"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: DurationBuckets,
			},
			[]string{"method", "route", "status", "service"},
		),
		RequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request body size in bytes.",
				Buckets: SizeBuckets,
			},
			[]string{"method", "route"},
		),
		ResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response body size in bytes.",
				Buckets: SizeBuckets,
			},
			[]string{"method", "route", "status"},
		),
		RequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "http_requests_in_flight",
				Help:        "Current in-flight HTTP requests.",
				ConstLabels: prometheus.Labels{"service": service},
			},
		),
	}

	reg.MustRegister(m.RequestsTotal, m.RequestDuration, m.RequestSize, m.ResponseSize, m.RequestsInFlight)
	return m
}

// Middleware returns a chi-compatible middleware that records all five HTTP metrics.
// Route labels use chi's route pattern (template), never raw URLs.
func (m *HTTPMetrics) Middleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Track in-flight
			atomic.AddInt64(&m.inFlight, 1)
			m.RequestsInFlight.Set(float64(atomic.LoadInt64(&m.inFlight)))
			defer func() {
				atomic.AddInt64(&m.inFlight, -1)
				m.RequestsInFlight.Set(float64(atomic.LoadInt64(&m.inFlight)))
			}()

			start := time.Now()
			sw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			// Normalize route: use chi route pattern, fall back to path
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			status := strconv.Itoa(sw.status)
			method := r.Method
			duration := time.Since(start).Seconds()

			m.RequestsTotal.WithLabelValues(method, route, status, service).Inc()
			m.RequestDuration.WithLabelValues(method, route, status, service).Observe(duration)
			m.RequestSize.WithLabelValues(method, route).Observe(float64(r.ContentLength))
			m.ResponseSize.WithLabelValues(method, route, status).Observe(float64(sw.bytesWritten))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}
