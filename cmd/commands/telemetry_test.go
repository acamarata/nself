package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// setupTelemetryTestHome creates a temp home dir and configures UserHomeDir to use it.
// Returns a cleanup function.
func setupTelemetryTestHome(t *testing.T) (homeDir string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir) // os.UserHomeDir reads $HOME on Unix
	if runtime.GOOS == "windows" {
		// os.UserHomeDir() reads USERPROFILE on Windows, not HOME.
		t.Setenv("USERPROFILE", dir)
	}
	return dir, func() {} // t.Setenv handles cleanup automatically
}

// TestTelemetryStatusShowsCorrectState verifies that status command reports the right state.
func TestTelemetryStatusShowsCorrectState(t *testing.T) {
	_, cleanup := setupTelemetryTestHome(t)
	defer cleanup()
	os.Unsetenv("NSELF_TELEMETRY_OPT_OUT")

	// Default: enabled from default source
	pref := config.GetTelemetryPreference()
	if !pref.Enabled {
		t.Error("expected default telemetry to be enabled")
	}
	if pref.Source != "default" {
		t.Errorf("expected source=default, got %q", pref.Source)
	}
}

// TestTelemetryOffPersistsPreference verifies that SetTelemetryEnabled(false) stores the value.
func TestTelemetryOffPersistsPreference(t *testing.T) {
	homeDir, cleanup := setupTelemetryTestHome(t)
	defer cleanup()
	os.Unsetenv("NSELF_TELEMETRY_OPT_OUT")

	if err := config.SetTelemetryEnabled(false); err != nil {
		t.Fatalf("SetTelemetryEnabled(false) failed: %v", err)
	}

	// Verify file was created and contains the right value
	confPath := filepath.Join(homeDir, ".nself", "config.toml")
	if _, err := os.Stat(confPath); err != nil {
		t.Fatalf("expected config file at %s, got error: %v", confPath, err)
	}

	pref := config.GetTelemetryPreference()
	if pref.Enabled {
		t.Error("expected telemetry to be disabled after SetTelemetryEnabled(false)")
	}
	if pref.Source != "config" {
		t.Errorf("expected source=config, got %q", pref.Source)
	}
}

// TestTelemetryEnvOverride verifies that NSELF_TELEMETRY_OPT_OUT=1 overrides the config file.
func TestTelemetryEnvOverride(t *testing.T) {
	_, cleanup := setupTelemetryTestHome(t)
	defer cleanup()

	// Persist enabled=true to config
	if err := config.SetTelemetryEnabled(true); err != nil {
		t.Fatalf("SetTelemetryEnabled(true) failed: %v", err)
	}

	// Set env override
	t.Setenv("NSELF_TELEMETRY_OPT_OUT", "1")

	pref := config.GetTelemetryPreference()
	if pref.Enabled {
		t.Error("expected env override to disable telemetry despite config enabled=true")
	}
	if pref.Source != "env" {
		t.Errorf("expected source=env, got %q", pref.Source)
	}
}
