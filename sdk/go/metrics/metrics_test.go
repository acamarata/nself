package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewRegistryBuildInfo(t *testing.T) {
	r := NewRegistry("ai", "1.2.3")
	ts := httptest.NewServer(r.Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	out := string(body)

	if !strings.Contains(out, `nself_plugin_build_info{plugin="ai",version="1.2.3"} 1`) {
		t.Errorf("expected build_info line, got: %s", out)
	}
}

func TestObserveRequest(t *testing.T) {
	r := NewRegistry("mux", "0.1.0")
	r.ObserveRequest("/v1/ping", "GET", 200, 15*time.Millisecond)
	r.ObserveRequest("/v1/ping", "GET", 500, 30*time.Millisecond)
	r.IncError("upstream")

	ts := httptest.NewServer(r.Handler())
	t.Cleanup(ts.Close)
	resp, _ := http.Get(ts.URL)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	out := string(body)

	if !strings.Contains(out, `nself_plugin_requests_total{method="GET",plugin="mux",route="/v1/ping",status="2xx"} 1`) {
		t.Errorf("expected 2xx counter, got: %s", out)
	}
	if !strings.Contains(out, `status="5xx"`) {
		t.Errorf("expected 5xx counter, got: %s", out)
	}
	if !strings.Contains(out, `nself_plugin_errors_total{kind="upstream",plugin="mux"} 1`) {
		t.Errorf("expected errors_total, got: %s", out)
	}
}

func TestMiddleware(t *testing.T) {
	r := NewRegistry("p", "v")
	mw := r.Middleware("/test")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(201)
	}))
	req := httptest.NewRequest("POST", "/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Errorf("status got %d", rec.Code)
	}
}
