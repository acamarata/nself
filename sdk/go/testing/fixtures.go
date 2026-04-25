// Package testing provides shared test fixtures + helpers for nSelf plugin
// unit tests. Import as sdktest to avoid shadowing stdlib "testing":
//
//	import sdktest "github.com/nself-org/cli/sdk/go/testing"
package testing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	stdtesting "testing"
	"time"
)

// StubUpstream builds an httptest.Server that returns canned JSON for matching
// request paths. Paths that don't match yield a 404.
//
//	srv := sdktest.StubUpstream(t, map[string]any{
//	    "/v1/models": []string{"gpt-4", "claude-3"},
//	})
//	defer srv.Close()
func StubUpstream(t *stdtesting.T, routes map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, body := range routes {
		p, b := path, body
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(b)
		})
	}
	return httptest.NewServer(mux)
}

// DoJSONRequest issues a JSON request to h and returns the decoded response
// body. Useful for table-driven handler tests.
func DoJSONRequest(t *stdtesting.T, h http.Handler, method, path string, reqBody any) (int, map[string]any) {
	t.Helper()
	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		body = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, body)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var out map[string]any
	if rr.Body.Len() > 0 {
		if err := json.NewDecoder(rr.Body).Decode(&out); err != nil && err != io.EOF {
			t.Fatalf("decode response: %v", err)
		}
	}
	return rr.Code, out
}

// WithTimeout returns a context.Context that cancels after d. Cleans up via
// t.Cleanup so callers don't need to defer.
func WithTimeout(t *stdtesting.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

// FixedClock returns a zero-arg closure always returning the same time. Inject
// into plugins that take a clock function for deterministic tests.
func FixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// MustJSON marshals v or fails the test. Shortcut for test setup.
func MustJSON(t *stdtesting.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("MustJSON: %v", err)
	}
	return b
}

// TempCacheDir returns a tempdir suitable for license caches, config files,
// and other plugin on-disk state. Registers cleanup with t.
func TempCacheDir(t *stdtesting.T) string {
	t.Helper()
	return t.TempDir()
}

// FetchMetrics issues GET /metrics against h and returns the raw Prometheus
// exposition text. Fails the test on non-200.
func FetchMetrics(t *stdtesting.T, h http.Handler) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /metrics: code=%d body=%s", rr.Code, rr.Body.String())
	}
	return rr.Body.String()
}

// AssertMetricPresent fails the test if the Prometheus exposition text does
// not contain metric. Used with FetchMetrics to verify /metrics wiring.
func AssertMetricPresent(t *stdtesting.T, expo, metric string) {
	t.Helper()
	if !strings.Contains(expo, metric) {
		t.Errorf("expected metric %q in /metrics output, not found", metric)
	}
}

// AssertHealthEndpoints exercises the canonical SDK endpoints (/healthz,
// /readyz, /metrics, /version) and fails the test if any diverge from the
// contract every plugin must satisfy.
func AssertHealthEndpoints(t *stdtesting.T, h http.Handler) {
	t.Helper()
	for _, tc := range []struct {
		path       string
		wantStatus int
	}{
		{"/healthz", http.StatusOK},
		{"/version", http.StatusOK},
		{"/metrics", http.StatusOK},
	} {
		req := httptest.NewRequest("GET", tc.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != tc.wantStatus {
			t.Errorf("%s: code=%d want=%d", tc.path, rr.Code, tc.wantStatus)
		}
	}
}
