package embedded

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testVersion is a fake version with a known pin used only in tests.
// It is injected into sha256Pins at the start of each test and removed after.
const testVersion = "0.0.0-test"

// testWASMContent is the fake WASM payload used by mock HTTP servers.
var testWASMContent = []byte("fake pglite wasm payload for testing")

// testPin returns the SHA-256 hex digest of testWASMContent.
func testPin() string {
	h := sha256.Sum256(testWASMContent)
	return hex.EncodeToString(h[:])
}

// withTestPin registers a fake version pin for the duration of t and removes
// it on cleanup.
func withTestPin(t *testing.T) string {
	t.Helper()
	pin := testPin()
	sha256Pins[testVersion] = pin
	t.Cleanup(func() { delete(sha256Pins, testVersion) })
	return pin
}

// useTempHome sets $HOME to a temp dir so cache paths stay inside t.TempDir().
// Returns the temp dir path.
func useTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

// useMockServer registers an httptest.Server that serves testWASMContent and
// overrides pgliteBaseURLOverride for the duration of t.
func useMockServer(t *testing.T, payload []byte, requestCounter *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCounter != nil {
			*requestCounter++
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	old := pgliteBaseURLOverride
	pgliteBaseURLOverride = srv.URL
	t.Cleanup(func() { pgliteBaseURLOverride = old })

	return srv
}

// TestFetchOrCached_HappyPath verifies that FetchOrCached downloads the WASM
// artifact from a mock HTTP server, writes it to the cache, and returns the
// correct absolute path.
func TestFetchOrCached_HappyPath(t *testing.T) {
	withTestPin(t)
	home := useTempHome(t)
	useMockServer(t, testWASMContent, nil)

	path, err := FetchOrCached(context.Background(), testVersion)
	if err != nil {
		t.Fatalf("FetchOrCached returned error: %v", err)
	}

	expectedPath := filepath.Join(home, ".nself", "cache", "pglite", testVersion, "pglite.wasm")
	if path != expectedPath {
		t.Errorf("path = %q, want %q", path, expectedPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != string(testWASMContent) {
		t.Error("cached file content does not match download payload")
	}
}

// TestFetchOrCached_SHA256Mismatch verifies that FetchOrCached returns an error
// when the downloaded artifact's SHA-256 does not match the pinned value.
func TestFetchOrCached_SHA256Mismatch(t *testing.T) {
	withTestPin(t)
	useTempHome(t)
	useMockServer(t, []byte("tampered content — wrong sha256"), nil)

	_, err := FetchOrCached(context.Background(), testVersion)
	if err == nil {
		t.Fatal("expected sha256 mismatch error, got nil")
	}
}

// TestFetchOrCached_CacheHit verifies that when a valid cached artifact exists,
// FetchOrCached returns its path without making any HTTP request.
func TestFetchOrCached_CacheHit(t *testing.T) {
	withTestPin(t)
	home := useTempHome(t)

	// Pre-populate the cache with a valid artifact.
	cacheDir := filepath.Join(home, ".nself", "cache", "pglite", testVersion)
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cachedPath := filepath.Join(cacheDir, "pglite.wasm")
	if err := os.WriteFile(cachedPath, testWASMContent, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var reqCount int
	useMockServer(t, testWASMContent, &reqCount)

	path, err := FetchOrCached(context.Background(), testVersion)
	if err != nil {
		t.Fatalf("FetchOrCached returned error: %v", err)
	}
	if path != cachedPath {
		t.Errorf("path = %q, want %q", path, cachedPath)
	}
	if reqCount != 0 {
		t.Errorf("expected 0 HTTP requests for cache hit, got %d", reqCount)
	}
}

// TestFetchOrCached_UnknownVersion verifies that FetchOrCached returns an error
// for a version with no sha256 pin registered.
func TestFetchOrCached_UnknownVersion(t *testing.T) {
	useTempHome(t)
	_, err := FetchOrCached(context.Background(), "9.9.9-does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown version, got nil")
	}
}

// TestFetchOrCached_StaleCache verifies that a cached file whose mtime exceeds
// cacheMaxAge triggers a re-download even when the digest still matches.
func TestFetchOrCached_StaleCache(t *testing.T) {
	withTestPin(t)
	home := useTempHome(t)

	cacheDir := filepath.Join(home, ".nself", "cache", "pglite", testVersion)
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cachedPath := filepath.Join(cacheDir, "pglite.wasm")
	if err := os.WriteFile(cachedPath, testWASMContent, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Back-date the file to exceed cacheMaxAge.
	staleTime := time.Now().Add(-(cacheMaxAge + time.Hour))
	if err := os.Chtimes(cachedPath, staleTime, staleTime); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	var reqCount int
	useMockServer(t, testWASMContent, &reqCount)

	_, err := FetchOrCached(context.Background(), testVersion)
	if err != nil {
		t.Fatalf("FetchOrCached returned error: %v", err)
	}
	if reqCount == 0 {
		t.Error("expected HTTP request for stale cache, got none")
	}
}
