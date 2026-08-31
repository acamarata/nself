//go:build cgo

package embedded

// emscripten_abi_runtime.go — Emscripten process/runtime control, memory
// management, and dynamic-linking imports for pglite v0.2.17 (groups 4a-4c
// of defineEmscriptenABI). Split from emscripten_abi.go (T-P6-E2-W1-S1-T3).
// Inputs:  *wasmtime.Linker, *wasmtime.Store, and the env::memory object
//          created by defineEmscriptenGlobalsMemoryTable (needed by
//          emscripten_resize_heap).
// Outputs: error — first definition failure aborts.
// Constraints: pure move, same values/order, no behavior change.

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// defineEmscriptenRuntimeFuncs registers the process/runtime control,
// memory management, and dynamic-linking env:: function imports.
func defineEmscriptenRuntimeFuncs(linker *wasmtime.Linker, store *wasmtime.Store, mem *wasmtime.Memory) error {
	// ── 4a. Process / runtime control ────────────────────────────────────────

	// exit — env::exit(code i32). pglite calls this on graceful shutdown
	// (code=0). Non-zero exits are surfaced as panics so wasmtime can trap them.
	if err := linker.DefineFunc(store, "env", "exit", func(code int32) {
		if code != 0 {
			panic(fmt.Sprintf("pglite: exit(%d)", code))
		}
		// code == 0: graceful shutdown; the goroutine running _start returns.
	}); err != nil {
		return fmt.Errorf("embedded/abi: exit: %w", err)
	}

	// emscripten_force_exit — triggers a process-level exit. We panic so
	// wasmtime surfaces the trap rather than silently discarding the call.
	if err := linker.DefineFunc(store, "env", "emscripten_force_exit", func(code int32) {
		panic(fmt.Sprintf("pglite: emscripten_force_exit(%d)", code))
	}); err != nil {
		return fmt.Errorf("embedded/abi: emscripten_force_exit: %w", err)
	}

	// _abort_js — called on assertion failures inside the C runtime.
	if err := linker.DefineFunc(store, "env", "_abort_js", func() {
		panic("pglite: _abort_js called")
	}); err != nil {
		return fmt.Errorf("embedded/abi: _abort_js: %w", err)
	}

	// __assert_fail — C assert() handler.
	if err := linker.DefineFunc(store, "env", "__assert_fail",
		func(condition, filename, line, fn int32) {
			panic(fmt.Sprintf("pglite: assert failed cond=%d file=%d line=%d fn=%d",
				condition, filename, line, fn))
		}); err != nil {
		return fmt.Errorf("embedded/abi: __assert_fail: %w", err)
	}

	// _emscripten_throw_longjmp — longjmp from C exception handling. We trap.
	if err := linker.DefineFunc(store, "env", "_emscripten_throw_longjmp", func() {
		panic("pglite: _emscripten_throw_longjmp")
	}); err != nil {
		return fmt.Errorf("embedded/abi: _emscripten_throw_longjmp: %w", err)
	}

	// __call_sighandler — invokes a registered C signal handler. No-op shim;
	// pglite does not use signals in the embedded path.
	if err := linker.DefineFunc(store, "env", "__call_sighandler",
		func(handler, sig int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: __call_sighandler: %w", err)
	}

	// is_web_env — returns 0 (not a browser environment).
	if err := linker.DefineFunc(store, "env", "is_web_env", func() int32 {
		return 0
	}); err != nil {
		return fmt.Errorf("embedded/abi: is_web_env: %w", err)
	}

	// _emscripten_runtime_keepalive_clear — browser-specific async keepalive.
	// No-op in WASI context.
	if err := linker.DefineFunc(store, "env", "_emscripten_runtime_keepalive_clear", func() {}); err != nil {
		return fmt.Errorf("embedded/abi: _emscripten_runtime_keepalive_clear: %w", err)
	}

	// ── 4b. Memory management ─────────────────────────────────────────────

	// emscripten_resize_heap — attempts to grow linear memory. Returns 1 on
	// success, 0 if the requested size cannot be satisfied.
	if err := linker.DefineFunc(store, "env", "emscripten_resize_heap",
		func(requestedSize int32) int32 {
			// requestedSize is the total desired heap size in bytes.
			// Memory page = 64 KB; compute the additional pages needed.
			currentBytes := int64(mem.DataSize(store))
			needed := int64(requestedSize) - currentBytes
			if needed <= 0 {
				return 1 // already large enough
			}
			pages := uint64((needed + 65535) / 65536)
			if _, err := mem.Grow(store, pages); err != nil {
				return 0 // growth failed
			}
			return 1
		}); err != nil {
		return fmt.Errorf("embedded/abi: emscripten_resize_heap: %w", err)
	}

	// _emscripten_memcpy_js — bulk memory copy host helper. We rely on the
	// WASM bulk-memory proposal (enabled in engine config), so this stub
	// should not be called in practice; return 0 signals success.
	if err := linker.DefineFunc(store, "env", "_emscripten_memcpy_js",
		func(dst, src, n int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: _emscripten_memcpy_js: %w", err)
	}

	// _mmap_js — mmap via JS glue. Not supported in native WASI path.
	// Actual WASM signature: (i32, i32, i32, i32, i64, i32, i32) -> (i32)
	// The offset parameter is i64; addr is the last i32 (output pointer).
	if err := linker.DefineFunc(store, "env", "_mmap_js",
		func(length, prot, flags, fd int32, offset int64, allocated, addr int32) int32 {
			return enosys
		}); err != nil {
		return fmt.Errorf("embedded/abi: _mmap_js: %w", err)
	}

	// _munmap_js — munmap. Not supported; returns ENOSYS.
	// Actual WASM signature: (i32, i32, i32, i32, i32, i64) -> (i32)
	if err := linker.DefineFunc(store, "env", "_munmap_js",
		func(a0, a1, a2, a3, a4 int32, a5 int64) int32 {
			return enosys
		}); err != nil {
		return fmt.Errorf("embedded/abi: _munmap_js: %w", err)
	}

	// ── 4c. Dynamic linking ──────────────────────────────────────────────

	// _dlopen_js — dynamic library open. Not supported; return 0 (NULL).
	// Actual WASM signature: (i32) -> (i32) — only one param (flags/handle).
	if err := linker.DefineFunc(store, "env", "_dlopen_js",
		func(flags int32) int32 {
			return 0
		}); err != nil {
		return fmt.Errorf("embedded/abi: _dlopen_js: %w", err)
	}

	// _dlsym_js — symbol lookup. Not supported; return 0 (NULL).
	// Actual WASM signature: (i32, i32, i32) -> (i32).
	if err := linker.DefineFunc(store, "env", "_dlsym_js",
		func(handle, symbol, a2 int32) int32 {
			return 0
		}); err != nil {
		return fmt.Errorf("embedded/abi: _dlsym_js: %w", err)
	}

	return nil
}
