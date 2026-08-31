//go:build cgo

package embedded

// emscripten_abi.go — Emscripten host-import shims for pglite v0.2.17.
//
// Purpose:
//   pglite v0.2.17 is compiled with Emscripten (emcc) targeting WASI. It imports
//   113 symbols from the "env" module that wasmtime's DefineWasi() does not
//   provide. This file (plus its emscripten_abi_{globals,runtime,time,syscalls,
//   invoke}.go siblings, split out for 300-line compliance per T-P6-E2-W1-S1-T3)
//   defines every env:: import required for the module to instantiate without
//   "unknown import" errors.
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
	"fmt"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// enosys is the standard "function not implemented" errno for stub syscalls.
const enosys = int32(-38)

// defineEmscriptenABI registers all 113 env:: imports required by pglite v0.2.17
// with Emscripten. It must be called after DefineWasi() and before Instantiate().
//
// Delegates to five per-domain helpers, called in the same order as the
// original monolithic function body (T-P6-E2-W1-S1-T3):
//
//	1-3. defineEmscriptenGlobalsMemoryTable (Globals, Memory, Table)
//	4a-4c. defineEmscriptenRuntimeFuncs (process/runtime, memory mgmt, dynamic linking)
//	4d-4f. defineEmscriptenTimeEventNetFuncs (time, event loop, network stubs)
//	4g. defineEmscriptenSyscallStubs (__syscall_* stubs)
//	4h. defineEmscriptenInvokeTrampolines (invoke_* trampolines)
func defineEmscriptenABI(linker *wasmtime.Linker, store *wasmtime.Store) error {
	mem, err := defineEmscriptenGlobalsMemoryTable(linker, store)
	if err != nil {
		return err
	}
	if err := defineEmscriptenRuntimeFuncs(linker, store, mem); err != nil {
		return err
	}
	if err := defineEmscriptenTimeEventNetFuncs(linker, store); err != nil {
		return err
	}
	if err := defineEmscriptenSyscallStubs(linker, store); err != nil {
		return err
	}
	if err := defineEmscriptenInvokeTrampolines(linker, store); err != nil {
		return err
	}
	return nil
}

// defineGOTNamespaces registers all imports from the GOT.mem and GOT.func
// namespace modules required by pglite v0.2.17.
//
// Emscripten's dynamic-linking ABI uses Global Offset Table (GOT) globals for
// position-independent addressing. The WASM binary imports mutable i32 globals
// from the "GOT.mem" namespace; the dynamic linker writes the resolved addresses
// before calling _start. No GOT.func imports are present in pglite v0.2.17
// (confirmed via WebAssembly.Module.imports in Node.js).
//
// Enumerated GOT.mem imports (1 total, confirmed from pglite v0.2.17 binary):
//   - __heap_base — the address one past the end of the static data segment;
//     pglite's dynamic linker/allocator uses it as the start of the heap.
//
// __heap_base MUST be a real, non-zero address derived from the module's own
// layout, not a placeholder. It marks the boundary the allocator treats as
// "already in use" — everything below it is static data the compiled WASM
// module keeps forever. Leaving __heap_base at 0 places the heap on top of the
// static data segment at address 0, and allocator arithmetic derived from it
// walks straight out of linear memory.
//
// The correct value comes from the module's own dylink.0 section
// (WASM_DYLINK_MEM_INFO.memorysize, rounded up to memoryalignment) — see
// dylink.go. For pglite v0.2.17 that is 2172900 bytes aligned to 4096 = 0x213000.
// Deriving it per-module means a pglite upgrade cannot silently invalidate it.
//
// Must be called after DefineWasi() and defineEmscriptenABI(), before Instantiate().
func defineGOTNamespaces(linker *wasmtime.Linker, store *wasmtime.Store, heapBase uint32) error {
	// All GOT.mem globals are mutable i32.
	i32Mut := wasmtime.NewGlobalType(wasmtime.NewValType(wasmtime.KindI32), true)

	gotMemGlobals := map[string]uint32{
		"__heap_base": heapBase,
	}
	for name, val := range gotMemGlobals {
		g, err := wasmtime.NewGlobal(store, i32Mut, wasmtime.ValI32(int32(val))) //nolint:gosec // heapBase < 2^31 for any real module
		if err != nil {
			return fmt.Errorf("embedded/abi: GOT.mem::%s create: %w", name, err)
		}
		if err := linker.Define(store, "GOT.mem", name, g); err != nil {
			return fmt.Errorf("embedded/abi: GOT.mem::%s define: %w", name, err)
		}
	}
	return nil
}
