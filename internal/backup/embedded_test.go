package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// TestEmbeddedRuntimeDir_EnvWins verifies that NSELF_RUNTIME_DIR overrides the
// cwd-based fallback when set.
func TestEmbeddedRuntimeDir_EnvWins(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_RUNTIME_DIR", dir)

	got := embeddedRuntimeDir()
	if got != dir {
		t.Errorf("expected %q, got %q", dir, got)
	}
}

// TestEmbeddedRuntimeDir_CWDFallback verifies that the cwd-based path is
// returned when NSELF_RUNTIME_DIR is unset.
func TestEmbeddedRuntimeDir_CWDFallback(t *testing.T) {
	t.Setenv("NSELF_RUNTIME_DIR", "")

	wd, _ := os.Getwd()
	got := embeddedRuntimeDir()
	expected := filepath.Join(wd, ".nself", "embedded-pg")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestEmbeddedDSN_DefaultDB verifies that an empty Postgres.DB field defaults
// to "nself" in the returned DSN.
func TestEmbeddedDSN_DefaultDB(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_RUNTIME_DIR", dir)

	cfg := &config.Config{}
	// DB left empty — should default to "nself"
	dsn := embeddedDSN(cfg)

	if !strings.Contains(dsn, "dbname=nself") {
		t.Errorf("expected DSN to contain 'dbname=nself', got %q", dsn)
	}
	if !strings.Contains(dsn, "host="+dir) {
		t.Errorf("expected DSN to contain 'host=%s', got %q", dir, dsn)
	}
	if !strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("expected DSN to contain 'sslmode=disable', got %q", dsn)
	}
}

// TestEmbeddedDSN_CustomDB verifies that cfg.Postgres.DB is used when set.
func TestEmbeddedDSN_CustomDB(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_RUNTIME_DIR", dir)

	cfg := &config.Config{}
	cfg.Postgres.DB = "myapp"
	dsn := embeddedDSN(cfg)

	if !strings.Contains(dsn, "dbname=myapp") {
		t.Errorf("expected DSN to contain 'dbname=myapp', got %q", dsn)
	}
}

// TestEmbeddedBackupDir_CreatesDir verifies that embeddedBackupDir creates the
// directory when it does not yet exist.
func TestEmbeddedBackupDir_CreatesDir(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{}
	cfg.Backup.Dir = filepath.Join(root, "custom-backups")

	dir, err := embeddedBackupDir(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, statErr := os.Stat(dir); statErr != nil {
		t.Errorf("expected backup dir to exist at %q: %v", dir, statErr)
	}
}

// TestEmbeddedBackupDir_DefaultPath verifies that when cfg.Backup.Dir is empty
// the directory falls back to ~/.nself/backups.
func TestEmbeddedBackupDir_DefaultPath(t *testing.T) {
	cfg := &config.Config{}
	// cfg.Backup.Dir is empty — should use ~/.nself/backups

	dir, err := embeddedBackupDir(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".nself", "backups")
	if dir != expected {
		t.Errorf("expected %q, got %q", expected, dir)
	}

	// Clean up the directory that was created.
	defer func() { _ = os.RemoveAll(dir) }()
}
