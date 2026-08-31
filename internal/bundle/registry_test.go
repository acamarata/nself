package bundle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLoad_FetchSuccess_PopulatesFromLiveServer proves Load performs an
// eager, real HTTP fetch (not a lazy/no-op) and populates the resolver from
// the response body.
func TestLoad_FetchSuccess_PopulatesFromLiveServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixtureBundlesJSON))
	}))
	defer srv.Close()
	t.Setenv("NSELF_BUNDLES_URL", srv.URL)
	t.Setenv("NSELF_BUNDLES_CACHE_PATH", t.TempDir()+"/bundles-cache.json")

	if err := Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := Get("claw"); !ok {
		t.Error("Load did not populate the resolver from the live server response")
	}
	// Re-seed the package-wide fixture for tests that run after this one.
	if err := LoadBytes([]byte(fixtureBundlesJSON)); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
}

// TestLoad_FetchFailure_FallsBackToCache proves the offline-cache fallback:
// with a prior successful fetch cached on disk, a subsequent failed fetch
// still returns the last-known-good bundle set rather than erroring
// (ticket acceptance criterion #4).
func TestLoad_FetchFailure_FallsBackToCache(t *testing.T) {
	cachePath := t.TempDir() + "/bundles-cache.json"
	t.Setenv("NSELF_BUNDLES_CACHE_PATH", cachePath)

	// First: a successful fetch populates the cache.
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixtureBundlesJSON))
	}))
	t.Setenv("NSELF_BUNDLES_URL", okSrv.URL)
	if err := Load(context.Background()); err != nil {
		t.Fatalf("priming Load: %v", err)
	}
	okSrv.Close()

	// Second: point at a dead server (connection refused) — fetch fails,
	// must degrade to the cache written above rather than erroring.
	deadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := deadSrv.URL
	deadSrv.Close() // now guaranteed connection-refused

	t.Setenv("NSELF_BUNDLES_URL", deadURL)
	if err := Load(context.Background()); err != nil {
		t.Fatalf("Load should fall back to cache on fetch failure, got error: %v", err)
	}
	if _, ok := Get("claw"); !ok {
		t.Error("Load did not serve the cached bundle set on fetch failure")
	}

	// Re-seed for subsequent tests in this package.
	if err := LoadBytes([]byte(fixtureBundlesJSON)); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
}

// TestLoad_FetchFailure_NoCache_ReturnsError proves Load does NOT silently
// serve an empty bundle set when there is neither a successful fetch nor a
// cache — it must name the network issue (ticket guide step 3).
func TestLoad_FetchFailure_NoCache_ReturnsError(t *testing.T) {
	deadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := deadSrv.URL
	deadSrv.Close()

	t.Setenv("NSELF_BUNDLES_URL", deadURL)
	t.Setenv("NSELF_BUNDLES_CACHE_PATH", t.TempDir()+"/never-written.json")

	err := Load(context.Background())
	if err == nil {
		t.Fatal("expected error when fetch fails and no cache exists")
	}
	if !strings.Contains(err.Error(), "fetching bundles.json") {
		t.Errorf("error should name the network issue, got: %v", err)
	}

	// Re-seed for subsequent tests in this package.
	if err := LoadBytes([]byte(fixtureBundlesJSON)); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
}

func TestLoadBytes_RejectsBadSchemaVersion(t *testing.T) {
	err := LoadBytes([]byte(`{"schema_version":"1.0.0","bundles":{"task":{"display":"x","tier":"free","plugins":[]}}}`))
	if err == nil {
		t.Fatal("expected schema_version rejection")
	}
	// Re-seed: LoadBytes must not have clobbered state on validation failure
	// in a way that breaks other tests.
	if err := LoadBytes([]byte(fixtureBundlesJSON)); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
}

func TestLoadBytes_RejectsEmptyBundles(t *testing.T) {
	err := LoadBytes([]byte(`{"schema_version":"2.0.0","bundles":{}}`))
	if err == nil {
		t.Fatal("expected empty-bundles rejection")
	}
	if err := LoadBytes([]byte(fixtureBundlesJSON)); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
}
