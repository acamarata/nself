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
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
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

func TestSetDefaultAndDefault(t *testing.T) {
	// Reset to a known state by calling Default before any set.
	r := NewRegistry("def-plugin", "1.0.0")
	SetDefault(r)

	got := Default()
	if got == nil {
		t.Fatal("Default() returned nil after SetDefault")
	}
	if got.Plugin != "def-plugin" {
		t.Errorf("want Plugin=def-plugin, got %q", got.Plugin)
	}
}

func TestStatusClassAllBranches(t *testing.T) {
	cases := []struct {
		code  int
		class string
	}{
		{100, "1xx"},
		{200, "2xx"},
		{201, "2xx"},
		{301, "3xx"},
		{404, "4xx"},
		{500, "5xx"},
	}
	for _, c := range cases {
		r := NewRegistry("sc", "1.0.0")
		r.ObserveRequest("/path", "GET", c.code, time.Millisecond)
		ts := httptest.NewServer(r.Handler())
		resp, err := http.Get(ts.URL)
		ts.Close()
		if err != nil {
			t.Fatalf("code %d: GET /metrics: %v", c.code, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), `status="`+c.class+`"`) {
			t.Errorf("code %d: want status=%q in metrics output", c.code, c.class)
		}
	}
}
