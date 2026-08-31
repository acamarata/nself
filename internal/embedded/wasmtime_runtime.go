//go:build cgo

package embedded

// CGO_ENABLED=1 is required for this file. The sprint spec (S17 T02) provides
// the Release Plan justification: wasmtime-go/v25 ships prebuilt platform-native
// static libraries (libwasmtime.a) per architecture and there is no pure-Go
// alternative with equivalent WASI support for the embedded-PG path.
//
// OS toolchain requirements:
//   - macOS:  Xcode Command Line Tools (ships libSystem, clang)
//   - Linux:  gcc or clang; glibc >= 2.17; pthreads
//   - Windows: not supported (nSelf embedded-PG is Linux/macOS only)
//
// Build tag for CI: CGO_ENABLED=1 is set in the embedded-pg-matrix.yml workflow.
// All other CLI targets retain CGO_ENABLED=0.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

const (
	// wasmPreopenDir is the directory exposed to pglite WASM via WASI filesystem
	// preopen. pglite writes its Postgres data files here.
	wasmPreopenDir = "pgdata"

	// startTimeout is the maximum time to wait for pglite to signal readiness via
	// the Unix domain socket.
	startTimeout = 30 * time.Second

	// healthCheckInterval is how often Healthy() polls the Unix socket.
	healthCheckInterval = 500 * time.Millisecond

	// healthCheckTimeout is the per-attempt deadline for a health probe.
	healthCheckTimeout = 2 * time.Second
)

// EmbeddedPGRuntime manages the lifecycle of a pglite WASM instance running
// inside wasmtime. It exposes the embedded Postgres via an AF_UNIX socket so
// that Hasura, Auth, and migrations can connect without modification.
//
// The runtime is single-instance: one EmbeddedPGRuntime per nSelf stack.
// Concurrent Start calls are safe; only the first call launches the instance.
type EmbeddedPGRuntime struct {
	mu       sync.Mutex
	engine   *wasmtime.Engine
	store    *wasmtime.Store
	module   *wasmtime.Module
	linker   *wasmtime.Linker
	instance *wasmtime.Instance

	runtimeDir string // e.g. $NSELF_RUNTIME_DIR
	wasmPath   string // absolute path to pglite.wasm
	sockPath   string // AF_UNIX socket path exposed to containers

	// stdoutLog and stderrLog back definePGWasi's fd_write (pg_wasi.go) for
	// fds 1 and 2. They must stay open for the runtime's whole lifetime —
	// __main_argc_argv keeps running in a goroutine after boot() returns —
	// so Stop closes them rather than boot deferring the close.
	stdoutLog *os.File
	stderrLog *os.File

	started bool
	stopped bool
	err     error // sticky error from a failed Start

	// heapBaseCached memoises the dylink-derived heap base so a restart does not
	// re-read and re-parse the 8 MB module. 0 means "not resolved yet" and is
	// also the correct value for a module with no dylink.0 section.
	heapBaseCached uint32
}

// NewEmbeddedPGRuntime constructs an EmbeddedPGRuntime. runtimeDir is the
// directory where the Unix socket and Postgres data dir will be created
// (typically $NSELF_RUNTIME_DIR). wasmPath is the absolute path to pglite.wasm
// as returned by FetchOrCached.
func NewEmbeddedPGRuntime(runtimeDir, wasmPath string) (*EmbeddedPGRuntime, error) {
	if err := os.MkdirAll(filepath.Join(runtimeDir, wasmPreopenDir), 0o700); err != nil {
		return nil, fmt.Errorf("embedded/runtime: create pgdata dir: %w", err)
	}

	cfg := wasmtime.NewConfig()
	cfg.SetWasmSIMD(true)
	cfg.SetConsumeFuel(false) // disable fuel metering for long-running PG

	eng := wasmtime.NewEngineWithConfig(cfg)

	return &EmbeddedPGRuntime{
		engine:     eng,
		runtimeDir: runtimeDir,
		wasmPath:   wasmPath,
		sockPath:   filepath.Join(runtimeDir, "pglite.sock"),
	}, nil
}

// resolveHeapBase returns the address the module's heap must start at, read
// from its own dylink.0 section and memoised for subsequent boots.
//
// A SIDE_MODULE declares how much static memory it occupies; the heap begins
// immediately after that, aligned as the module asks. Getting this wrong does
// not fail loudly — the module instantiates fine and then traps once the
// allocator touches an address derived from a bogus base.
//
// A module with no dylink.0 section is not a SIDE_MODULE and needs no host-side
// placement, so 0 is returned and the global stays where the module put it.
func (r *EmbeddedPGRuntime) resolveHeapBase() (uint32, error) {
	if r.heapBaseCached != 0 {
		return r.heapBaseCached, nil
	}

	raw, err := os.ReadFile(r.wasmPath)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", r.wasmPath, err)
	}

	info, err := parseDylinkMemInfo(raw)
	if errors.Is(err, errNoDylink) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	r.heapBaseCached = info.HeapBase()
	return r.heapBaseCached, nil
}

// SockPath returns the AF_UNIX socket path where the embedded PG is reachable.
// Use this as the host component of the UDS DSN:
//
//	host=<SockPath()> dbname=nself sslmode=disable
func (r *EmbeddedPGRuntime) SockPath() string {
	return r.sockPath
}

// Start boots the pglite WASM module and waits until the embedded PG is ready
// to accept connections. It is idempotent: calling Start on an already-running
// runtime is a no-op.
//
// ctx is used only for the startup timeout; the runtime continues after ctx is
// cancelled. To stop the runtime call Stop.
func (r *EmbeddedPGRuntime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started && r.err == nil {
		return nil // already running
	}
	if r.stopped {
		return fmt.Errorf("embedded/runtime: runtime has been stopped and cannot be restarted")
	}
	if r.err != nil {
		return r.err // propagate sticky error
	}

	if err := r.boot(ctx); err != nil {
		r.err = err
		return err
	}
	r.started = true
	return nil
}
