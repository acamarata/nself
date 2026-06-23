// Package server_test covers the HTTP server factory for nSelf plugins.
package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nself-org/cli/sdk/go/v2/metrics"
	"github.com/nself-org/cli/sdk/go/v2/server"
)

// readyAlways is a HealthChecker that always returns nil.
type readyAlways struct{}

func (r readyAlways) Ready(_ context.Context) error { return nil }

// readyError is a HealthChecker that always returns an error.
type readyError struct{ msg string }

func (r readyError) Ready(_ context.Context) error { return errors.New(r.msg) }

// newTestServer builds a server.New mux with optional overrides and returns it.
func newTestServer(t *testing.T, opts server.Options) http.Handler {
	t.Helper()
	return server.New(opts)
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return out
}

func TestNew_Healthz(t *testing.T) {
	h := newTestServer(t, server.Options{Plugin: "test-plugin", Version: "1.0.0"})
	rr := get(t, h, "/healthz")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	body := decodeJSON(t, rr)
	if body["status"] != "ok" {
		t.Errorf("want status=ok, got %v", body["status"])
	}
	if body["plugin"] != "test-plugin" {
		t.Errorf("want plugin=test-plugin, got %v", body["plugin"])
	}
	if body["version"] != "1.0.0" {
		t.Errorf("want version=1.0.0, got %v", body["version"])
	}
}

func TestNew_Readyz_OK(t *testing.T) {
	h := newTestServer(t, server.Options{
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Ready:   readyAlways{},
	})
	rr := get(t, h, "/readyz")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	body := decodeJSON(t, rr)
	if body["status"] != "ready" {
		t.Errorf("want status=ready, got %v", body["status"])
	}
}

func TestNew_Readyz_NoChecker(t *testing.T) {
	// When Ready is nil, /readyz should return 200 ready.
	h := newTestServer(t, server.Options{Plugin: "test-plugin", Version: "1.0.0"})
	rr := get(t, h, "/readyz")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestNew_Readyz_Error(t *testing.T) {
	h := newTestServer(t, server.Options{
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Ready:   readyError{msg: "db unavailable"},
	})
	rr := get(t, h, "/readyz")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rr.Code)
	}
	body := decodeJSON(t, rr)
	if body["status"] != "not_ready" {
		t.Errorf("want status=not_ready, got %v", body["status"])
	}
	if !strings.Contains(body["error"].(string), "db unavailable") {
		t.Errorf("want error to contain 'db unavailable', got %v", body["error"])
	}
}

func TestNew_Version(t *testing.T) {
	h := newTestServer(t, server.Options{Plugin: "my-plugin", Version: "2.3.4"})
	rr := get(t, h, "/version")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	body := decodeJSON(t, rr)
	if body["plugin"] != "my-plugin" {
		t.Errorf("want plugin=my-plugin, got %v", body["plugin"])
	}
	if body["version"] != "2.3.4" {
		t.Errorf("want version=2.3.4, got %v", body["version"])
	}
	if body["sdk"] != "plugin-sdk-go" {
		t.Errorf("want sdk=plugin-sdk-go, got %v", body["sdk"])
	}
}

func TestNew_CustomRoutes(t *testing.T) {
	h := newTestServer(t, server.Options{
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Routes: func(r chi.Router, _ *metrics.Registry) {
			r.Get("/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"pong":true}`))
			})
		},
	})
	rr := get(t, h, "/v1/ping")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestNew_MetricsEndpoint(t *testing.T) {
	h := newTestServer(t, server.Options{Plugin: "test-plugin", Version: "1.0.0"})
	rr := get(t, h, "/metrics")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("want text/plain content type, got %q", ct)
	}
}

func TestNew_ExistingMetrics(t *testing.T) {
	reg := metrics.NewRegistry("test-plugin", "1.0.0")
	h := newTestServer(t, server.Options{
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Metrics: reg,
	})
	rr := get(t, h, "/healthz")
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}
