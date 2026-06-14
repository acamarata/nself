//go:build cgo

package embedded

// emscripten_abi.go — Emscripten host-import shims for pglite v0.2.17.
//
// Purpose:
//   pglite v0.2.17 is compiled with Emscripten (emcc) targeting WASI. It imports
//   113 symbols from the "env" module that wasmtime's DefineWasi() does not
//   provide. This file defines every env:: import required for the module to
//   instantiate without "unknown import" errors.
//
// Inputs:
//   *wasmtime.Linker, *wasmtime.Store — injected from wasmtime_runtime.go boot()
//
// Outputs:
//   error — first definition failure aborts boot; all 113 symbols must succeed.
//
// Constraints:
//   - Only define exactly what pglite v0.2.17 imports (enumerated via
//     WebAssembly.Module.imports in Node.js — 113 env:: symbols confirmed).
//   - Globals __memory_base, __table_base, __stack_pointer are i32 immutable/mutable.
//   - env::memory is the shared linear memory (64 MB initial, growable).
//   - env::__indirect_function_table is a funcref table (0 initial, growable).
//   - invoke_* trampolines call table element at index [tableIdx] with tail args.
//   - __syscall_* stubs return -38 (ENOSYS) — pglite uses its own VFS, so real
//     syscall routing is not needed for the smoke-test subset.
//   - emscripten_resize_heap: attempts memory.Grow; returns 0 on failure.
//   - Network stubs (getaddrinfo, getnameinfo, __syscall_socket, etc.) return
//     ENOSYS because pglite/embedded-pg has no network path in P1 scope.
//
// SPORT: MASTER-FEATURES.md § embedded-postgres-wasm
//
// See also: wasmtime_runtime.go (boot function that calls defineEmscriptenABI).

import (
	"errors"
	"fmt"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// enosys is the standard "function not implemented" errno for stub syscalls.
const enosys = int32(-38)

// errIndirectCallNotFound is returned by invoke_* trampolines when the
// __indirect_function_table slot at the requested index is empty (null funcref)
// or out of bounds.
var errIndirectCallNotFound = errors.New("embedded/abi: indirect call: function not found in table")

// tableGet retrieves the *Func at index idx from the __indirect_function_table.
//
// Returns (nil, errIndirectCallNotFound) when the slot is out of bounds or
// holds a null funcref, so callers can return a typed zero value safely.
func tableGet(tbl *wasmtime.Table, store wasmtime.Storelike, idx int32) (*wasmtime.Func, error) {
	if idx < 0 {
		return nil, errIndirectCallNotFound
	}
	val, err := tbl.Get(store, uint32(idx))
	if err != nil {
		// Out-of-bounds index.
		return nil, errIndirectCallNotFound
	}
	fn := val.Funcref()
	if fn == nil {
		return nil, errIndirectCallNotFound
	}
	return fn, nil
}

// defineGOTNamespaces registers all imports from the GOT.mem and GOT.func
// namespace modules required by pglite v0.2.17.
//
// Emscripten's dynamic-linking ABI uses Global Offset Table (GOT) globals for
// position-independent addressing. The WASM binary imports mutable i32 globals
// from the "GOT.mem" namespace; the runtime dynamic linker writes the resolved
// addresses before calling _start. "GOT.func" imports are not present in
// pglite v0.2.17 (verified via WebAssembly.Module.imports in Node.js).
//
// Enumerated GOT.mem imports (1 total, confirmed from pglite v0.2.17 binary):
//
//	GOT.mem::__heap_base — mutable i32; address of the start of the free heap.
//
// Mutability: GOT.mem globals MUST be mutable so the dynamic linker can write
// the final relocated address. Initial value 0 is correct — the linker writes
// the real address during module initialisation.
//
// Must be called after DefineWasi() and defineEmscriptenABI(), before Instantiate().
func defineGOTNamespaces(linker *wasmtime.Linker, store *wasmtime.Store) error {
	// All GOT.mem globals are mutable i32.
	i32Mut := wasmtime.NewGlobalType(wasmtime.NewValType(wasmtime.KindI32), true)

	gotMemGlobals := []string{
		"__heap_base",
	}
	for _, name := range gotMemGlobals {
		g, err := wasmtime.NewGlobal(store, i32Mut, wasmtime.ValI32(0))
		if err != nil {
			return fmt.Errorf("embedded/abi: GOT.mem::%s create: %w", name, err)
		}
		if err := linker.Define(store, "GOT.mem", name, g); err != nil {
			return fmt.Errorf("embedded/abi: GOT.mem::%s define: %w", name, err)
		}
	}
	return nil
}

// defineEmscriptenABI registers all 113 env:: imports required by pglite v0.2.17
// with Emscripten. It must be called after DefineWasi() and before Instantiate().
//
// The function is split into four logical groups:
//  1. Globals (__memory_base, __table_base, __stack_pointer)
//  2. Memory (env::memory — shared linear memory)
//  3. Table (__indirect_function_table — funcref table for invoke_* trampolines)
//  4. Functions — all remaining 108 env:: function imports
func defineEmscriptenABI(linker *wasmtime.Linker, store *wasmtime.Store) error {
	// ── 1. Globals ──────────────────────────────────────────────────────────
	//
	// Emscripten SIDE_MODULE linking globals. Values are set to 0; pglite's
	// dynamic linker sets them at runtime before any code runs.

	i32Const := wasmtime.NewGlobalType(wasmtime.NewValType(wasmtime.KindI32), false)
	i32Mut := wasmtime.NewGlobalType(wasmtime.NewValType(wasmtime.KindI32), true)

	memBase, err := wasmtime.NewGlobal(store, i32Const, wasmtime.ValI32(0))
	if err != nil {
		return fmt.Errorf("embedded/abi: __memory_base: %w", err)
	}
	if err := linker.Define(store, "env", "__memory_base", memBase); err != nil {
		return fmt.Errorf("embedded/abi: define __memory_base: %w", err)
	}

	tableBase, err := wasmtime.NewGlobal(store, i32Const, wasmtime.ValI32(0))
	if err != nil {
		return fmt.Errorf("embedded/abi: __table_base: %w", err)
	}
	if err := linker.Define(store, "env", "__table_base", tableBase); err != nil {
		return fmt.Errorf("embedded/abi: define __table_base: %w", err)
	}

	// __stack_pointer is mutable (pglite's stack allocator modifies it).
	stackPtr, err := wasmtime.NewGlobal(store, i32Mut, wasmtime.ValI32(0))
	if err != nil {
		return fmt.Errorf("embedded/abi: __stack_pointer: %w", err)
	}
	if err := linker.Define(store, "env", "__stack_pointer", stackPtr); err != nil {
		return fmt.Errorf("embedded/abi: define __stack_pointer: %w", err)
	}

	// ── 2. Linear Memory ────────────────────────────────────────────────────
	//
	// Emscripten programs import a shared memory from "env". We provide 64 MB
	// initial (1024 pages × 64 KB) with growth enabled (max = 0 means unlimited
	// in wasmtime).
	const initialPages = 1024 // 64 MB (1024 × 65536 bytes)
	mem, err := wasmtime.NewMemory(store, wasmtime.NewMemoryType(initialPages, false, 0 /* no max */))
	if err != nil {
		return fmt.Errorf("embedded/abi: env::memory: %w", err)
	}
	if err := linker.Define(store, "env", "memory", mem); err != nil {
		return fmt.Errorf("embedded/abi: define memory: %w", err)
	}

	// ── 3. Indirect Function Table ───────────────────────────────────────────
	//
	// invoke_* trampolines index into this funcref table. We start with 0
	// elements; pglite's dynamic linker populates it at runtime.
	tbl, err := wasmtime.NewTable(
		store,
		wasmtime.NewTableType(wasmtime.NewValType(wasmtime.KindFuncref), 0, false /* no max */, 0),
		wasmtime.ValFuncref(nil),
	)
	if err != nil {
		return fmt.Errorf("embedded/abi: __indirect_function_table: %w", err)
	}
	if err := linker.Define(store, "env", "__indirect_function_table", tbl); err != nil {
		return fmt.Errorf("embedded/abi: define __indirect_function_table: %w", err)
	}

	// ── 4. Functions ─────────────────────────────────────────────────────────

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
	if err := linker.DefineFunc(store, "env", "_mmap_js",
		func(length, prot, flags, fd, offset int32, allocated, addr int32) int32 {
			return enosys
		}); err != nil {
		return fmt.Errorf("embedded/abi: _mmap_js: %w", err)
	}

	// _munmap_js — munmap. Always succeeds (no-op).
	if err := linker.DefineFunc(store, "env", "_munmap_js",
		func(addr, length int32) int32 {
			return 0
		}); err != nil {
		return fmt.Errorf("embedded/abi: _munmap_js: %w", err)
	}

	// ── 4c. Dynamic linking ──────────────────────────────────────────────

	// _dlopen_js — dynamic library open. Not supported; return 0 (NULL).
	if err := linker.DefineFunc(store, "env", "_dlopen_js",
		func(filename, flags int32) int32 {
			return 0
		}); err != nil {
		return fmt.Errorf("embedded/abi: _dlopen_js: %w", err)
	}

	// _dlsym_js — symbol lookup. Not supported; return 0 (NULL).
	if err := linker.DefineFunc(store, "env", "_dlsym_js",
		func(handle, symbol int32) int32 {
			return 0
		}); err != nil {
		return fmt.Errorf("embedded/abi: _dlsym_js: %w", err)
	}

	// ── 4d. Time ─────────────────────────────────────────────────────────

	// emscripten_date_now — returns milliseconds since epoch (f64).
	if err := linker.DefineFunc(store, "env", "emscripten_date_now", func() float64 {
		return float64(time.Now().UnixMilli())
	}); err != nil {
		return fmt.Errorf("embedded/abi: emscripten_date_now: %w", err)
	}

	// emscripten_get_now — high-resolution monotonic time in milliseconds.
	var startTime = time.Now()
	if err := linker.DefineFunc(store, "env", "emscripten_get_now", func() float64 {
		return float64(time.Since(startTime).Microseconds()) / 1000.0
	}); err != nil {
		return fmt.Errorf("embedded/abi: emscripten_get_now: %w", err)
	}

	// _gmtime_js — convert Unix timestamp to UTC tm struct via pointer.
	// pglite passes a timestamp (f64) and a pointer to a pre-allocated tm struct.
	// We stub to no-op; the embedded SQL path does not exercise date formatting.
	if err := linker.DefineFunc(store, "env", "_gmtime_js",
		func(time_ float64, tmPtr int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: _gmtime_js: %w", err)
	}

	// _localtime_js — convert Unix timestamp to local tm struct.
	if err := linker.DefineFunc(store, "env", "_localtime_js",
		func(time_ float64, tmPtr int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: _localtime_js: %w", err)
	}

	// _tzset_js — set timezone info. No-op.
	if err := linker.DefineFunc(store, "env", "_tzset_js",
		func(timezone, daylight, stdName, dstName int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: _tzset_js: %w", err)
	}

	// _setitimer_js — set interval timer. No-op shim.
	if err := linker.DefineFunc(store, "env", "_setitimer_js",
		func(which, timeout_ms float64) {}); err != nil {
		return fmt.Errorf("embedded/abi: _setitimer_js: %w", err)
	}

	// ── 4e. Emscripten event loop ─────────────────────────────────────────

	// emscripten_set_main_loop — browser-only event loop. No-op in WASI.
	if err := linker.DefineFunc(store, "env", "emscripten_set_main_loop",
		func(fn, fps, simulateInfiniteLoop int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: emscripten_set_main_loop: %w", err)
	}

	// emscripten_asm_const_int — execute inline asm JS. Not supported; return 0.
	if err := linker.DefineFunc(store, "env", "emscripten_asm_const_int",
		func(code, sigPtr, argbuf int32) int32 {
			return 0
		}); err != nil {
		return fmt.Errorf("embedded/abi: emscripten_asm_const_int: %w", err)
	}

	// _emscripten_system — execute shell command. Blocked; return -1 (EPERM).
	if err := linker.DefineFunc(store, "env", "_emscripten_system",
		func(cmd int32) int32 {
			return -1
		}); err != nil {
		return fmt.Errorf("embedded/abi: _emscripten_system: %w", err)
	}

	// ── 4f. Network stubs (ENOSYS) ────────────────────────────────────────
	//
	// pglite's embedded-PG path uses Unix-domain sockets via WASI preopens,
	// not the Emscripten network layer. These stubs satisfy the import
	// requirement while returning ENOSYS on any actual call.

	if err := linker.DefineFunc(store, "env", "getaddrinfo",
		func(node, service, hints, res int32) int32 { return enosys }); err != nil {
		return fmt.Errorf("embedded/abi: getaddrinfo: %w", err)
	}
	if err := linker.DefineFunc(store, "env", "getnameinfo",
		func(sa, salen, host, hostlen, serv, servlen, flags int32) int32 { return enosys }); err != nil {
		return fmt.Errorf("embedded/abi: getnameinfo: %w", err)
	}

	// ── 4g. __syscall_* stubs ─────────────────────────────────────────────
	//
	// Emscripten maps Linux syscalls to these __syscall_* wrappers.
	// pglite's VFS intercepts file I/O above the syscall layer; these stubs
	// return ENOSYS on direct calls and satisfy the import table only.

	syscallEnosys1 := func(a0 int32) int32 { return enosys }
	syscallEnosys2 := func(a0, a1 int32) int32 { return enosys }
	syscallEnosys3 := func(a0, a1, a2 int32) int32 { return enosys }
	syscallEnosys4 := func(a0, a1, a2, a3 int32) int32 { return enosys }
	syscallEnosys5 := func(a0, a1, a2, a3, a4 int32) int32 { return enosys }
	syscallEnosys6 := func(a0, a1, a2, a3, a4, a5 int32) int32 { return enosys }

	for _, def := range []struct {
		name string
		fn   interface{}
	}{
		{"__syscall__newselect", syscallEnosys5},
		{"__syscall_bind", syscallEnosys3},
		{"__syscall_chdir", syscallEnosys1},
		{"__syscall_chmod", syscallEnosys2},
		{"__syscall_connect", syscallEnosys3},
		{"__syscall_dup", syscallEnosys1},
		{"__syscall_dup3", syscallEnosys3},
		{"__syscall_faccessat", syscallEnosys4},
		{"__syscall_fadvise64", syscallEnosys4},
		{"__syscall_fallocate", syscallEnosys4},
		{"__syscall_fcntl64", syscallEnosys3},
		{"__syscall_fdatasync", syscallEnosys1},
		{"__syscall_fstat64", syscallEnosys2},
		{"__syscall_ftruncate64", syscallEnosys3},
		{"__syscall_getcwd", syscallEnosys2},
		{"__syscall_getdents64", syscallEnosys3},
		{"__syscall_getsockname", syscallEnosys3},
		{"__syscall_getsockopt", syscallEnosys5},
		{"__syscall_ioctl", syscallEnosys3},
		{"__syscall_lstat64", syscallEnosys2},
		{"__syscall_mkdirat", syscallEnosys3},
		{"__syscall_newfstatat", syscallEnosys4},
		{"__syscall_openat", syscallEnosys4},
		{"__syscall_pipe", syscallEnosys1},
		{"__syscall_poll", syscallEnosys3},
		{"__syscall_readlinkat", syscallEnosys4},
		{"__syscall_recvfrom", syscallEnosys6},
		{"__syscall_renameat", syscallEnosys4},
		{"__syscall_rmdir", syscallEnosys1},
		{"__syscall_sendto", syscallEnosys6},
		{"__syscall_socket", syscallEnosys3},
		{"__syscall_stat64", syscallEnosys2},
		{"__syscall_symlinkat", syscallEnosys3},
		{"__syscall_truncate64", syscallEnosys3},
		{"__syscall_unlinkat", syscallEnosys3},
	} {
		// Capture loop variable for the closure.
		d := def
		if err := linker.DefineFunc(store, "env", d.name, d.fn); err != nil {
			return fmt.Errorf("embedded/abi: %s: %w", d.name, err)
		}
	}

	// ── 4h. invoke_* trampolines ─────────────────────────────────────────
	//
	// Emscripten uses invoke_<sig> helpers for indirect calls through the
	// function table when exceptions/longjmp are in play.
	//
	// Signature encoding (Emscripten convention):
	//   v = void, i = i32, j = i64, d = f64, f = f32
	//
	// pglite v0.2.17 imports 48 distinct invoke_* signatures. Each trampoline:
	//  1. Reads the funcref at __indirect_function_table[tableIdx] via tableGet.
	//  2. Calls it with the trailing typed arguments via wasmtime.Func.Call.
	//  3. Returns the typed result (or zero on null/OOB slot; never panics).
	//
	// The *wasmtime.Caller first parameter is provided by wasmtime and acts as
	// the Storelike context; it is NOT counted as a wasm parameter. The last
	// *wasmtime.Trap return is the conventional "host trap" return; nil = ok.

	type invokeError = *wasmtime.Trap

	// ── void return ──────────────────────────────────────────────────────────

	if err := linker.DefineFunc(store, "env", "invoke_v",
		func(c *wasmtime.Caller, idx int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_v[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_v: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_vi",
		func(c *wasmtime.Caller, idx, a0 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_vi[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_vi: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_vii",
		func(c *wasmtime.Caller, idx, a0, a1 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_vii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_vii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viii",
		func(c *wasmtime.Caller, idx, a0, a1, a2 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4, a5); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiiiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4, a5, a6); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiiiiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6, a7 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4, a5, a6, a7); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiiiiiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6, a7, a8 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4, a5, a6, a7, a8); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiiiiiiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiiiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiiiiiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiiiiiiiiiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiiiiiiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_vid",
		func(c *wasmtime.Caller, idx int32, a0 float64) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_vid[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_vid: %w", err)
	}

	// ── void + i64 variants ───────────────────────────────────────────────────

	if err := linker.DefineFunc(store, "env", "invoke_viiij",
		func(c *wasmtime.Caller, idx, a0, a1, a2 int32, a3 int64) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiij[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiij: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viij",
		func(c *wasmtime.Caller, idx, a0, a1 int32, a2 int64) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viij[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viij: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viiji",
		func(c *wasmtime.Caller, idx, a0, a1 int32, a2 int64, a3 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viiji[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viiji: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viijii",
		func(c *wasmtime.Caller, idx, a0, a1 int32, a2 int64, a3, a4 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viijii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viijii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viijiiii",
		func(c *wasmtime.Caller, idx, a0, a1 int32, a2 int64, a3, a4, a5, a6 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4, a5, a6); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viijiiii[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viijiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_vij",
		func(c *wasmtime.Caller, idx, a0 int32, a1 int64) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_vij[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_vij: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_viji",
		func(c *wasmtime.Caller, idx, a0 int32, a1 int64, a2 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_viji[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_viji: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_vijiji",
		func(c *wasmtime.Caller, idx, a0 int32, a1 int64, a2 int32, a3 int64, a4 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1, a2, a3, a4); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_vijiji[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_vijiji: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_vj",
		func(c *wasmtime.Caller, idx int32, a0 int64) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_vj[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_vj: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_vji",
		func(c *wasmtime.Caller, idx int32, a0 int64, a1 int32) invokeError {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return nil
			}
			if _, e := fn.Call(c, a0, a1); e != nil {
				return wasmtime.NewTrap(fmt.Sprintf("invoke_vji[%d]: %v", idx, e))
			}
			return nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_vji: %w", err)
	}

	// ── i32 return ────────────────────────────────────────────────────────────

	if err := linker.DefineFunc(store, "env", "invoke_i",
		func(c *wasmtime.Caller, idx int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_i[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_i: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_ii",
		func(c *wasmtime.Caller, idx, a0 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_ii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_ii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iii",
		func(c *wasmtime.Caller, idx, a0, a1 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5, a6)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6, a7 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5, a6, a7)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiiiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6, a7, a8 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5, a6, a7, a8)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiiiiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiiiiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiiiiiiiiiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiiiiiiiiiiiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiiiiiiiiiiiiiii: %w", err)
	}

	// ── i32 return + i64 variants ─────────────────────────────────────────────

	if err := linker.DefineFunc(store, "env", "invoke_iiij",
		func(c *wasmtime.Caller, idx, a0, a1 int32, a2 int64) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiij[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiij: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiiiji",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3 int32, a4 int64, a5 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiiiji[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiiiji: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiij",
		func(c *wasmtime.Caller, idx, a0, a1, a2 int32, a3 int64) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiij[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiij: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiiijii",
		func(c *wasmtime.Caller, idx, a0, a1, a2 int32, a3 int64, a4, a5 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiiijii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiiijii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_iiji",
		func(c *wasmtime.Caller, idx, a0 int32, a1 int64, a2 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_iiji[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_iiji: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_ij",
		func(c *wasmtime.Caller, idx int32, a0 int64) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_ij[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_ij: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_ijiiiii",
		func(c *wasmtime.Caller, idx int32, a0 int64, a1, a2, a3, a4, a5 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_ijiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_ijiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_ijiiiiii",
		func(c *wasmtime.Caller, idx int32, a0 int64, a1, a2, a3, a4, a5, a6 int32) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5, a6)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_ijiiiiii[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_ijiiiiii: %w", err)
	}

	// ── i64 return ────────────────────────────────────────────────────────────

	if err := linker.DefineFunc(store, "env", "invoke_ji",
		func(c *wasmtime.Caller, idx, a0 int32) (int64, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_ji[%d]: %v", idx, e2))
			}
			return ret.(int64), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_ji: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_jii",
		func(c *wasmtime.Caller, idx, a0, a1 int32) (int64, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_jii[%d]: %v", idx, e2))
			}
			return ret.(int64), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_jii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_jiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3 int32) (int64, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_jiiii[%d]: %v", idx, e2))
			}
			return ret.(int64), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_jiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_jiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4 int32) (int64, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_jiiiii[%d]: %v", idx, e2))
			}
			return ret.(int64), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_jiiiii: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_jiiiiiiii",
		func(c *wasmtime.Caller, idx, a0, a1, a2, a3, a4, a5, a6, a7 int32) (int64, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0, a1, a2, a3, a4, a5, a6, a7)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_jiiiiiiii[%d]: %v", idx, e2))
			}
			return ret.(int64), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_jiiiiiiii: %w", err)
	}

	// ── f64 return ────────────────────────────────────────────────────────────

	if err := linker.DefineFunc(store, "env", "invoke_di",
		func(c *wasmtime.Caller, idx, a0 int32) (float64, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_di[%d]: %v", idx, e2))
			}
			return ret.(float64), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_di: %w", err)
	}

	if err := linker.DefineFunc(store, "env", "invoke_id",
		func(c *wasmtime.Caller, idx int32, a0 float64) (int32, invokeError) {
			fn, e := tableGet(tbl, c, idx)
			if e != nil {
				return 0, nil
			}
			ret, e2 := fn.Call(c, a0)
			if e2 != nil {
				return 0, wasmtime.NewTrap(fmt.Sprintf("invoke_id[%d]: %v", idx, e2))
			}
			return ret.(int32), nil
		}); err != nil {
		return fmt.Errorf("embedded/abi: invoke_id: %w", err)
	}

	return nil
}
