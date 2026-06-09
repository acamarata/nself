package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Integration tests: HasuraHealthzHandler — fast / slow / down
//
// Each test spins up a mock "Hasura" HTTP server that simulates one of the
// three scenarios, then calls the handler under test and asserts the HTTP
// status and JSON body match the acceptance criteria.
// ---------------------------------------------------------------------------

// newMockHasura returns a test server whose /healthz handler behaves as
// described by the given function. The caller is responsible for calling
// ts.Close().
func newMockHasura(delay time.Duration, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			time.Sleep(delay)
		}
		w.WriteHeader(statusCode)
	}))
}

// callHealthz creates a hasura_healthz handler backed by the given Hasura URL
// and threshold, fires a test HTTP request, and returns the recorder.
func callHealthz(hasuraURL string, thresholdMS int) *httptest.ResponseRecorder {
	cfg := HasuraHealthzConfig{
		HasuraURL: hasuraURL + "/healthz",
		TimeoutMS: thresholdMS,
	}
	handler := HasuraHealthzHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// decodeBody unmarshals the recorder body into a hasuraHealthzBody.
func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) hasuraHealthzBody {
	t.Helper()
	var body hasuraHealthzBody
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}

// TestHasuraHealthzHandler_Ok verifies: Hasura responds in 100ms with HTTP 200
// → /healthz returns 200 {"status":"ok","hasura":"ok"}.
//
// Acceptance: "Integration test: mock Hasura responds in 100ms → 200 ok"
func TestHasuraHealthzHandler_Ok(t *testing.T) {
	ts := newMockHasura(100*time.Millisecond, http.StatusOK)
	defer ts.Close()

	rr := callHealthz(ts.URL, 2000) // 2000ms threshold — Hasura well within it

	if rr.Code != http.StatusOK {
		t.Errorf("expected HTTP 200, got %d", rr.Code)
	}
	body := decodeBody(t, rr)
	if body.Status != "ok" {
		t.Errorf("expected status %q, got %q", "ok", body.Status)
	}
	if body.Hasura != "ok" {
		t.Errorf("expected hasura %q, got %q", "ok", body.Hasura)
	}
}

// TestHasuraHealthzHandler_Degraded verifies: Hasura sleeps 5s but threshold
// is 300ms → /healthz returns 200 {"status":"degraded","hasura":"slow"}.
//
// Acceptance: "Integration test: mock Hasura sleeps 5s → 200 degraded"
func TestHasuraHealthzHandler_Degraded(t *testing.T) {
	ts := newMockHasura(5*time.Second, http.StatusOK)
	defer ts.Close()

	// Use a 300ms threshold so the test doesn't take 5s.
	rr := callHealthz(ts.URL, 300)

	if rr.Code != http.StatusOK {
		t.Errorf("expected HTTP 200 (degraded), got %d", rr.Code)
	}
	body := decodeBody(t, rr)
	if body.Status != "degraded" {
		t.Errorf("expected status %q, got %q", "degraded", body.Status)
	}
	if body.Hasura != "slow" {
		t.Errorf("expected hasura %q, got %q", "slow", body.Hasura)
	}
}

// TestHasuraHealthzHandler_Down verifies: Hasura server is closed (connection
// refused) → /healthz returns 503 {"status":"down","hasura":"down"}.
//
// Acceptance: "Integration test: mock Hasura connection refused → 503 down"
func TestHasuraHealthzHandler_Down(t *testing.T) {
	// Start a server then immediately close it to get a connection-refused target.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	downURL := ts.URL
	ts.Close() // closed — any connection attempt will be refused

	rr := callHealthz(downURL, 2000)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected HTTP 503, got %d", rr.Code)
	}
	body := decodeBody(t, rr)
	if body.Status != "down" {
		t.Errorf("expected status %q, got %q", "down", body.Status)
	}
	if body.Hasura != "down" {
		t.Errorf("expected hasura %q, got %q", "down", body.Hasura)
	}
}

// TestHasuraHealthzConfigFromEnv_Defaults verifies that HasuraHealthzConfigFromEnv
// returns 2000ms and the default Hasura URL when no env vars are set.
func TestHasuraHealthzConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv("HEALTHZ_HASURA_TIMEOUT_MS", "")
	t.Setenv("HEALTHZ_HASURA_URL", "")

	cfg := HasuraHealthzConfigFromEnv()
	if cfg.TimeoutMS != defaultHasuraTimeoutMS {
		t.Errorf("expected default timeout %d ms, got %d", defaultHasuraTimeoutMS, cfg.TimeoutMS)
	}
	if cfg.HasuraURL != defaultHasuraURL {
		t.Errorf("expected default URL %q, got %q", defaultHasuraURL, cfg.HasuraURL)
	}
}

// TestHasuraHealthzConfigFromEnv_CustomTimeout verifies that a valid
// HEALTHZ_HASURA_TIMEOUT_MS env var overrides the default.
func TestHasuraHealthzConfigFromEnv_CustomTimeout(t *testing.T) {
	t.Setenv("HEALTHZ_HASURA_TIMEOUT_MS", "500")
	t.Setenv("HEALTHZ_HASURA_URL", "")

	cfg := HasuraHealthzConfigFromEnv()
	if cfg.TimeoutMS != 500 {
		t.Errorf("expected timeout 500 ms, got %d", cfg.TimeoutMS)
	}
}

// TestHasuraHealthzConfigFromEnv_InvalidFallsToDefault verifies that an
// out-of-range HEALTHZ_HASURA_TIMEOUT_MS falls back to the 2000ms default
// rather than panicking or using a bad value.
func TestHasuraHealthzConfigFromEnv_InvalidFallsToDefault(t *testing.T) {
	for _, bad := range []string{"0", "-1", "999999", "abc"} {
		t.Run(bad, func(t *testing.T) {
			t.Setenv("HEALTHZ_HASURA_TIMEOUT_MS", bad)
			t.Setenv("HEALTHZ_HASURA_URL", "")

			cfg := HasuraHealthzConfigFromEnv()
			if cfg.TimeoutMS != defaultHasuraTimeoutMS {
				t.Errorf("invalid value %q: expected fallback to %d, got %d",
					bad, defaultHasuraTimeoutMS, cfg.TimeoutMS)
			}
		})
	}
}
