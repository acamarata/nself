package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeTestTarGz creates a minimal gzipped tarball containing a single file at
// {name}/{name} with the given content and returns the archive bytes.
func makeTestTarGz(t *testing.T, pluginName, content string) []byte {
	t.Helper()
	var buf strings.Builder
	// use a pipe via os.Pipe for simplicity — build in-memory using a temp file
	tmp, err := os.CreateTemp("", "test-archive-*.tar.gz")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)

	body := []byte(content)
	hdr := &tar.Header{
		Name:     pluginName + "/" + pluginName,
		Typeflag: tar.TypeReg,
		Mode:     0o755,
		Size:     int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	tw.Close()
	gz.Close()
	tmp.Close()

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	return data
}

// sha256Hex returns the hex-encoded SHA-256 of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// TestVerifySHA256_Match confirms that verifySHA256 returns nil on correct checksum.
func TestVerifySHA256_Match(t *testing.T) {
	tmp, err := os.CreateTemp("", "cs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	content := []byte("hello binary plugin")
	tmp.Write(content)
	tmp.Close()

	expected := sha256Hex(content)
	if err := verifySHA256(tmp.Name(), expected); err != nil {
		t.Fatalf("expected nil error for matching checksum, got: %v", err)
	}
}

// TestVerifySHA256_Mismatch confirms that verifySHA256 returns an error on wrong checksum.
func TestVerifySHA256_Mismatch(t *testing.T) {
	tmp, err := os.CreateTemp("", "cs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.Write([]byte("hello binary plugin"))
	tmp.Close()

	if err := verifySHA256(tmp.Name(), "deadbeef"); err == nil {
		t.Fatal("expected error for mismatched checksum, got nil")
	}
}

// TestInstallBinaryPlugin_MockHTTP verifies the full binary plugin install flow
// using a mocked HTTP server serving a pre-built tarball.
// The test covers: download, checksum verification, extraction, and chmod.
func TestInstallBinaryPlugin_MockHTTP(t *testing.T) {
	const pluginName = "test-binary-plugin"
	const version = "1.0.0"
	const binaryContent = "#!/bin/sh\necho hello"

	// Build the test tarball.
	archiveData := makeTestTarGz(t, pluginName, binaryContent)
	checksum := sha256Hex(archiveData)

	// Stand up a mock HTTP server serving the tarball.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer srv.Close()

	// Build a manifest pointing at the mock server.
	manifest := &PluginManifest{
		Name:          pluginName,
		Version:       version,
		Description:   "test",
		Category:      "test",
		License:       "MIT",
		Language:      "rust",
		Repository:    srv.URL + "/repo",
		Checksum:      checksum,
		PublishStatus: "stable",
	}

	// Override the URL so it hits our mock.
	// We replace the binaryReleaseURL resolution by injecting via the repository
	// field which our binaryReleaseURL already uses.
	// The generated URL will be: {srv.URL}/repo/releases/download/v1.0.0/test-binary-plugin-v1.0.0-<arch>.tar.gz
	// The mock ignores the path and always returns archiveData, so this works.

	pluginDir := t.TempDir()
	ctx := context.Background()

	if err := installBinaryPlugin(ctx, manifest, pluginDir); err != nil {
		t.Fatalf("installBinaryPlugin: %v", err)
	}

	// Confirm binary exists and is executable.
	binPath := filepath.Join(pluginDir, pluginName, binaryName(pluginName))
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("binary not found at %s: %v", binPath, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("binary at %s is not executable, mode: %o", binPath, info.Mode())
	}
}

// TestInstallBinaryPlugin_ChecksumFailure verifies that a checksum mismatch
// causes the install to fail and no files are left on disk.
func TestInstallBinaryPlugin_ChecksumFailure(t *testing.T) {
	const pluginName = "bad-checksum-plugin"
	archiveData := makeTestTarGz(t, pluginName, "binary content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer srv.Close()

	manifest := &PluginManifest{
		Name:          pluginName,
		Version:       "1.0.0",
		Description:   "test",
		Category:      "test",
		License:       "MIT",
		Language:      "rust",
		Repository:    srv.URL + "/repo",
		Checksum:      "0000000000000000000000000000000000000000000000000000000000000000",
		PublishStatus: "stable",
	}

	pluginDir := t.TempDir()
	err := installBinaryPlugin(context.Background(), manifest, pluginDir)
	if err == nil {
		t.Fatal("expected checksum failure error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("expected 'checksum' in error, got: %v", err)
	}

	// destDir must not exist after a checksum failure.
	destDir := filepath.Join(pluginDir, pluginName)
	if _, statErr := os.Stat(destDir); !os.IsNotExist(statErr) {
		t.Errorf("plugin dir %s should not exist after checksum failure", destDir)
	}
}
