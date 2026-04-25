package truststate

import (
	"os"
	"path/filepath"
	"testing"
)

// overrideHome temporarily overrides UserHomeDir by pointing $HOME to dir.
func overrideHome(t *testing.T, dir string) {
	t.Helper()
	orig := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	_ = orig // keep for clarity
}

// TestSaveAndLoad_RoundTrip verifies that a TrustState written by Save can be
// read back by Load with the same domain.
func TestSaveAndLoad_RoundTrip(t *testing.T) {
	home := t.TempDir()
	overrideHome(t, home)

	state := TrustState{TrustedDomain: "local.nself.dev"}
	if err := Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.TrustedDomain != state.TrustedDomain {
		t.Errorf("TrustedDomain: got %q, want %q", got.TrustedDomain, state.TrustedDomain)
	}
}

// TestLoad_NotFound returns zero-value TrustState when no state file exists.
func TestLoad_NotFound(t *testing.T) {
	home := t.TempDir()
	overrideHome(t, home)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if got.TrustedDomain != "" {
		t.Errorf("expected empty TrustedDomain for missing file, got %q", got.TrustedDomain)
	}
}

// TestLoad_CorruptJSON verifies that corrupted JSON returns an error.
func TestLoad_CorruptJSON(t *testing.T) {
	home := t.TempDir()
	overrideHome(t, home)

	statePath := filepath.Join(home, ".nself", "trust-state.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("NOT JSON"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load with corrupt JSON should return error, got nil")
	}
}

// TestSave_CreatesDirectory verifies that Save creates ~/.nself/ if it does
// not exist.
func TestSave_CreatesDirectory(t *testing.T) {
	home := t.TempDir()
	overrideHome(t, home)

	// Verify directory doesn't exist yet.
	nselfDir := filepath.Join(home, ".nself")
	if _, err := os.Stat(nselfDir); !os.IsNotExist(err) {
		t.Skip("~/.nself already exists in temp home — skipping directory creation test")
	}

	if err := Save(TrustState{TrustedDomain: "test.local"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(nselfDir); err != nil {
		t.Errorf("Save did not create ~/.nself directory: %v", err)
	}
}

// TestSave_OverwritesPreviousState verifies that saving twice keeps the latest.
func TestSave_OverwritesPreviousState(t *testing.T) {
	home := t.TempDir()
	overrideHome(t, home)

	if err := Save(TrustState{TrustedDomain: "first.local"}); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := Save(TrustState{TrustedDomain: "second.local"}); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.TrustedDomain != "second.local" {
		t.Errorf("expected second.local after overwrite, got %q", got.TrustedDomain)
	}
}
