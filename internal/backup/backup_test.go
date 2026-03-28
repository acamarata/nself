package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateBackupPath_Valid verifies that a valid relative path inside the
// backup directory is accepted and returns the correct resolved absolute path.
func TestValidateBackupPath_Valid(t *testing.T) {
	backupDir, err := os.MkdirTemp("", "backup-valid-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(backupDir)

	got, err := ValidateBackupPath(backupDir, "data.dump")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(backupDir, "data.dump")
	// Both paths should resolve to the same location. Use EvalSymlinks on
	// want so macOS /var -> /private/var and similar do not cause a mismatch.
	wantResolved, _ := filepath.EvalSymlinks(filepath.Dir(want))
	wantResolved = filepath.Join(wantResolved, filepath.Base(want))

	if got != wantResolved {
		t.Errorf("path = %q, want %q", got, wantResolved)
	}
}

// TestValidateBackupPath_PathTraversal verifies that a path containing ".."
// segments that would escape the backup directory is rejected.
func TestValidateBackupPath_PathTraversal(t *testing.T) {
	backupDir, err := os.MkdirTemp("", "backup-traversal-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(backupDir)

	_, err = ValidateBackupPath(backupDir, "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "path traversal") && !strings.Contains(msg, "escapes") {
		t.Errorf("error message %q should contain 'path traversal' or 'escapes'", msg)
	}
}

// TestValidateBackupPath_Symlink verifies that a symlink inside the backup
// directory that points to a location outside the backup directory is rejected.
func TestValidateBackupPath_Symlink(t *testing.T) {
	backupDir, err := os.MkdirTemp("", "backup-symlink-*")
	if err != nil {
		t.Fatalf("create backup temp dir: %v", err)
	}
	defer os.RemoveAll(backupDir)

	evilDir, err := os.MkdirTemp("", "evil-*")
	if err != nil {
		t.Fatalf("create evil temp dir: %v", err)
	}
	defer os.RemoveAll(evilDir)

	// Create a symlink named "safe-link" inside backupDir that points outside.
	linkPath := filepath.Join(backupDir, "safe-link")
	if err := os.Symlink(evilDir, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err = ValidateBackupPath(backupDir, "safe-link/data.dump")
	if err == nil {
		t.Fatal("expected error for symlink pointing outside backup dir, got nil")
	}
}

// TestListBackups_EmptyDir verifies that ListBackups returns an empty (non-nil)
// slice and no error when the backup directory contains no files.
func TestListBackups_EmptyDir(t *testing.T) {
	backupDir := t.TempDir()

	names, err := ListBackups(backupDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if names == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(names) != 0 {
		t.Errorf("expected empty slice, got %v", names)
	}
}

// TestListBackups_MultipleFiles verifies that ListBackups returns all regular
// files in the backup directory and skips subdirectories.
func TestListBackups_MultipleFiles(t *testing.T) {
	backupDir := t.TempDir()

	// Create three backup files.
	wantFiles := []string{"alpha.dump", "beta.dump", "gamma.dump"}
	for _, name := range wantFiles {
		f, err := os.Create(filepath.Join(backupDir, name))
		if err != nil {
			t.Fatalf("create file %s: %v", name, err)
		}
		f.Close()
	}

	// Also create a subdirectory — it must NOT appear in the result.
	if err := os.Mkdir(filepath.Join(backupDir, "subdir"), 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	names, err := ListBackups(backupDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != len(wantFiles) {
		t.Fatalf("expected %d files, got %d: %v", len(wantFiles), len(names), names)
	}

	// Build a set for order-independent comparison.
	got := make(map[string]bool, len(names))
	for _, n := range names {
		got[n] = true
	}
	for _, want := range wantFiles {
		if !got[want] {
			t.Errorf("expected file %q in result, but it was missing (got %v)", want, names)
		}
	}
}

// TestValidateBackupPath_ValidSubdir verifies that a path nested inside a
// subdirectory of the backup directory is accepted.
func TestValidateBackupPath_ValidSubdir(t *testing.T) {
	backupDir, err := os.MkdirTemp("", "backup-subdir-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(backupDir)

	subdir := filepath.Join(backupDir, "2024", "monthly")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	got, err := ValidateBackupPath(backupDir, "2024/monthly/db.dump")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolved path must still be under backupDir.
	resolvedBase, _ := filepath.EvalSymlinks(backupDir)
	prefix := resolvedBase + string(filepath.Separator)
	if got != resolvedBase && !strings.HasPrefix(got, prefix) {
		t.Errorf("path %q does not start with backup dir %q", got, resolvedBase)
	}
}
