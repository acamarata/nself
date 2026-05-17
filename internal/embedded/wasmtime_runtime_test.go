package embedded

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewEmbeddedPGRuntime_CreatesDataDir verifies that NewEmbeddedPGRuntime
// creates the pgdata subdirectory inside runtimeDir.
func TestNewEmbeddedPGRuntime_CreatesDataDir(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "pglite.wasm")
	// Write a minimal placeholder so os.Stat succeeds (we won't actually boot).
	if err := os.WriteFile(wasmPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	rt, err := NewEmbeddedPGRuntime(dir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil runtime")
	}

	dataDir := filepath.Join(dir, wasmPreopenDir)
	fi, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("pgdata dir not created: %v", err)
	}
	if !fi.IsDir() {
		t.Errorf("%s is not a directory", dataDir)
	}
}

// TestEmbeddedPGRuntime_SockPath verifies that SockPath returns the expected
// Unix socket path derived from runtimeDir.
func TestEmbeddedPGRuntime_SockPath(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "pglite.wasm")
	_ = os.WriteFile(wasmPath, []byte("placeholder"), 0o600)

	rt, err := NewEmbeddedPGRuntime(dir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}

	want := filepath.Join(dir, "pglite.sock")
	if got := rt.SockPath(); got != want {
		t.Errorf("SockPath() = %q, want %q", got, want)
	}
}

// TestEmbeddedPGRuntime_Healthy_BeforeStart verifies that Healthy returns false
// before the runtime has been started.
func TestEmbeddedPGRuntime_Healthy_BeforeStart(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "pglite.wasm")
	_ = os.WriteFile(wasmPath, []byte("placeholder"), 0o600)

	rt, err := NewEmbeddedPGRuntime(dir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}

	if rt.Healthy() {
		t.Error("Healthy() should return false before Start")
	}
}

// TestEmbeddedPGRuntime_StopIdempotent verifies that calling Stop multiple times
// does not return an error.
func TestEmbeddedPGRuntime_StopIdempotent(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "pglite.wasm")
	_ = os.WriteFile(wasmPath, []byte("placeholder"), 0o600)

	rt, err := NewEmbeddedPGRuntime(dir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}

	if err := rt.Stop(); err != nil {
		t.Errorf("first Stop: %v", err)
	}
	if err := rt.Stop(); err != nil {
		t.Errorf("second Stop (idempotent): %v", err)
	}
}

// TestEmbeddedPGRuntime_StartAfterStop verifies that Start returns an error
// after Stop has been called.
func TestEmbeddedPGRuntime_StartAfterStop(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "pglite.wasm")
	_ = os.WriteFile(wasmPath, []byte("placeholder"), 0o600)

	rt, err := NewEmbeddedPGRuntime(dir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}

	if err := rt.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	// Start should fail because the runtime has been stopped.
	if err := rt.Start(t.Context()); err == nil {
		t.Error("Start after Stop should return an error")
	}
}
