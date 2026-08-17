//go:build cgo

package embedded

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
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

// TestWaitForSocket_EarlyExitFailsFast is a regression test for the boot
// deadlock that wedged the Embedded PG Integration job.
//
// boot() runs with r.mu held by Start(). The goroutine running pglite used to
// report an early exit by taking r.mu, which blocked forever against its own
// caller; the error was never recorded and boot() polled for a socket that
// would never appear until the context deadline. Every test in the package then
// failed at its own bound (2100s / 90s / 90s) with a bare timeout and no
// diagnostic, and the panic dump showed goroutine 26 parked in waitForSocket
// while the spawned goroutine sat in sync.Mutex.Lock.
//
// The exit is now delivered over a channel that waitForSocket selects on, so a
// failed boot surfaces the real cause immediately.
func TestWaitForSocket_EarlyExitFailsFast(t *testing.T) {
	r := &EmbeddedPGRuntime{sockPath: filepath.Join(t.TempDir(), "nonexistent.sock")}

	exitCh := make(chan error, 1)
	sentinel := errors.New("pglite exited unexpectedly: boom")
	exitCh <- sentinel

	// A generous deadline: if the fix regresses, this blocks the full duration
	// instead of returning at once, and the elapsed-time assertion catches it.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	err := r.waitForSocket(ctx, exitCh)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected the pglite exit error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected the real pglite cause to surface, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("waitForSocket took %v — it should fail fast on early exit, not poll to the deadline", elapsed)
	}
}

// TestWaitForSocket_NilExitChannelStillTimesOut proves the exit path is additive:
// with no exit channel the original timeout behaviour is unchanged.
func TestWaitForSocket_NilExitChannelStillTimesOut(t *testing.T) {
	r := &EmbeddedPGRuntime{sockPath: filepath.Join(t.TempDir(), "nonexistent.sock")}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	if err := r.waitForSocket(ctx, nil); err == nil {
		t.Fatal("expected a timeout error when the socket never appears")
	}
}
