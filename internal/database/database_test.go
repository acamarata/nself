package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRestoreSymlinkBypass verifies that a symlink inside the backup directory
// that resolves to a location outside it is rejected by Restore (S1-T09 / S1-T10
// regression guard). This complements the traversal tests in restore_test.go.
func TestRestoreSymlinkBypass(t *testing.T) {
	backupDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create a symlink named "link" inside the backup dir that points outside.
	linkPath := filepath.Join(backupDir, "link")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	cfg := newTestCfg(backupDir)

	// Attempt to restore via "link/data.dump" — the symlink target is outside
	// the backup directory so ValidateBackupPath must reject it.
	err := Restore(context.Background(), cfg, "link/data.dump")
	if err == nil {
		t.Fatal("expected error for symlink that escapes backup directory, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid restore path") &&
		!strings.Contains(msg, "escapes") &&
		!strings.Contains(msg, "path traversal") {
		t.Errorf("error message %q does not indicate path validation failure", msg)
	}
}
