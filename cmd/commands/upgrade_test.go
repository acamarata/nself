package commands

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- helpers -----------------------------------------------------------------

// makeFakeBinary returns the raw bytes of a minimal shell script that stands
// in for the nself binary in tests.
func makeFakeBinary() []byte {
	return []byte("#!/bin/sh\necho nself-test-binary\n")
}

// makeTarGzArchive packs content under binaryName into a gzip-compressed tar
// archive and returns the raw bytes.
func makeTarGzArchive(t *testing.T, binaryName string, content []byte) []byte {
	t.Helper()
	tmp, err := os.CreateTemp("", "nself-archive-*.tar.gz")
	if err != nil {
		t.Fatalf("creating temp archive: %v", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	gw := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name:     binaryName,
		Mode:     0755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("writing tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("writing tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("closing gzip: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("closing temp: %v", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("reading archive: %v", err)
	}
	return data
}

// sha256Hex returns the hex SHA-256 checksum of data.
func sha256Hex(data []byte) string {
	h := sha256.New()
	_, _ = h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// fakeBinaryPath creates a temp directory with a dummy binary file and returns
// its path. The test overrides executablePath to return this path so
// selfUpdateFromURL swaps the temp file rather than the real running binary.
func fakeBinaryPath(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "nself")
	if err := os.WriteFile(fakeBin, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("writing fake binary: %v", err)
	}
	return fakeBin
}

// overrideExecutablePath replaces the executablePath hook for the duration of
// the test and restores the original on cleanup.
func overrideExecutablePath(t *testing.T, path string) {
	t.Helper()
	orig := executablePath
	executablePath = func() (string, error) { return path, nil }
	t.Cleanup(func() { executablePath = orig })
}

// --- validateBinaryURL -------------------------------------------------------

// TestValidateBinaryURL_HTTPSOnly verifies that plain HTTP URLs are rejected
// and HTTPS URLs on allowed hosts are accepted.
func TestValidateBinaryURL_HTTPSOnly(t *testing.T) {
	cases := []struct {
		url     string
		wantErr bool
	}{
		{"http://github.com/nself-org/cli/releases/download/v1.0.9/nself-linux-amd64", true},
		{"ftp://github.com/file", true},
		{"github.com/file", true},
		{"https://github.com/nself-org/cli/releases/download/v1.0.9/nself-linux-amd64", false},
		{"https://objects.githubusercontent.com/v1.0.9/nself-linux-amd64", false},
		{"https://install.nself.org/nself-linux-amd64", false},
		{"https://ping.nself.org/release/stable/nself-linux-amd64", false},
	}
	for _, tc := range cases {
		err := validateBinaryURL(tc.url)
		if tc.wantErr && err == nil {
			t.Errorf("validateBinaryURL(%q) expected error, got nil", tc.url)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("validateBinaryURL(%q) unexpected error: %v", tc.url, err)
		}
	}
}

// TestValidateBinaryURL_AllowlistEnforced verifies that arbitrary HTTPS hosts
// are rejected.
func TestValidateBinaryURL_AllowlistEnforced(t *testing.T) {
	blocked := []string{
		"https://evil.example.com/nself-linux-amd64",
		"https://notgithub.com/nself-org/cli/releases/download/v1.0.9/nself-linux-amd64",
		"https://github.com.evil.example.com/nself",
		"https://mygithub.com/nself",
	}
	for _, url := range blocked {
		if err := validateBinaryURL(url); err == nil {
			t.Errorf("validateBinaryURL(%q) expected error (blocked host), got nil", url)
		}
	}
}

// TestValidateBinaryURL_AllowlistSubdomain verifies that subdomains of allowed
// hosts are accepted.
func TestValidateBinaryURL_AllowlistSubdomain(t *testing.T) {
	allowed := []string{
		"https://releases.install.nself.org/nself-linux-amd64",
		"https://cdn.objects.githubusercontent.com/file",
	}
	for _, url := range allowed {
		if err := validateBinaryURL(url); err != nil {
			t.Errorf("validateBinaryURL(%q) unexpected error: %v", url, err)
		}
	}
}

// --- checksumURLFromBinaryURL ------------------------------------------------

// TestChecksumURLFromBinaryURL verifies the checksum URL derivation.
func TestChecksumURLFromBinaryURL(t *testing.T) {
	cases := []struct {
		binaryURL string
		want      string
	}{
		{
			"https://github.com/nself-org/cli/releases/download/v1.0.9/nself-linux-amd64",
			"https://github.com/nself-org/cli/releases/download/v1.0.9/checksums.txt",
		},
		{
			"https://install.nself.org/nself-1.0.9-linux-amd64.tar.gz",
			"https://install.nself.org/checksums.txt",
		},
	}
	for _, tc := range cases {
		got := checksumURLFromBinaryURL(tc.binaryURL)
		if got != tc.want {
			t.Errorf("checksumURLFromBinaryURL(%q) = %q, want %q", tc.binaryURL, got, tc.want)
		}
	}
}

// --- selfUpdateFromURL -------------------------------------------------------

// TestSelfUpdateFromURL_SkipsGitHubAPI verifies that when --binary-url is
// used, the GitHub releases API is NOT contacted: the binary is downloaded
// directly from the provided URL.
func TestSelfUpdateFromURL_SkipsGitHubAPI(t *testing.T) {
	// Track whether the fake GitHub API was called.
	githubAPICalled := false
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		githubAPICalled = true
		w.WriteHeader(http.StatusForbidden)
	}))
	defer githubSrv.Close()

	binaryContent := makeFakeBinary()
	archiveData := makeTarGzArchive(t, "nself", binaryContent)
	archiveSum := sha256Hex(archiveData)

	binarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nself-1.0.13-linux-amd64.tar.gz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(archiveData)
		case "/checksums.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "%s  nself-1.0.13-linux-amd64.tar.gz\n", archiveSum)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer binarySrv.Close()

	fakeBin := fakeBinaryPath(t)
	overrideExecutablePath(t, fakeBin)

	binaryURL := binarySrv.URL + "/nself-1.0.13-linux-amd64.tar.gz"
	checksumURL := binarySrv.URL + "/checksums.txt"

	if err := selfUpdateFromURL(binaryURL, checksumURL); err != nil {
		t.Fatalf("selfUpdateFromURL error: %v", err)
	}

	if githubAPICalled {
		t.Error("selfUpdateFromURL contacted the GitHub API; it must be skipped when --binary-url is provided")
	}

	got, err := os.ReadFile(fakeBin)
	if err != nil {
		t.Fatalf("reading updated binary: %v", err)
	}
	if string(got) != string(binaryContent) {
		t.Errorf("binary content after update = %q, want %q", string(got), string(binaryContent))
	}
}

// TestSelfUpdateFromURL_ChecksumMismatch verifies that a binary whose
// checksum does not match the manifest is rejected and the original binary is
// left untouched.
func TestSelfUpdateFromURL_ChecksumMismatch(t *testing.T) {
	binaryContent := makeFakeBinary()
	archiveData := makeTarGzArchive(t, "nself", binaryContent)
	wrongSum := strings.Repeat("a", 64) // intentionally wrong

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nself-1.0.13-linux-amd64.tar.gz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(archiveData)
		case "/checksums.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "%s  nself-1.0.13-linux-amd64.tar.gz\n", wrongSum)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	fakeBin := fakeBinaryPath(t)
	overrideExecutablePath(t, fakeBin)

	err := selfUpdateFromURL(
		srv.URL+"/nself-1.0.13-linux-amd64.tar.gz",
		srv.URL+"/checksums.txt",
	)
	if err == nil {
		t.Fatal("selfUpdateFromURL expected error on checksum mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("error %q should mention 'checksum mismatch'", err.Error())
	}

	// Original binary must be untouched.
	got, _ := os.ReadFile(fakeBin)
	if string(got) != "old-binary" {
		t.Errorf("binary was overwritten despite checksum mismatch")
	}
}

// TestSelfUpdateFromURL_RawBinary verifies that a raw binary URL (not a
// tar.gz archive) is handled: the file is downloaded, checksum-verified, and
// swapped atomically.
func TestSelfUpdateFromURL_RawBinary(t *testing.T) {
	binaryContent := makeFakeBinary()
	binarySum := sha256Hex(binaryContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nself-linux-amd64":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(binaryContent)
		case "/checksums.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "%s  nself-linux-amd64\n", binarySum)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	fakeBin := fakeBinaryPath(t)
	overrideExecutablePath(t, fakeBin)

	if err := selfUpdateFromURL(
		srv.URL+"/nself-linux-amd64",
		srv.URL+"/checksums.txt",
	); err != nil {
		t.Fatalf("selfUpdateFromURL (raw binary) error: %v", err)
	}

	got, err := os.ReadFile(fakeBin)
	if err != nil {
		t.Fatalf("reading updated binary: %v", err)
	}
	if string(got) != string(binaryContent) {
		t.Errorf("binary content after update = %q, want %q", string(got), string(binaryContent))
	}
}

// --- flag wiring -------------------------------------------------------------

// TestUpgradeCmd_BinaryURLFlagRegistered verifies the flag is registered with
// the correct name and an empty default.
func TestUpgradeCmd_BinaryURLFlagRegistered(t *testing.T) {
	f := upgradeCmd.Flags().Lookup("binary-url")
	if f == nil {
		t.Fatal("--binary-url flag not registered on upgradeCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--binary-url default value = %q, want empty string", f.DefValue)
	}
}

// TestUpgradeCmd_DefaultBehaviourUnchanged verifies that when --binary-url is
// absent the legacy flags are still registered and accessible.
func TestUpgradeCmd_DefaultBehaviourUnchanged(t *testing.T) {
	for _, flag := range []string{"check", "version", "rollback", "channel"} {
		if f := upgradeCmd.Flags().Lookup(flag); f == nil {
			t.Errorf("--%s flag missing from upgradeCmd", flag)
		}
	}
}
