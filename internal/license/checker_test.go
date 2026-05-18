// Package license — checker_test.go covers BundleEntitled, bundleEntitledFromCache,
// and CollectLicenseKey. These three functions were at 0% before this file.
package license

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---- helpers ----------------------------------------------------------------

// checkerCacheDir points LICENSE_CACHE_PATH at a temp file for the duration of t.
func checkerCacheDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := fmt.Sprintf("%s/license.json", dir)
	t.Setenv("LICENSE_CACHE_PATH", path)
}

// checkerWriteCache writes a CacheEntry for the given key/tier/plugins to the cache.
func checkerWriteCache(t *testing.T, key, tier string, plugins []string) {
	t.Helper()
	entry := &CacheEntry{
		KeyHash:        HashKey(key),
		Tier:           tier,
		PluginsAllowed: plugins,
		FetchedAt:      1700000000,
		ExpiresAt:      9999999999,
	}
	if err := WriteCache(entry); err != nil {
		t.Fatalf("checkerWriteCache: %v", err)
	}
}

// bundleServer sets up an httptest server that responds to every request with
// the given status + bundleValidateResponse JSON body.
func newBundleServer(t *testing.T, status int, body bundleValidateResponse) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LICENSE_PING_URL", srv.URL)
}

// ---- BundleEntitled ---------------------------------------------------------

func TestBundleEntitled_EmptyKey(t *testing.T) {
	checkerCacheDir(t)
	ok, err := BundleEntitled(context.Background(), "", "nclaw")
	if ok {
		t.Error("expected false for empty key, got true")
	}
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
}

func TestBundleEntitled_ServerValid(t *testing.T) {
	checkerCacheDir(t)
	newBundleServer(t, http.StatusOK, bundleValidateResponse{
		Valid:   true,
		Tier:    "plus",
		Bundle:  "nclaw",
		Plugins: []string{"claw", "ai", "mux"},
	})

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for valid bundle, got false")
	}
}

func TestBundleEntitled_ServerInvalid(t *testing.T) {
	checkerCacheDir(t)
	newBundleServer(t, http.StatusOK, bundleValidateResponse{
		Valid:  false,
		Tier:   "free",
		Reason: "not entitled to this bundle",
	})

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if ok {
		t.Error("expected false for invalid bundle, got true")
	}
	if err == nil {
		t.Error("expected error describing denial, got nil")
	}
}

func TestBundleEntitled_ServerUnauthorized(t *testing.T) {
	checkerCacheDir(t)
	newBundleServer(t, http.StatusUnauthorized, bundleValidateResponse{})

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if ok {
		t.Error("expected false for 401, got true")
	}
	if err == nil {
		t.Error("expected error for 401, got nil")
	}
}

func TestBundleEntitled_ServerForbidden(t *testing.T) {
	checkerCacheDir(t)
	newBundleServer(t, http.StatusForbidden, bundleValidateResponse{})

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if ok {
		t.Error("expected false for 403, got true")
	}
	if err == nil {
		t.Error("expected error for 403, got nil")
	}
}

func TestBundleEntitled_ServerUnexpectedStatus(t *testing.T) {
	checkerCacheDir(t)
	newBundleServer(t, http.StatusInternalServerError, bundleValidateResponse{})

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if ok {
		t.Error("expected false for 500, got true")
	}
	if err == nil {
		t.Error("expected error for 500, got nil")
	}
}

func TestBundleEntitled_NetworkErrorFailClosed(t *testing.T) {
	checkerCacheDir(t)
	// Unreachable port — no server listening.
	t.Setenv("LICENSE_PING_URL", "http://127.0.0.1:19999")
	t.Setenv("NSELF_LICENSE_FAIL_OPEN", "")

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if ok {
		t.Error("expected false on network error (fail-closed), got true")
	}
	if err == nil {
		t.Error("expected error on network failure, got nil")
	}
}

func TestBundleEntitled_NetworkErrorFailOpen_PlusTier(t *testing.T) {
	checkerCacheDir(t)
	key := "nself_pro_abcdefghijklmnopqrstuvwxyz123456"
	checkerWriteCache(t, key, "plus", nil)

	t.Setenv("LICENSE_PING_URL", "http://127.0.0.1:19999")
	t.Setenv("NSELF_LICENSE_FAIL_OPEN", "1")
	defer t.Setenv("NSELF_LICENSE_FAIL_OPEN", "")

	ok, err := BundleEntitled(context.Background(), key, "nclaw")
	if err != nil {
		t.Fatalf("unexpected error with fail-open + plus tier: %v", err)
	}
	if !ok {
		t.Error("expected true for plus tier via fail-open, got false")
	}
}

func TestBundleEntitled_NetworkErrorFailOpen_NoCacheAvailable(t *testing.T) {
	checkerCacheDir(t)
	t.Setenv("LICENSE_PING_URL", "http://127.0.0.1:19999")
	t.Setenv("NSELF_LICENSE_FAIL_OPEN", "1")
	defer t.Setenv("NSELF_LICENSE_FAIL_OPEN", "")

	// No cache file written — should return false + error.
	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if ok {
		t.Error("expected false with no cache in fail-open, got true")
	}
	if err == nil {
		t.Error("expected error with no cache in fail-open, got nil")
	}
}

func TestBundleEntitled_InvalidJSON(t *testing.T) {
	checkerCacheDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LICENSE_PING_URL", srv.URL)

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if ok {
		t.Error("expected false for invalid JSON, got true")
	}
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestBundleEntitled_ValidFalseEmptyReason(t *testing.T) {
	checkerCacheDir(t)
	// valid=false with empty reason triggers the default "not entitled" message.
	newBundleServer(t, http.StatusOK, bundleValidateResponse{
		Valid:  false,
		Tier:   "free",
		Reason: "",
		Bundle: "nchat",
	})

	ok, err := BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nchat")
	if ok {
		t.Error("expected false for empty-reason denial, got true")
	}
	if err == nil {
		t.Error("expected non-nil error for denial, got nil")
	}
}

// ---- bundleEntitledFromCache ------------------------------------------------

func TestBundleEntitledFromCache_NilCache(t *testing.T) {
	checkerCacheDir(t)
	ok, err := bundleEntitledFromCache("nself_pro_testkey", "nclaw")
	if ok {
		t.Error("expected false with no cache, got true")
	}
	if err == nil {
		t.Error("expected error with no cache, got nil")
	}
}

func TestBundleEntitledFromCache_WrongKey(t *testing.T) {
	checkerCacheDir(t)
	checkerWriteCache(t, "some-other-key", "plus", nil)

	ok, err := bundleEntitledFromCache("different-key", "nclaw")
	if ok {
		t.Error("expected false for mismatched key hash, got true")
	}
	if err == nil {
		t.Error("expected error for mismatched key hash, got nil")
	}
}

func TestBundleEntitledFromCache_PlusTierCoversAll(t *testing.T) {
	const key = "nself_pro_abcdefghijklmnopqrstuvwxyz123456"

	for _, tier := range []string{"plus", "enterprise", "owner"} {
		tier := tier
		t.Run("tier="+tier, func(t *testing.T) {
			checkerCacheDir(t)
			checkerWriteCache(t, key, tier, nil)

			ok, err := bundleEntitledFromCache(key, "nclaw")
			if err != nil {
				t.Fatalf("unexpected error for tier %q: %v", tier, err)
			}
			if !ok {
				t.Errorf("expected true for tier %q, got false", tier)
			}
		})
	}
}

func TestBundleEntitledFromCache_BundleSentinelMatch(t *testing.T) {
	checkerCacheDir(t)
	const key = "nself_pro_abcdefghijklmnopqrstuvwxyz123456"
	checkerWriteCache(t, key, "bundle", []string{"bundle:nclaw", "claw", "ai"})

	ok, err := bundleEntitledFromCache(key, "nclaw")
	if err != nil {
		t.Fatalf("unexpected error with bundle sentinel: %v", err)
	}
	if !ok {
		t.Error("expected true for bundle:nclaw sentinel, got false")
	}
}

func TestBundleEntitledFromCache_BundleSentinelCaseInsensitive(t *testing.T) {
	checkerCacheDir(t)
	const key = "nself_pro_abcdefghijklmnopqrstuvwxyz123456"
	checkerWriteCache(t, key, "bundle", []string{"Bundle:NClaw"})

	ok, err := bundleEntitledFromCache(key, "nclaw")
	if err != nil {
		t.Fatalf("unexpected error with case-mixed sentinel: %v", err)
	}
	if !ok {
		t.Error("expected true for case-insensitive sentinel, got false")
	}
}

func TestBundleEntitledFromCache_NoSentinelNoHighTier(t *testing.T) {
	checkerCacheDir(t)
	const key = "nself_pro_abcdefghijklmnopqrstuvwxyz123456"
	checkerWriteCache(t, key, "basic", []string{"claw", "ai", "mux"})

	ok, err := bundleEntitledFromCache(key, "nclaw")
	if ok {
		t.Error("expected false for tier=basic + no sentinel, got true")
	}
	if err == nil {
		t.Error("expected error when bundle not found in cache, got nil")
	}
}

// ---- CollectLicenseKey ------------------------------------------------------

func TestCollectLicenseKey_OwnerEnvPriority(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY_OWNER", "owner-key-value")
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "plugin-key-value")

	got := CollectLicenseKey()
	if got != "owner-key-value" {
		t.Errorf("expected owner key, got %q", got)
	}
}

func TestCollectLicenseKey_PluginEnvFallback(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY_OWNER", "")
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "plugin-key-value")

	got := CollectLicenseKey()
	if got != "plugin-key-value" {
		t.Errorf("expected plugin key, got %q", got)
	}
}

func TestCollectLicenseKey_BothEnvEmpty_FallsToGetKey(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY_OWNER", "")
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")
	// Falls through to GetKey() which reads from disk. Must not panic.
	_ = CollectLicenseKey()
}

func TestCollectLicenseKey_OwnerAlone(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY_OWNER", "solo-owner-key")
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")

	got := CollectLicenseKey()
	if got != "solo-owner-key" {
		t.Errorf("expected owner key, got %q", got)
	}
}

// ---- BundleEntitled request shape checks ------------------------------------

func TestBundleEntitled_KeySentInHeader(t *testing.T) {
	checkerCacheDir(t)
	const wantKey = "nself_pro_testkey_headercheck_ABCDEFGH"
	var gotKey string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-NSelf-License-Key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(bundleValidateResponse{Valid: true, Tier: "plus"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LICENSE_PING_URL", srv.URL)

	ok, err := BundleEntitled(context.Background(), wantKey, "nclaw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
	if gotKey != wantKey {
		t.Errorf("X-NSelf-License-Key = %q, want %q", gotKey, wantKey)
	}
}

func TestBundleEntitled_UsesHTTPPost(t *testing.T) {
	checkerCacheDir(t)
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(bundleValidateResponse{Valid: true, Tier: "plus"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LICENSE_PING_URL", srv.URL)

	_, _ = BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "nclaw")
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %q", gotMethod)
	}
}

func TestBundleEntitled_BundleParamInURL(t *testing.T) {
	checkerCacheDir(t)
	var gotBundle string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBundle = r.URL.Query().Get("bundle")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(bundleValidateResponse{Valid: true, Tier: "plus"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LICENSE_PING_URL", srv.URL)

	_, _ = BundleEntitled(context.Background(), "nself_pro_abcdefghijklmnopqrstuvwxyz123456", "ntv")
	if gotBundle != "ntv" {
		t.Errorf("expected bundle=ntv in URL, got %q", gotBundle)
	}
}
