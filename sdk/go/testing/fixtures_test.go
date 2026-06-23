// Package testing_test exercises the shared test fixtures.
// Importing as sdktest avoids a conflict with stdlib "testing".
package testing_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	stdtesting "testing"
	"time"

	sdktest "github.com/nself-org/cli/sdk/go/v2/testing"
)

func TestStubUpstream(t *stdtesting.T) {
	srv := sdktest.StubUpstream(t, map[string]any{
		"/v1/models": []string{"claude", "gpt-4"},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

func TestStubUpstream_UnknownPath(t *stdtesting.T) {
	srv := sdktest.StubUpstream(t, map[string]any{
		"/v1/known": "ok",
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/unknown")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", resp.StatusCode)
	}
}

func TestDoJSONRequest(t *stdtesting.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"answer":42}`))
	})
	code, body := sdktest.DoJSONRequest(t, handler, "GET", "/v1/test", nil)
	if code != http.StatusOK {
		t.Errorf("want 200, got %d", code)
	}
	if body["answer"] != float64(42) {
		t.Errorf("want answer=42, got %v", body["answer"])
	}
}

func TestDoJSONRequest_WithBody(t *stdtesting.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	})
	code, _ := sdktest.DoJSONRequest(t, handler, "POST", "/v1/items", map[string]string{"name": "test"})
	if code != http.StatusCreated {
		t.Errorf("want 201, got %d", code)
	}
}

func TestWithTimeout(t *stdtesting.T) {
	ctx := sdktest.WithTimeout(t, 5*time.Second)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	select {
	case <-ctx.Done():
		t.Fatal("context should not be done immediately")
	default:
		// good
	}
}

func TestFixedClock(t *stdtesting.T) {
	fixed := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	clock := sdktest.FixedClock(fixed)
	if got := clock(); !got.Equal(fixed) {
		t.Errorf("want %v, got %v", fixed, got)
	}
}

func TestMustJSON(t *stdtesting.T) {
	b := sdktest.MustJSON(t, map[string]string{"key": "val"})
	if !strings.Contains(string(b), "val") {
		t.Errorf("want 'val' in JSON output, got %q", string(b))
	}
}

func TestTempCacheDir(t *stdtesting.T) {
	dir := sdktest.TempCacheDir(t)
	if dir == "" {
		t.Fatal("expected non-empty temp dir")
	}
}

func TestFetchMetrics(t *stdtesting.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# HELP nself_plugin_build_info\nnself_plugin_build_info 1\n"))
	})
	out := sdktest.FetchMetrics(t, handler)
	if !strings.Contains(out, "nself_plugin_build_info") {
		t.Errorf("expected metric in output, got: %q", out)
	}
}

func TestAssertMetricPresent(t *stdtesting.T) {
	expo := "nself_plugin_requests_total 5\n"
	// This should not cause a test failure since the metric is present.
	sdktest.AssertMetricPresent(t, expo, "nself_plugin_requests_total")
}

func TestAssertHealthEndpoints(t *stdtesting.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"1.0.0"}`))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# metrics\n"))
	})

	// Build a test server for the stub upstream helper.
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Also exercise AssertHealthEndpoints directly against the mux.
	sdktest.AssertHealthEndpoints(t, mux)
}
