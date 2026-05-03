package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCheckHasuraMetadataBackup_NoDir verifies that a missing backup directory
// returns a warn result.
func TestCheckHasuraMetadataBackup_NoDir(t *testing.T) {
	res := CheckHasuraMetadataBackup("/nonexistent/path/that/does/not/exist")
	if res.Status != "warn" {
		t.Errorf("status = %q, want %q", res.Status, "warn")
	}
	if !strings.Contains(res.Name, "BACKUP-METADATA-01") {
		t.Errorf("name %q should contain BACKUP-METADATA-01", res.Name)
	}
}

// TestCheckHasuraMetadataBackup_NoFiles verifies that an empty backup directory
// returns a fail result.
func TestCheckHasuraMetadataBackup_NoFiles(t *testing.T) {
	dir := t.TempDir()
	res := CheckHasuraMetadataBackup(dir)
	if res.Status != "fail" {
		t.Errorf("status = %q, want %q", res.Status, "fail")
	}
	if res.FixCmd == "" {
		t.Error("FixCmd should be set")
	}
}

// TestCheckHasuraMetadataBackup_FreshBackup verifies that a backup file modified
// less than 36 hours ago returns a pass result.
func TestCheckHasuraMetadataBackup_FreshBackup(t *testing.T) {
	dir := t.TempDir()

	// Create a hasura-metadata-<today>.json file with current mtime.
	today := time.Now().UTC().Format("2006-01-02")
	fname := "hasura-metadata-" + today + ".json"
	if err := os.WriteFile(filepath.Join(dir, fname), []byte(`{}`), 0600); err != nil {
		t.Fatalf("create backup file: %v", err)
	}

	res := CheckHasuraMetadataBackup(dir)
	if res.Status != "pass" {
		t.Errorf("status = %q, want %q (message: %s)", res.Status, "pass", res.Message)
	}
}

// TestCheckHasuraMetadataBackup_StaleBackup verifies that a backup file older
// than 36 hours returns a fail result.
func TestCheckHasuraMetadataBackup_StaleBackup(t *testing.T) {
	dir := t.TempDir()

	fname := "hasura-metadata-2020-01-01.json"
	fpath := filepath.Join(dir, fname)
	if err := os.WriteFile(fpath, []byte(`{}`), 0600); err != nil {
		t.Fatalf("create backup file: %v", err)
	}

	// Back-date the mtime to 2 days ago.
	staleTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(fpath, staleTime, staleTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	res := CheckHasuraMetadataBackup(dir)
	if res.Status != "fail" {
		t.Errorf("status = %q, want %q (message: %s)", res.Status, "fail", res.Message)
	}
	if !strings.Contains(res.Message, "36h") {
		t.Errorf("message %q should mention 36h threshold", res.Message)
	}
}

// TestCheckHasuraMetadataBackup_IgnoresNonMetadataFiles verifies that files
// without the hasura-metadata- prefix are not counted.
func TestCheckHasuraMetadataBackup_IgnoresNonMetadataFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a file with an unrelated name.
	if err := os.WriteFile(filepath.Join(dir, "myproject_full_20240101.tar"), []byte("x"), 0600); err != nil {
		t.Fatalf("create file: %v", err)
	}

	res := CheckHasuraMetadataBackup(dir)
	if res.Status != "fail" {
		t.Errorf("status = %q, want %q — unrelated files should not count", res.Status, "fail")
	}
}
