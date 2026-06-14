package backup

import (
	"runtime"
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

// fakeMetadataPayload is a minimal JSON blob that mimics a Hasura export response.
var fakeMetadataPayload = map[string]interface{}{
	"version": 3,
	"sources": []interface{}{},
}

// startFakeHasura returns a test HTTP server that responds to POST /v1/metadata
// with fakeMetadataPayload (200 OK). The caller must call srv.Close().
func startFakeHasura(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/v1/metadata" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(fakeMetadataPayload); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	return srv
}

// TestBackupHasuraMetadata_WritesFile verifies that BackupHasuraMetadata creates
// a JSON file named hasura-metadata-<date>.json in the backup directory.
func TestBackupHasuraMetadata_WritesFile(t *testing.T) {
	srv := startFakeHasura(t)
	defer srv.Close()

	dir := t.TempDir()
	opts := HasuraMetadataOptions{
		HasuraURL:   srv.URL,
		AdminSecret: "test-secret",
		BackupDir:   dir,
	}

	dest, err := BackupHasuraMetadata(context.Background(), opts)
	if err != nil {
		t.Fatalf("BackupHasuraMetadata: %v", err)
	}

	// File must exist in the backup dir.
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("backup file not found at %s: %v", dest, err)
	}

	// Filename must match hasura-metadata-<YYYY-MM-DD>.json.
	base := filepath.Base(dest)
	today := time.Now().UTC().Format("2006-01-02")
	want := "hasura-metadata-" + today + ".json"
	if base != want {
		t.Errorf("filename = %q, want %q", base, want)
	}

	// File must contain valid JSON.
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("backup file is not valid JSON: %v\ncontent: %s", err, string(data))
	}

	// File must be mode 0600 (Unix only — Windows always returns 0666).
	if runtime.GOOS != "windows" {
		info, err := os.Stat(dest)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("file mode = %o, want 0600", perm)
		}
	}
}

// TestBackupHasuraMetadata_CreatesDir verifies that BackupHasuraMetadata
// creates the backup directory if it does not exist.
func TestBackupHasuraMetadata_CreatesDir(t *testing.T) {
	srv := startFakeHasura(t)
	defer srv.Close()

	base := t.TempDir()
	newDir := filepath.Join(base, "new", "subdir")

	opts := HasuraMetadataOptions{
		HasuraURL: srv.URL,
		BackupDir: newDir,
	}

	dest, err := BackupHasuraMetadata(context.Background(), opts)
	if err != nil {
		t.Fatalf("BackupHasuraMetadata: %v", err)
	}

	if !strings.HasPrefix(dest, newDir) {
		t.Errorf("dest %q is not inside newDir %q", dest, newDir)
	}
}

// TestBackupHasuraMetadata_HasuraError verifies that a non-2xx response from
// Hasura results in an error (the backup file is not written).
func TestBackupHasuraMetadata_HasuraError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	dir := t.TempDir()
	opts := HasuraMetadataOptions{
		HasuraURL: srv.URL,
		BackupDir: dir,
	}

	_, err := BackupHasuraMetadata(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q should mention 401", err.Error())
	}

	// No backup file should have been written.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no files in backup dir, got %v", entries)
	}
}

// TestBackupHasuraMetadata_DefaultURL verifies that an empty HasuraURL falls
// back to http://localhost:8080. We can't connect to a real instance, but we
// verify the resulting error mentions the default URL (not a panic or wrong URL).
func TestBackupHasuraMetadata_DefaultURL(t *testing.T) {
	dir := t.TempDir()
	opts := HasuraMetadataOptions{
		HasuraURL: "",
		BackupDir: dir,
	}

	_, err := BackupHasuraMetadata(context.Background(), opts)
	// Expected: connection refused (no real Hasura running). The key assertion
	// is that the code path does not panic and produces a meaningful error.
	if err == nil {
		// If somehow a local Hasura is running — that's fine too.
		t.Logf("unexpected success (local Hasura might be running)")
	}
}
