package database

import (
	"os"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// TestEmbeddedPGDatabaseURL_Default verifies the UDS DSN when no runtime dir env
// var is set. Falls back to $PWD/.nself/embedded-pg.
func TestEmbeddedPGDatabaseURL_Default(t *testing.T) {
	t.Setenv("NSELF_RUNTIME_DIR", "")
	cfg := &config.Config{}
	cfg.Postgres.DB = "nself"

	runtimeDir := embeddedPGRuntimeDir(cfg)
	dsn := cfg.EmbeddedPGDatabaseURL(runtimeDir)

	if !strings.Contains(dsn, "host=") {
		t.Errorf("expected DSN to contain 'host=', got %q", dsn)
	}
	if !strings.Contains(dsn, "dbname=nself") {
		t.Errorf("expected DSN to contain 'dbname=nself', got %q", dsn)
	}
	if !strings.Contains(dsn, "sslmode=disable") {
		t.Errorf("expected DSN to contain 'sslmode=disable', got %q", dsn)
	}
	if !strings.Contains(dsn, ".nself/embedded-pg") {
		t.Errorf("expected DSN host to include '.nself/embedded-pg', got %q", dsn)
	}
}

// TestEmbeddedPGDatabaseURL_RuntimeDirEnv verifies the UDS DSN uses
// NSELF_RUNTIME_DIR when it is set.
func TestEmbeddedPGDatabaseURL_RuntimeDirEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_RUNTIME_DIR", dir)

	cfg := &config.Config{}
	cfg.Postgres.DB = "mydb"

	runtimeDir := embeddedPGRuntimeDir(cfg)
	if runtimeDir != dir {
		t.Errorf("embeddedPGRuntimeDir: expected %q, got %q", dir, runtimeDir)
	}

	dsn := cfg.EmbeddedPGDatabaseURL(runtimeDir)
	if !strings.Contains(dsn, "host="+dir) {
		t.Errorf("expected DSN host=%s, got %q", dir, dsn)
	}
	if !strings.Contains(dsn, "dbname=mydb") {
		t.Errorf("expected DSN dbname=mydb, got %q", dsn)
	}
}

// TestEmbeddedPGDatabaseURL_DefaultDB verifies the UDS DSN uses "nself" when
// cfg.Postgres.DB is empty.
func TestEmbeddedPGDatabaseURL_DefaultDB(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_RUNTIME_DIR", dir)

	cfg := &config.Config{}
	// DB left empty — should default to "nself"

	dsn := cfg.EmbeddedPGDatabaseURL(dir)
	if !strings.Contains(dsn, "dbname=nself") {
		t.Errorf("expected fallback dbname=nself, got %q", dsn)
	}
}

// TestEmbeddedPGRuntimeDir_EnvWins verifies env var wins over cwd-based default.
func TestEmbeddedPGRuntimeDir_EnvWins(t *testing.T) {
	dir := "/tmp/test-runtime"
	t.Setenv("NSELF_RUNTIME_DIR", dir)

	cfg := &config.Config{}
	got := embeddedPGRuntimeDir(cfg)
	if got != dir {
		t.Errorf("expected %q, got %q", dir, got)
	}
}

// TestEmbeddedPGRuntimeDir_CWDFallback verifies the cwd-based fallback when
// NSELF_RUNTIME_DIR is unset.
func TestEmbeddedPGRuntimeDir_CWDFallback(t *testing.T) {
	t.Setenv("NSELF_RUNTIME_DIR", "")
	cfg := &config.Config{}

	wd, _ := os.Getwd()
	got := embeddedPGRuntimeDir(cfg)
	expected := wd + "/.nself/embedded-pg"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
