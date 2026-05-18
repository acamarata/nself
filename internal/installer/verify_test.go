package installer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// sha256Hex returns the hex-encoded SHA-256 of b.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TestDownloadAndVerify_HappyPath verifies that DownloadAndVerify returns a
// temporary file path when the downloaded content matches the expected SHA.
func TestDownloadAndVerify_HappyPath(t *testing.T) {
	content := []byte("#!/bin/sh\necho 'hello from mock install.sh'\n")
	expected := sha256Hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	tmpPath, err := DownloadAndVerify(context.Background(), srv.URL, expected)
	if err != nil {
		t.Fatalf("DownloadAndVerify returned unexpected error: %v", err)
	}
	defer os.Remove(tmpPath)

	if tmpPath == "" {
		t.Fatal("DownloadAndVerify returned empty path on success")
	}

	// Verify the file exists and contains the expected content.
	got, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("read temp file %q: %v", tmpPath, err)
	}
	if string(got) != string(content) {
		t.Errorf("temp file content = %q, want %q", got, content)
	}
}

// TestDownloadAndVerify_Mismatch verifies that DownloadAndVerify returns an
// error and leaves no temp file behind when the checksum does not match.
func TestDownloadAndVerify_Mismatch(t *testing.T) {
	content := []byte("#!/bin/sh\necho 'tampered'\n")
	wrongSHA := sha256Hex([]byte("not the same content"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	tmpPath, err := DownloadAndVerify(context.Background(), srv.URL, wrongSHA)
	if err == nil {
		os.Remove(tmpPath)
		t.Fatal("DownloadAndVerify: expected checksum mismatch error, got nil")
	}

	// Temp file must not remain on disk after a mismatch.
	if tmpPath != "" {
		if _, statErr := os.Stat(tmpPath); statErr == nil {
			os.Remove(tmpPath)
			t.Errorf("DownloadAndVerify left temp file %q behind on mismatch", tmpPath)
		}
	}
}

// TestDownloadAndVerify_EmptySHA verifies that DownloadAndVerify rejects an
// empty expectedSHA immediately without making any network request.
func TestDownloadAndVerify_EmptySHA(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := DownloadAndVerify(context.Background(), srv.URL, "")
	if err == nil {
		t.Fatal("DownloadAndVerify: expected error for empty SHA, got nil")
	}
	if called {
		t.Error("DownloadAndVerify: made HTTP request despite empty expectedSHA")
	}
}

// TestDownloadAndVerify_HTTPError verifies that a non-200 response produces an
// error.
func TestDownloadAndVerify_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := DownloadAndVerify(context.Background(), srv.URL, sha256Hex([]byte("x")))
	if err == nil {
		t.Fatal("DownloadAndVerify: expected error for HTTP 404, got nil")
	}
}

// TestDownloadAndVerify_TempDirPerms verifies that the private temporary
// directory is created with 0700 permissions and the script file with 0600
// permissions. This enforces the TOCTOU mitigation: the directory is
// owner-only so no same-uid sibling process can swap the file path.
func TestDownloadAndVerify_TempDirPerms(t *testing.T) {
	content := []byte("#!/bin/sh\necho 'perm check'\n")
	expected := sha256Hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	tmpPath, err := DownloadAndVerify(context.Background(), srv.URL, expected)
	if err != nil {
		t.Fatalf("DownloadAndVerify returned unexpected error: %v", err)
	}
	dir := filepath.Dir(tmpPath)
	defer os.RemoveAll(dir)

	// Directory must be 0700 (owner-only).
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat temp dir %q: %v", dir, err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("temp dir permissions = %04o, want 0700", perm)
	}

	// Script file must be 0600.
	fileInfo, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("stat temp file %q: %v", tmpPath, err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("temp file permissions = %04o, want 0600", perm)
	}
}

// TestDownloadAndVerify_ContextCancel verifies that a context cancelled before
// the download completes returns an error and does not proceed to exec.
func TestDownloadAndVerify_ContextCancel(t *testing.T) {
	// Serve content slowly enough that the already-cancelled context fires.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write headers but block on body — the cancelled context should
		// cause the client-side read to fail before we write anything.
		_, _ = w.Write([]byte("#!/bin/sh\n"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the request

	tmpPath, err := DownloadAndVerify(ctx, srv.URL, sha256Hex([]byte("any")))
	if err == nil {
		if tmpPath != "" {
			os.RemoveAll(filepath.Dir(tmpPath))
		}
		t.Fatal("DownloadAndVerify: expected error for cancelled context, got nil")
	}
	// No temp file should remain.
	if tmpPath != "" {
		if _, statErr := os.Stat(tmpPath); statErr == nil {
			os.RemoveAll(filepath.Dir(tmpPath))
			t.Errorf("DownloadAndVerify left temp file %q behind on context cancel", tmpPath)
		}
	}
}

// TestDownloadAndVerify_LimitReaderCap verifies that a body larger than the
// 2 MiB cap produces a checksum mismatch error, not a silent truncation.
// The caller supplies the hash of the full content; the LimitReader truncates
// the body, so the computed hash will not match, and DownloadAndVerify must
// return an error with a clear mismatch message.
func TestDownloadAndVerify_LimitReaderCap(t *testing.T) {
	const capBytes = 2 * 1024 * 1024 // 2 MiB — matches internal LimitReader cap

	// Build a body strictly larger than the cap.
	body := bytes.Repeat([]byte("x"), capBytes+1)
	// The "correct" hash the caller would supply is for the full body.
	correctSHA := sha256Hex(body)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	tmpPath, err := DownloadAndVerify(context.Background(), srv.URL, correctSHA)
	if err == nil {
		if tmpPath != "" {
			os.RemoveAll(filepath.Dir(tmpPath))
		}
		t.Fatal("DownloadAndVerify: expected checksum mismatch for body > 2 MiB cap, got nil")
	}
	// Confirm the error is a checksum mismatch (not some other failure).
	ie, ok := err.(*InstallerError)
	if !ok {
		t.Fatalf("DownloadAndVerify: error is %T, want *InstallerError", err)
	}
	if ie.Code != ErrOllamaInstallFailed {
		t.Errorf("error code = %q, want %q", ie.Code, ErrOllamaInstallFailed)
	}
	// No temp artefacts should remain.
	if tmpPath != "" {
		if _, statErr := os.Stat(tmpPath); statErr == nil {
			os.RemoveAll(filepath.Dir(tmpPath))
			t.Errorf("DownloadAndVerify left temp file %q behind after mismatch", tmpPath)
		}
	}
}
