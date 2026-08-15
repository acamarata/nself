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
	"fmt"
	"net"
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

	started bool
	stopped bool
	err     error // sticky error from a failed Start
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

// boot loads, instantiates, and starts the WASM module. It must be called with
// r.mu held.
func (r *EmbeddedPGRuntime) boot(ctx context.Context) error {
	// Attempt to deserialize a cached compiled module for fast cold-start.
	// Falls back to compiling from the WASM binary if the cached version is
	// absent or incompatible.
	// loadOrCompileModule is CPU-bound and uninterruptible: on a cold cache it
	// runs a full wasmtime compile of the pglite Postgres WASM, which can take
	// minutes on small CI runners. Run it under the caller's context so a boot
	// that exceeds the deadline returns a clear error instead of blocking past
	// every timeout and taking the whole process down with a test panic.
	type modResult struct {
		mod *wasmtime.Module
		err error
	}
	modCh := make(chan modResult, 1)
	go func() {
		m, err := r.loadOrCompileModule()
		modCh <- modResult{mod: m, err: err}
	}()

	var module *wasmtime.Module
	select {
	case <-ctx.Done():
		// The compile goroutine is left to finish and populate the on-disk
		// compiled cache, so a subsequent boot can reuse it rather than
		// repeating the work.
		return fmt.Errorf("embedded/runtime: timed out compiling pglite WASM (cold compile still running in background, its cache will speed up the next boot): %w", ctx.Err())
	case res := <-modCh:
		if res.err != nil {
			return fmt.Errorf("embedded/runtime: load module: %w", res.err)
		}
		module = res.mod
	}
	r.module = module

	store := wasmtime.NewStore(r.engine)
	r.store = store

	linker := wasmtime.NewLinker(r.engine)
	if err := linker.DefineWasi(); err != nil {
		return fmt.Errorf("embedded/runtime: define WASI: %w", err)
	}

	// pglite v0.2.17 is compiled with Emscripten and requires all 113 env::
	// host imports to be defined before instantiation. DefineWasi() only covers
	// WASI preview-1; defineEmscriptenABI defines the remaining Emscripten
	// symbols (env::exit, invoke_* trampolines, __syscall_* stubs, globals,
	// memory, and the indirect function table).
	if err := defineEmscriptenABI(linker, store); err != nil {
		return fmt.Errorf("embedded/runtime: Emscripten ABI: %w", err)
	}

	// pglite v0.2.17 also imports GOT.mem namespace globals used by
	// Emscripten's dynamic-linking GOT relocation. These are mutable i32
	// globals whose values are written by the dynamic linker at startup.
	// Confirmed imports (via WebAssembly.Module.imports): GOT.mem::__heap_base.
	if err := defineGOTNamespaces(linker, store); err != nil {
		return fmt.Errorf("embedded/runtime: GOT namespaces: %w", err)
	}

	r.linker = linker

	// Configure WASI: preopened filesystem, environment.
	wasiCfg := wasmtime.NewWasiConfig()
	wasiCfg.SetStdoutFile(filepath.Join(r.runtimeDir, "pglite-stdout.log"))
	wasiCfg.SetStderrFile(filepath.Join(r.runtimeDir, "pglite-stderr.log"))
	wasiCfg.PreopenDir(filepath.Join(r.runtimeDir, wasmPreopenDir), "/data")

	store.SetWasi(wasiCfg)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("embedded/runtime: context expired before instantiate: %w", err)
	}
	instance, err := linker.Instantiate(store, module)
	if err != nil {
		return fmt.Errorf("embedded/runtime: instantiate: %w", err)
	}
	r.instance = instance

	// pglite v0.2.17 is compiled as an Emscripten SIDE_MODULE and does NOT
	// export _start (the WASI command-model entry point). Instead it has:
	//   - A WASM start section (called automatically by wasmtime during Instantiate)
	//     that runs the C runtime constructors (__wasm_call_ctors).
	//   - __main_argc_argv — the actual C main() entry point that starts Postgres.
	//
	// We call __main_argc_argv in a goroutine; it blocks indefinitely while
	// Postgres serves connections. argc=0, argv=0 is sufficient — pglite reads
	// its config from the WASI filesystem preopens, not from command-line args.
	mainFn := instance.GetExport(store, "__main_argc_argv")
	if mainFn == nil {
		return fmt.Errorf("embedded/runtime: pglite WASM missing __main_argc_argv export")
	}
	go func() {
		if _, err := mainFn.Func().Call(store, int32(0), int32(0)); err != nil {
			// __main_argc_argv never returns under normal operation; an error here
			// means the Postgres process exited unexpectedly.
			r.mu.Lock()
			if !r.stopped {
				r.err = fmt.Errorf("embedded/runtime: pglite exited unexpectedly: %w", err)
			}
			r.mu.Unlock()
		}
	}()

	// Wait for the socket to become available.
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(startTimeout)
	}
	waitCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	return r.waitForSocket(waitCtx)
}

// loadOrCompileModule tries to deserialize a pre-compiled module from the
// serialized cache path. On cache miss or engine incompatibility it compiles
// from the WASM binary and writes a new cache entry.
func (r *EmbeddedPGRuntime) loadOrCompileModule() (*wasmtime.Module, error) {
	cachePath := r.wasmPath + ".compiled"

	if data, err := os.ReadFile(cachePath); err == nil {
		mod, err := wasmtime.NewModuleDeserialize(r.engine, data)
		if err == nil {
			return mod, nil // cache hit
		}
		// Cache incompatible with current engine config — recompile.
		_ = os.Remove(cachePath)
	}

	// Compile from source WASM.
	data, err := os.ReadFile(r.wasmPath)
	if err != nil {
		return nil, fmt.Errorf("read pglite.wasm: %w", err)
	}
	mod, err := wasmtime.NewModule(r.engine, data)
	if err != nil {
		return nil, fmt.Errorf("compile pglite.wasm: %w", err)
	}

	// Serialize and cache for future cold-starts.
	if serialized, err := mod.Serialize(); err == nil {
		tmp, err := os.CreateTemp(filepath.Dir(cachePath), ".pglite-compiled-*")
		if err == nil {
			if _, werr := tmp.Write(serialized); werr == nil {
				tmp.Close()
				_ = os.Rename(tmp.Name(), cachePath)
			} else {
				tmp.Close()
				_ = os.Remove(tmp.Name())
			}
		}
	}

	return mod, nil
}

// waitForSocket polls the Unix domain socket until it accepts a connection or
// ctx expires.
func (r *EmbeddedPGRuntime) waitForSocket(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("embedded/runtime: timed out waiting for pglite socket at %s: %w", r.sockPath, ctx.Err())
		default:
		}

		dialCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "unix", r.sockPath)
		cancel()
		if err == nil {
			conn.Close()
			return nil // ready
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("embedded/runtime: timed out waiting for pglite socket: %w", ctx.Err())
		case <-time.After(healthCheckInterval):
		}
	}
}

// Stop shuts down the embedded PG runtime and removes the Unix socket file.
// It is safe to call Stop multiple times; only the first call takes effect.
func (r *EmbeddedPGRuntime) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return nil
	}
	r.stopped = true

	_ = os.Remove(r.sockPath)

	// wasmtime does not have a graceful shutdown path for an instantiated WASM
	// module. The goroutine running _start will be abandoned; the process will
	// clean up when it exits. This is acceptable for a CLI tool.
	return nil
}

// Healthy reports whether the embedded PG is currently accepting connections
// on its Unix socket. It returns false if the runtime has not been started or
// has been stopped.
func (r *EmbeddedPGRuntime) Healthy() bool {
	r.mu.Lock()
	started := r.started
	stopped := r.stopped
	sockPath := r.sockPath
	r.mu.Unlock()

	if !started || stopped {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", sockPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
