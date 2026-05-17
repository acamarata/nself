package truststate

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoad_ReadError covers the branch where os.ReadFile fails with a non-NotExist
// error (e.g., the file exists but is a directory — produces EISDIR).
func TestLoad_ReadError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a directory at the path where the state file would live.
	// Reading a directory triggers a non-NotExist error.
	statePath := filepath.Join(home, ".nself", "trust-state.json")
	if err := os.MkdirAll(statePath, 0755); err != nil {
		t.Fatalf("MkdirAll to create dir-at-file-path: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load should error when state path is a directory, got nil")
	}
}

// TestSave_MkdirAllFail covers the Save branch where MkdirAll fails.
// We create a regular file at the directory path to block MkdirAll.
func TestSave_MkdirAllFail(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a file at ~/.nself so MkdirAll on it will fail.
	nselfPath := filepath.Join(home, ".nself")
	if err := os.WriteFile(nselfPath, []byte("block"), 0600); err != nil {
		t.Fatalf("WriteFile to block dir creation: %v", err)
	}

	err := Save(TrustState{TrustedDomain: "test.local"})
	if err == nil {
		t.Error("Save should error when MkdirAll is blocked, got nil")
	}
}

// TestSave_FilePermissions verifies the saved file is written with mode 0600.
func TestSave_FilePermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Save(TrustState{TrustedDomain: "perm.test"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	statePath := filepath.Join(home, ".nself", "trust-state.json")
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected file permissions 0600, got %04o", perm)
	}
}

// TestStatePath_UserHomeDirError covers the UserHomeDir error branch in statePath.
// Setting HOME="" causes os.UserHomeDir() to return "$HOME is not defined" on macOS.
// Both Load() and Save() call statePath(), so we exercise them both here to hit
// the "statePath() fails" branches in Load and Save as well.
func TestLoad_StatePathError(t *testing.T) {
	t.Setenv("HOME", "")
	_, err := Load()
	if err == nil {
		t.Error("Load should error when HOME is empty, got nil")
	}
}

func TestSave_StatePathError(t *testing.T) {
	t.Setenv("HOME", "")
	err := Save(TrustState{TrustedDomain: "no-home.test"})
	if err == nil {
		t.Error("Save should error when HOME is empty, got nil")
	}
}

// TestLoad_UnmarshalError covers the json.Unmarshal error branch.
// Writing invalid JSON to the state file triggers this.
func TestLoad_UnmarshalError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create the .nself directory and write invalid JSON to trust-state.json.
	nselfDir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(nselfDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	stateFile := filepath.Join(nselfDir, "trust-state.json")
	if err := os.WriteFile(stateFile, []byte("not valid json {{{"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load should error when state file contains invalid JSON, got nil")
	}
}

// TestSave_WriteFileError covers the os.WriteFile error branch in Save.
// We make the .nself directory read-only so the file cannot be created inside it.
func TestSave_WriteFileError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create ~/.nself with mode 0555 (no write). MkdirAll succeeds (dir exists) but
	// WriteFile inside it fails with EACCES.
	nselfDir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(nselfDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Chmod(nselfDir, 0555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(nselfDir, 0755) })

	err := Save(TrustState{TrustedDomain: "write-fail.test"})
	if err == nil {
		t.Error("Save should error when .nself dir is read-only, got nil")
	}
}

// TestLoad_EmptyDomain verifies an empty trusted domain round-trips correctly.
func TestLoad_EmptyDomain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Save(TrustState{TrustedDomain: ""}); err != nil {
		t.Fatalf("Save empty domain: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.TrustedDomain != "" {
		t.Errorf("expected empty domain, got %q", got.TrustedDomain)
	}
}
