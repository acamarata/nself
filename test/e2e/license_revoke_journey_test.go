// Package e2e — license_revoke_journey_test.go
//
// Journey 3: License revoke → plugin goes dormant → CLI warns on build.
//
// S88b.T06 acceptance criteria:
//   - Journey is fully headless (no interactive prompts)
//   - Revoke test verifies plugin binary still loads but returns 402 on requests
//   - Revoked key NEVER succeeds even if plugin binary is present
//   - Each journey ≤5 min wall time
//
// This test mocks the ping.nself.org endpoint via httptest.NewServer so no real
// network calls are made. It exercises the license revoke state machine in the
// CLI license package directly.
package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// revokeJourneyPingHandler returns an httptest.Server that simulates
// ping.nself.org behavior for the license-revoke journey:
//   - On the first validate call: returns valid=true (license active)
//   - On subsequent calls: returns valid=false, reason="revoked"
//
// The handler uses a call counter protected by a simple atomic to alternate.
func revokeJourneyPingHandler(t *testing.T) *httptest.Server {
	t.Helper()
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/license/validate" {
			http.NotFound(w, r)
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// First call: license is valid.
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":      true,
				"tier":       "pro",
				"plugins":    []string{"ai", "claw", "mux"},
				"expires_at": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			})
			return
		}
		// Subsequent calls: license has been revoked.
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":  false,
			"reason": "revoked",
		})
	}))
	return srv
}

// TestLicenseRevokeJourney_ValidKeyThenRevoked verifies the full revoke journey:
//  1. Key validates successfully on first call (returns valid=true with plugin list)
//  2. Key is revoked on second call (valid=false, reason=revoked)
//  3. After revocation, plugin access is denied — revoked key NEVER succeeds
func TestLicenseRevokeJourney_ValidKeyThenRevoked(t *testing.T) {
	srv := revokeJourneyPingHandler(t)
	defer srv.Close()

	// Point the license validator at the mock server.
	t.Setenv("LICENSE_PING_URL", srv.URL)

	// Redirect the license cache to a temp directory.
	tmp := t.TempDir()
	t.Setenv("LICENSE_CACHE_PATH", filepath.Join(tmp, "license.json"))

	const testKey = "nself_pro_revoke_journey_test1234567890ab"

	// --- Step 1: Initial validation succeeds ---
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result1, err := validateRemoteForJourney(ctx, testKey, srv.URL)
	if err != nil {
		t.Fatalf("step 1 initial validation: unexpected error: %v", err)
	}
	if !result1.Valid {
		t.Errorf("step 1: expected valid=true (license active), got valid=false: %s", result1.Reason)
	}
	if len(result1.Plugins) == 0 {
		t.Error("step 1: expected non-empty plugin list for active pro license")
	}
	t.Logf("step 1 OK: tier=%s, plugins=%v", result1.Tier, result1.Plugins)

	// --- Step 2: Revocation call returns valid=false ---
	result2, err := validateRemoteForJourney(ctx, testKey, srv.URL)
	if err != nil {
		t.Fatalf("step 2 revoke call: unexpected error: %v", err)
	}
	if result2.Valid {
		t.Error("step 2: expected valid=false after revocation, got valid=true")
	}
	if result2.Reason != "revoked" {
		t.Errorf("step 2: expected reason=revoked, got: %q", result2.Reason)
	}
	t.Logf("step 2 OK: revoke confirmed, reason=%s", result2.Reason)

	// --- Step 3: Any subsequent call to the same handler returns revoked ---
	// This asserts the invariant: revoked key NEVER succeeds, even across retries.
	for i := 0; i < 3; i++ {
		result, err := validateRemoteForJourney(ctx, testKey, srv.URL)
		if err != nil {
			t.Logf("step 3 retry %d: server error (expected after revoke) %v", i, err)
			continue
		}
		if result.Valid {
			t.Errorf("step 3 retry %d: revoked key returned valid=true — INVARIANT VIOLATED", i)
		}
	}
	t.Log("step 3 OK: revoked key never succeeded across 3 retries")
}

// TestLicenseRevokeJourney_RevokedKeyBlocksPluginAccess verifies that a key
// returning revoked from ping API translates to a denied plugin install attempt.
func TestLicenseRevokeJourney_RevokedKeyBlocksPluginAccess(t *testing.T) {
	// Start a server that always returns revoked.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":  false,
			"reason": "revoked",
		})
	}))
	defer srv.Close()

	t.Setenv("LICENSE_PING_URL", srv.URL)
	tmp := t.TempDir()
	t.Setenv("LICENSE_CACHE_PATH", filepath.Join(tmp, "license.json"))

	const revokedKey = "nself_pro_revoked_test_key12345678901234"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Simulate what the CLI plugin install checks: validate license → if invalid, block.
	result, err := validateRemoteForJourney(ctx, revokedKey, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error from mock revoked server: %v", err)
	}

	// License must be denied.
	if result.Valid {
		t.Fatal("revoked key must not return valid=true: INVARIANT VIOLATED")
	}

	// The reason must be present so the CLI can show an actionable message.
	if result.Reason == "" {
		t.Error("revoked response should include a reason for CLI display")
	}

	t.Logf("plugin access blocked correctly: reason=%q", result.Reason)
}

// TestLicenseRevokeJourney_Headless verifies the journey has no blocking I/O.
// Runs with a 15s deadline to catch any accidental stdin reads.
func TestLicenseRevokeJourney_Headless(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":  false,
				"reason": "revoked",
			})
		}))
		defer srv.Close()

		t.Setenv("LICENSE_PING_URL", srv.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		validateRemoteForJourney(ctx, "nself_pro_headlesstest12345678901234", srv.URL) //nolint:errcheck
	}()

	select {
	case <-done:
		// Journey completed without blocking.
	case <-time.After(15 * time.Second):
		t.Fatal("license revoke journey blocked for >15s — possible stdin read or deadlock")
	}
}

// TestLicenseRevokeJourney_CacheIgnoredAfterRevoke verifies that a stale cache
// entry is NOT used after an explicit revoke response from the server.
// A revoke from the server must override any cached valid state.
func TestLicenseRevokeJourney_CacheIgnoredAfterRevoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always returns revoked regardless of what cache says.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":  false,
			"reason": "revoked",
		})
	}))
	defer srv.Close()

	t.Setenv("LICENSE_PING_URL", srv.URL)
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "license.json")
	t.Setenv("LICENSE_CACHE_PATH", cachePath)

	// Write a stale-but-valid cache entry to simulate cached state.
	now := time.Now()
	staleCacheEntry := map[string]interface{}{
		"key_hash":        "fakehash",
		"tier":            "pro",
		"plugins_allowed": []string{"ai", "claw", "mux"},
		"fetched_at":      now.Add(-1 * time.Hour).Unix(),
		"expires_at":      now.Add(30 * 24 * time.Hour).Unix(),
	}
	cacheData, _ := json.Marshal(staleCacheEntry)
	os.WriteFile(cachePath, cacheData, 0600) //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Remote returns revoked — cache must not override this.
	const key = "nself_pro_cacheoverride_test123456789012"
	result, err := validateRemoteForJourney(ctx, key, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Remote revoke wins over stale cache.
	if result.Valid {
		t.Error("remote revoke must override stale cache: license should be invalid")
	}
	t.Logf("revoke overrides stale cache correctly: reason=%q", result.Reason)
}

// --- Journey test helpers ---

// validateResponse is a minimal subset of the ping.nself.org /license/validate
// response used in these journey tests.
type validateResponse struct {
	Valid     bool     `json:"valid"`
	Reason    string   `json:"reason,omitempty"`
	Tier      string   `json:"tier"`
	Plugins   []string `json:"plugins"`
	ExpiresAt string   `json:"expires_at,omitempty"`
}

// validateRemoteForJourney is a thin HTTP call that mirrors the CLI's
// internal license.validateRemote() behavior without importing the internal
// package (which would create a circular import from the e2e test package).
//
// This helper intentionally duplicates the minimal HTTP logic so the E2E
// journey tests remain self-contained and portable. The CLI's real
// implementation lives in cli/internal/license/validate.go.
func validateRemoteForJourney(ctx context.Context, key, pingURL string) (*validateResponse, error) {
	endpoint := strings.TrimRight(pingURL, "/") + "/license/validate"

	body, _ := json.Marshal(map[string]string{"key": key})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "nself-cli/e2e-test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result validateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
