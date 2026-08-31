//go:build cgo

package embedded

// Purpose: boot sequence and lifecycle helpers (boot, module loading, socket wait, stop, health check) for the embedded-PG wasmtime runtime.
// Inputs: an EmbeddedPGRuntime constructed by NewEmbeddedPGRuntime in wasmtime_runtime.go.
// Outputs: a running (or stopped) WASM Postgres process communicating over a Unix socket.
// Constraints: split out of wasmtime_runtime.go as a pure move (CLI-R12); no behavior change. Requires CGO_ENABLED=1 (see wasmtime_runtime.go header for rationale).

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

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

	// pglite v0.2.17 is compiled with Emscripten and requires all 113 env::
	// host imports to be defined before instantiation, PLUS 12
	// wasi_snapshot_preview1 imports (defined below by definePGWasi, not
	// wasmtime's own linker.DefineWasi()). defineEmscriptenABI defines
	// env::exit, invoke_* trampolines, __syscall_* stubs, globals, memory,
	// and the indirect function table, and returns the created env::memory
	// object so definePGWasi can bind to the exact same linear memory.
	mem, err := defineEmscriptenABI(linker, store)
	if err != nil {
		return fmt.Errorf("embedded/runtime: Emscripten ABI: %w", err)
	}

	// pglite v0.2.17 also imports GOT.mem namespace globals used by Emscripten's
	// dynamic-linking GOT relocation: GOT.mem::__heap_base.
	//
	// Under Emscripten's JS loader a dynamic linker writes these before any
	// module code runs. wasmtime has no dynamic linker, so the host must supply
	// the final value or the module keeps whatever it was given — see
	// defineGOTNamespaces and dylink.go.
	heapBase, hbErr := r.resolveHeapBase()
	if hbErr != nil {
		return fmt.Errorf("embedded/runtime: resolve heap base: %w", hbErr)
	}
	if err := defineGOTNamespaces(linker, store, heapBase); err != nil {
		return fmt.Errorf("embedded/runtime: GOT namespaces: %w", err)
	}

	// wasmtime's linker.DefineWasi() resolves guest memory via the
	// INSTANCE's exported "memory" — pglite is a SIDE_MODULE that imports
	// env::memory and exports nothing named "memory", so DefineWasi()'s
	// functions trap with "missing required memory export" on first call
	// (see pg_wasi.go header for the full diagnosis). definePGWasi replaces
	// it entirely, binding the 12 wasi_snapshot_preview1 functions pglite
	// actually imports directly to mem.
	stdoutLog, err := os.OpenFile(filepath.Join(r.runtimeDir, "pglite-stdout.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("embedded/runtime: open stdout log: %w", err)
	}
	r.stdoutLog = stdoutLog
	stderrLog, err := os.OpenFile(filepath.Join(r.runtimeDir, "pglite-stderr.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("embedded/runtime: open stderr log: %w", err)
	}
	r.stderrLog = stderrLog

	if err := definePGWasi(linker, store, mem, stdoutLog, stderrLog); err != nil {
		return fmt.Errorf("embedded/runtime: WASI: %w", err)
	}

	r.linker = linker

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
	// Report an early pglite exit over a channel, NOT by taking r.mu.
	//
	// boot() runs with r.mu already held by Start(). If this goroutine took r.mu
	// on its error path it would block forever against its own caller, the error
	// would never be recorded, and boot() would sit in waitForSocket until the
	// context deadline — turning a pglite crash that is knowable in seconds into
	// a full-timeout failure with no diagnostic. That deadlock is exactly what
	// wedged the Embedded PG Integration job: every test failed at its bound
	// (2100s / 90s / 90s) and the panic dump showed goroutine 26 in waitForSocket
	// while the goroutine spawned here sat in sync.Mutex.Lock.
	//
	// Buffered so this send never blocks even if nobody is listening (e.g. the
	// socket came up fine and pglite exited later).
	exitCh := make(chan error, 1)
	go func() {
		if _, err := mainFn.Func().Call(store, int32(0), int32(0)); err != nil {
			// __main_argc_argv never returns under normal operation; an error here
			// means the Postgres process exited unexpectedly.
			exitCh <- fmt.Errorf("embedded/runtime: pglite exited unexpectedly: %w", err)
			return
		}
		exitCh <- fmt.Errorf("embedded/runtime: pglite main returned unexpectedly without error")
	}()

	// Wait for the socket to become available.
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(startTimeout)
	}
	waitCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	return r.waitForSocket(waitCtx, exitCh)
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
				_ = tmp.Close()
				_ = os.Rename(tmp.Name(), cachePath)
			} else {
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
			}
		}
	}

	return mod, nil
}

// waitForSocket polls the Unix domain socket until it accepts a connection,
// ctx expires, or pglite exits early.
//
// exitCh carries an early pglite exit. Selecting on it is what makes a failed
// boot fail FAST with the real cause ("pglite exited unexpectedly: ...") instead
// of polling a socket that will never appear and reporting a bare timeout when
// the deadline expires. Pass a nil channel to disable that path.
func (r *EmbeddedPGRuntime) waitForSocket(ctx context.Context, exitCh <-chan error) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("embedded/runtime: timed out waiting for pglite socket at %s: %w", r.sockPath, ctx.Err())
		case err := <-exitCh:
			// pglite died before the socket appeared — no point polling further.
			return err
		default:
		}

		dialCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "unix", r.sockPath)
		cancel()
		if err == nil {
			_ = conn.Close()
			return nil // ready
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("embedded/runtime: timed out waiting for pglite socket: %w", ctx.Err())
		case err := <-exitCh:
			return err
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

	if r.stdoutLog != nil {
		_ = r.stdoutLog.Close()
	}
	if r.stderrLog != nil {
		_ = r.stderrLog.Close()
	}

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
	_ = conn.Close()
	return true
}
