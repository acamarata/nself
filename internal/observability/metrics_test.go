package observability

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry(RegistryConfig{
		Service: "test", Instance: "localhost", Env: "dev", Version: "1.0.0",
	})
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	// Should have process and go collectors registered
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatal("expected at least process/go collector metrics")
	}
}

func TestValidateLabels(t *testing.T) {
	if err := ValidateLabels([]string{"method", "route"}); err != nil {
		t.Errorf("valid labels rejected: %v", err)
	}
	if err := ValidateLabels([]string{"user_id"}); err == nil {
		t.Error("user_id should be forbidden")
	}
}

func TestMetricsHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter_total",
		Help: "A test counter.",
	})
	reg.MustRegister(c)
	c.Inc()

	handler := MetricsHandler(reg)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "test_counter_total") {
		t.Error("expected test_counter_total in metrics output")
	}
}

func TestHTTPMetricsMiddleware(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewHTTPMetrics(reg, "test")

	r := chi.NewRouter()
	r.Use(m.Middleware("test"))
	r.Get("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify metrics were recorded
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}

	metricNames := map[string]bool{}
	for _, mf := range mfs {
		metricNames[*mf.Name] = true
	}

	expected := []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_request_size_bytes",
		"http_response_size_bytes",
		"http_requests_in_flight",
	}
	for _, name := range expected {
		if !metricNames[name] {
			t.Errorf("missing metric: %s", name)
		}
	}
}

func TestAIMetricsRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewAIMetrics(reg)
	if m == nil {
		t.Fatal("NewAIMetrics returned nil")
	}

	// Record a sample
	m.CallsTotal.WithLabelValues("anthropic", "claude-4", "chat", "ok").Inc()
	m.CallDuration.WithLabelValues("anthropic", "claude-4", "chat").Observe(1.5)
	m.InputTokensTotal.WithLabelValues("anthropic", "claude-4").Add(1000)
	m.OutputTokensTotal.WithLabelValues("anthropic", "claude-4").Add(500)
	m.CostUSDTotal.WithLabelValues("anthropic", "claude-4").Add(0.05)
	m.ConcurrentCalls.WithLabelValues("anthropic").Set(3)

	// Record all 10 metrics so they appear in Gather
	m.RetriesTotal.WithLabelValues("anthropic", "claude-4", "timeout").Inc()
	m.RateLimitHits.WithLabelValues("anthropic").Inc()
	m.QualityScore.WithLabelValues("anthropic", "claude-4", "chat").Observe(0.9)
	m.ContextTokens.WithLabelValues("anthropic", "claude-4").Observe(4096)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(mfs) < 10 {
		t.Errorf("expected at least 10 metric families, got %d", len(mfs))
	}
}

func TestDBMetricsRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDBMetrics(reg)
	m.QueriesTotal.WithLabelValues("claw", "SELECT", "topics", "ok").Inc()
	m.QueryDuration.WithLabelValues("claw", "SELECT", "topics", "ok").Observe(0.05)
	m.PoolInUse.WithLabelValues("claw").Set(5)
	m.PoolIdle.WithLabelValues("claw").Set(3)
	m.PoolMax.WithLabelValues("claw").Set(20)
	m.ConnWait.WithLabelValues("claw").Observe(0.001)
	m.DeadlocksTotal.WithLabelValues("claw").Inc()

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(mfs) < 7 {
		t.Errorf("expected at least 7 DB metric families, got %d", len(mfs))
	}
}
