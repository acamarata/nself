package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadOrCreateInstallIDCreates verifies that LoadOrCreateInstallID creates a
// UUID file on first call and returns a non-empty string.
func TestLoadOrCreateInstallIDCreates(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	id := LoadOrCreateInstallID()
	if id == "" {
		t.Fatal("expected non-empty install-ID")
	}

	// UUID v4 format: 8-4-4-4-12 hex chars
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Errorf("install-ID %q does not look like UUID v4 (want 5 hyphen-separated groups)", id)
	}

	// File must exist with 0600 permissions.
	path := filepath.Join(tmpHome, installIDFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("install-ID file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("install-ID file permissions = %o, want 0600", info.Mode().Perm())
	}
}

// TestLoadOrCreateInstallIDIdempotent verifies that repeated calls return the same ID.
func TestLoadOrCreateInstallIDIdempotent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	id1 := LoadOrCreateInstallID()
	id2 := LoadOrCreateInstallID()
	if id1 != id2 {
		t.Errorf("install-ID changed between calls: %q vs %q", id1, id2)
	}
}

// TestIsEnabledDefault verifies that telemetry is enabled by default (no env, no files).
func TestIsEnabledDefault(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	os.Unsetenv("NSELF_TELEMETRY")
	os.Unsetenv("NSELF_TELEMETRY_OPT_OUT")

	if !IsEnabled() {
		t.Error("expected IsEnabled=true by default (opt-out model)")
	}
}

// TestIsEnabledEnvOverrideZero verifies NSELF_TELEMETRY=0 disables.
func TestIsEnabledEnvOverrideZero(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("NSELF_TELEMETRY", "0")

	if IsEnabled() {
		t.Error("expected IsEnabled=false when NSELF_TELEMETRY=0")
	}
}

// TestIsEnabledEnvOverrideFalse verifies NSELF_TELEMETRY=false disables.
func TestIsEnabledEnvOverrideFalse(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("NSELF_TELEMETRY", "false")

	if IsEnabled() {
		t.Error("expected IsEnabled=false when NSELF_TELEMETRY=false")
	}
}

// TestIsEnabledLegacyOptOut verifies NSELF_TELEMETRY_OPT_OUT=1 disables.
func TestIsEnabledLegacyOptOut(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	os.Unsetenv("NSELF_TELEMETRY")
	t.Setenv("NSELF_TELEMETRY_OPT_OUT", "1")

	if IsEnabled() {
		t.Error("expected IsEnabled=false when NSELF_TELEMETRY_OPT_OUT=1")
	}
}

// TestIsEnabledStatefile verifies the ~/.config/nself/telemetry state file.
func TestIsEnabledStatefile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	os.Unsetenv("NSELF_TELEMETRY")
	os.Unsetenv("NSELF_TELEMETRY_OPT_OUT")

	path := filepath.Join(tmpHome, telemetryStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write "false" — should disable.
	if err := os.WriteFile(path, []byte("false\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if IsEnabled() {
		t.Error("expected IsEnabled=false when statefile=false")
	}

	// Write "true" — should enable.
	if err := os.WriteFile(path, []byte("true\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !IsEnabled() {
		t.Error("expected IsEnabled=true when statefile=true")
	}
}

// TestIsEnabledEnvBeatsStatefile verifies env override wins over statefile.
func TestIsEnabledEnvBeatsStatefile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("NSELF_TELEMETRY", "0")

	// Even with statefile=true, env=0 wins.
	path := filepath.Join(tmpHome, telemetryStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("true\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if IsEnabled() {
		t.Error("expected IsEnabled=false: env override must beat statefile")
	}
}

// TestWriteStatefile verifies WriteStatefile persists correctly and is readable by IsEnabled.
func TestWriteStatefile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	os.Unsetenv("NSELF_TELEMETRY")
	os.Unsetenv("NSELF_TELEMETRY_OPT_OUT")

	if err := WriteStatefile(false); err != nil {
		t.Fatalf("WriteStatefile(false): %v", err)
	}
	if IsEnabled() {
		t.Error("expected IsEnabled=false after WriteStatefile(false)")
	}

	if err := WriteStatefile(true); err != nil {
		t.Fatalf("WriteStatefile(true): %v", err)
	}
	if !IsEnabled() {
		t.Error("expected IsEnabled=true after WriteStatefile(true)")
	}

	// Verify file permissions are 0600.
	path := filepath.Join(tmpHome, telemetryStateFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat statefile: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("statefile permissions = %o, want 0600", info.Mode().Perm())
	}
}
