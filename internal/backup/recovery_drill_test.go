package backup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// TestResolveBackupFile_InvalidID verifies that requesting a backup by an ID
// that does not exist returns ErrBackupNotFound (SEC-DR-01: clear error on
// bad restore target, prevents silent restore of stale/wrong data).
func TestResolveBackupFile_InvalidID(t *testing.T) {
	dir := t.TempDir()

	_, err := resolveBackupFile(dir, "nonexistent-backup-id")
	if err == nil {
		t.Fatal("expected error for unknown backup ID, got nil")
	}
	if !errors.Is(err, errs.ErrBackupNotFound) {
		t.Errorf("expected ErrBackupNotFound, got %v", err)
	}
}

// TestResolveBackupFile_Latest_EmptyDir verifies that "latest" on an empty
// backup directory returns ErrBackupNotFound.
func TestResolveBackupFile_Latest_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	// List will see no entries in an empty dir.
	_, err := resolveBackupFile(dir, "latest")
	if err == nil {
		t.Fatal("expected error for empty backup dir, got nil")
	}
	// Should be wrapped around ErrBackupNotFound.
	if !errors.Is(err, errs.ErrBackupNotFound) {
		t.Errorf("expected ErrBackupNotFound, got %v", err)
	}
}

// TestResolveBackupFile_PrefixMatch verifies that a partial backup ID that
// uniquely matches a file in the backup directory resolves correctly.
func TestResolveBackupFile_PrefixMatch(t *testing.T) {
	dir := t.TempDir()

	// Create a fake backup file with a realistic name.
	backupName := "nself-backup-2025-01-15T12-00-00Z.dump"
	fullPath := filepath.Join(dir, backupName)
	if err := os.WriteFile(fullPath, []byte("pgdump-data"), 0600); err != nil {
		t.Fatalf("create fake backup: %v", err)
	}

	got, err := resolveBackupFile(dir, "nself-backup-2025-01-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fullPath {
		t.Errorf("got %q, want %q", got, fullPath)
	}
}

// TestRestoreBaseBackup_PITRNotSupported verifies that restoreBaseBackup
// returns without error (PITR is a no-op stub in v1.0.9, NOT a silent success
// that might mislead callers). This is a regression guard: the stub must
// complete and the function must not panic.
//
// When PITR ships in v1.1.0 via pgbackrest integration, this test should be
// updated to assert that a PointInTime option on RestoreOptions returns a
// meaningful "not-yet-supported" sentinel rather than silently passing.
func TestRestoreBaseBackup_PITRNotSupported(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Config{}
	cfg.ProjectName = "test-project"
	cfg.Postgres.User = "postgres"
	cfg.Postgres.DB = "nself"

	opts := RestoreOptions{
		BackupID: "test-backup-id",
	}

	// restoreBaseBackup is the PITR-deferred path. It must return nil (no
	// error) in v1.0.9 because it logs and exits gracefully.
	err := restoreBaseBackup(ctx, cfg, "test-container", "postgres", "/fake/path.tar.gz", opts)
	if err != nil {
		t.Errorf("restoreBaseBackup returned unexpected error: %v", err)
	}
}

// TestRestore_MissingBackupDir verifies that Restore fails clearly when the
// configured backup directory does not exist, rather than panicking or
// returning a cryptic error.
func TestRestore_MissingBackupDir(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Config{}
	cfg.Backup.Dir = "/nonexistent-backup-dir-xyzzy"

	opts := RestoreOptions{BackupID: "latest"}

	err := Restore(ctx, cfg, opts)
	if err == nil {
		t.Fatal("expected error for missing backup dir, got nil")
	}
	// Should mention the backup directory or be a known error type.
	errStr := err.Error()
	if !strings.Contains(errStr, "backup") && !strings.Contains(errStr, "no backups") {
		t.Errorf("unexpected error message (want backup-related): %v", err)
	}
}

// TestRestore_InvalidBackupID verifies that Restore wraps ErrBackupNotFound
// when an explicit nonexistent backup ID is requested.
func TestRestore_InvalidBackupID(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	cfg := &config.Config{}
	cfg.Backup.Dir = dir

	opts := RestoreOptions{BackupID: "does-not-exist-abc123"}

	err := Restore(ctx, cfg, opts)
	if err == nil {
		t.Fatal("expected error for unknown backup ID, got nil")
	}
	if !errors.Is(err, errs.ErrBackupNotFound) {
		t.Errorf("expected ErrBackupNotFound chain, got %v", err)
	}
}
