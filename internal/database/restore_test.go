package database

import (
	"context"
	"strings"
	"testing"

	"nself/internal/config"
)

// newTestCfg returns a minimal Config with Backup.Dir set to backupDir.
func newTestCfg(backupDir string) *config.Config {
	return &config.Config{
		Backup: config.BackupConfig{
			Dir: backupDir,
		},
	}
}

// TestRestorePathTraversal verifies that a relative traversal path (../../etc/shadow)
// is rejected before any file I/O or docker exec occurs.
func TestRestorePathTraversal(t *testing.T) {
	backupDir := t.TempDir()
	cfg := newTestCfg(backupDir)

	err := Restore(context.Background(), cfg, "../../etc/shadow")
	if err == nil {
		t.Fatal("expected error for path traversal input, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid restore path") &&
		!strings.Contains(msg, "escapes") &&
		!strings.Contains(msg, "path traversal") {
		t.Errorf("error message %q does not mention path validation failure", msg)
	}
}

// TestRestoreOutsideBackupDir verifies that a deeply-nested traversal path that
// escapes the backup directory is rejected by path validation.
//
// Note: filepath.Join(absBase, "/etc/passwd") strips the leading slash and
// produces <absBase>/etc/passwd — which is still inside the backup dir. To
// trigger the guard with an absolute-looking payload the traversal must use
// enough ".." segments to climb above absBase, e.g. "../../../etc/passwd".
// This test uses a four-level climb which is guaranteed to escape any temp dir.
func TestRestoreOutsideBackupDir(t *testing.T) {
	backupDir := t.TempDir()
	cfg := newTestCfg(backupDir)

	err := Restore(context.Background(), cfg, "../../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path that escapes the backup directory, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid restore path") &&
		!strings.Contains(msg, "escapes") &&
		!strings.Contains(msg, "path traversal") {
		t.Errorf("error message %q does not mention path validation failure", msg)
	}
}

// TestRestoreValidPath_FileNotFound verifies that a valid relative path inside the
// backup directory passes path validation but fails because the file does not exist.
// This confirms the guard is not over-blocking legitimate paths.
func TestRestoreValidPath_FileNotFound(t *testing.T) {
	backupDir := t.TempDir()
	cfg := newTestCfg(backupDir)

	// "mybackup.dump" is a well-formed name inside the backup dir; it just doesn't
	// exist on disk, so os.Open will fail — not path validation.
	err := Restore(context.Background(), cfg, "mybackup.dump")
	if err == nil {
		t.Fatal("expected error because file does not exist, got nil")
	}

	msg := err.Error()

	// The error must NOT be a path-validation rejection.
	if strings.Contains(msg, "invalid restore path") ||
		strings.Contains(msg, "escapes") ||
		strings.Contains(msg, "path traversal") {
		t.Errorf("got unexpected path-validation error for a valid path: %q", msg)
	}

	// The error must be about opening/accessing the file (not a path-validation rejection).
	if !strings.Contains(msg, "open backup file") && !strings.Contains(msg, "restore file not accessible") {
		t.Errorf("expected file-open or accessibility error, got: %q", msg)
	}
}
