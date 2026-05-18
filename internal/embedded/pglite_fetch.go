package embedded

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// pgliteBaseURL is the CDN endpoint served by ping_api (CS_1).
	pgliteBaseURL = "https://ping.nself.org/assets/pglite"

	// cacheMaxAge is how long a cached WASM file is trusted without re-verifying
	// its SHA-256 digest against the pinned value.
	cacheMaxAge = 7 * 24 * time.Hour

	// downloadTimeout caps the time allowed for the WASM artifact download.
	downloadTimeout = 120 * time.Second
)

// pgliteBaseURLOverride can be set by tests to redirect downloads to a local
// httptest.Server without modifying the pgliteBaseURL constant.
var pgliteBaseURLOverride string

// activePgliteBaseURL returns the effective CDN base URL.
func activePgliteBaseURL() string {
	if pgliteBaseURLOverride != "" {
		return pgliteBaseURLOverride
	}
	return pgliteBaseURL
}

// FetchOrCached returns the absolute path to the pglite.wasm artifact for the
// given version. It checks the local cache first; if the artifact is absent or
// its digest does not match the pinned value it downloads and re-caches it.
//
// The cache directory is ~/.nself/cache/pglite/<version>/.
// The artifact is written atomically (tmp → rename) so partial downloads never
// leave a corrupt cached copy.
//
// FetchOrCached returns an error if:
//   - version is not in sha256Pins
//   - the downloaded artifact's SHA-256 does not match the pin
//   - any I/O or network operation fails
func FetchOrCached(ctx context.Context, version string) (string, error) {
	pin, ok := sha256Pins[version]
	if !ok {
		return "", fmt.Errorf("embedded/pglite: unknown version %q — add a sha256 pin before using", version)
	}

	cacheDir, err := pgliteCacheDir(version)
	if err != nil {
		return "", err
	}

	cachedPath := filepath.Join(cacheDir, "pglite.wasm")

	// Fast path: cache hit with valid digest.
	if ok, _ := fileMatchesDigest(cachedPath, pin, cacheMaxAge); ok {
		return cachedPath, nil
	}

	// Slow path: download from CDN.
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return "", fmt.Errorf("embedded/pglite: create cache dir: %w", err)
	}

	url := fmt.Sprintf("%s/%s/pglite.wasm", activePgliteBaseURL(), version)
	if err := downloadAndVerify(ctx, url, cachedPath, pin); err != nil {
		return "", err
	}

	return cachedPath, nil
}

// pgliteCacheDir returns the version-specific cache directory path.
// It resolves the home directory via os.UserHomeDir so it works correctly
// under sudo and in CI environments.
func pgliteCacheDir(version string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("embedded/pglite: resolve home dir: %w", err)
	}
	return filepath.Join(home, ".nself", "cache", "pglite", version), nil
}

// fileMatchesDigest reports whether path exists, is not a directory, has a
// modification time within maxAge of now, and its SHA-256 hex digest equals
// want. It returns (false, nil) for any condition that warrants a re-download
// rather than an error.
func fileMatchesDigest(path, want string, maxAge time.Duration) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, nil // absent or inaccessible — trigger download
	}
	if fi.IsDir() {
		return false, nil
	}
	if time.Since(fi.ModTime()) > maxAge {
		return false, nil // stale — re-verify
	}

	got, err := sha256HexFile(path)
	if err != nil {
		return false, nil
	}
	return got == want, nil
}

// downloadAndVerify fetches url into a temp file in the same directory as dst,
// verifies its SHA-256 digest against want, then atomically replaces dst.
func downloadAndVerify(ctx context.Context, url, dst, want string) error {
	dlCtx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("embedded/pglite: build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("embedded/pglite: download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("embedded/pglite: server returned %d for %s", resp.StatusCode, url)
	}

	// Write to a sibling temp file so rename is atomic on the same filesystem.
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".pglite-download-*")
	if err != nil {
		return fmt.Errorf("embedded/pglite: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op after successful rename

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("embedded/pglite: write download: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("embedded/pglite: close temp file: %w", err)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("embedded/pglite: sha256 mismatch for version: got %s, want %s", got, want)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		return fmt.Errorf("embedded/pglite: atomic rename: %w", err)
	}
	return nil
}

// sha256HexFile computes the SHA-256 hex digest of the file at path.
func sha256HexFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
