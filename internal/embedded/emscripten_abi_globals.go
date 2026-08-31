//go:build cgo

package embedded

// emscripten_abi_globals.go — Emscripten globals, linear memory, and the
// indirect function table imports for pglite v0.2.17 (groups 1-3 of
// defineEmscriptenABI). Split from emscripten_abi.go (T-P6-E2-W1-S1-T3).
// Inputs:  *wasmtime.Linker, *wasmtime.Store.
// Outputs: the created env::memory (*wasmtime.Memory), needed by
//          defineEmscriptenRuntimeFuncs's emscripten_resize_heap; error.
// Constraints: pure move, same values/order, no behavior change.

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// defineEmscriptenGlobalsMemoryTable registers __memory_base, __table_base,
// __stack_pointer, env::memory, and __indirect_function_table. Returns the
// created memory object (needed by defineEmscriptenRuntimeFuncs).
func defineEmscriptenGlobalsMemoryTable(linker *wasmtime.Linker, store *wasmtime.Store) (mem *wasmtime.Memory, err error) {
	// ── 1. Globals ──────────────────────────────────────────────────────────
	//
	// Emscripten SIDE_MODULE linking globals. Values are set to 0; pglite's
	// dynamic linker sets them at runtime before any code runs.

	// Declared by pglite v0.2.17's env::memory import; also sets the stack top.
	const (
		pageSize     = 65536
		initialPages = 2048 // 128 MB (2048 x 65536 bytes) — declared minimum
	)

	i32Const := wasmtime.NewGlobalType(wasmtime.NewValType(wasmtime.KindI32), false)
	i32Mut := wasmtime.NewGlobalType(wasmtime.NewValType(wasmtime.KindI32), true)

	memBase, err := wasmtime.NewGlobal(store, i32Const, wasmtime.ValI32(0))
	if err != nil {
		return nil, fmt.Errorf("embedded/abi: __memory_base: %w", err)
	}
	if err := linker.Define(store, "env", "__memory_base", memBase); err != nil {
		return nil, fmt.Errorf("embedded/abi: define __memory_base: %w", err)
	}

	tableBase, err := wasmtime.NewGlobal(store, i32Const, wasmtime.ValI32(0))
	if err != nil {
		return nil, fmt.Errorf("embedded/abi: __table_base: %w", err)
	}
	if err := linker.Define(store, "env", "__table_base", tableBase); err != nil {
		return nil, fmt.Errorf("embedded/abi: define __table_base: %w", err)
	}

	// __stack_pointer is mutable (pglite's stack allocator modifies it) and MUST
	// start at the top of a usable stack region, not at 0.
	//
	// The WASM stack grows DOWNWARD: every function prologue does
	// `sp -= frame_size`. Starting it at 0 means the very first prologue
	// underflows i32 and dereferences near the top of the 4 GB address space.
	// That produced the trap that made every embedded PG integration test fail:
	//
	//   memory fault at wasm address 0xfffffef0 in linear memory of size 0x8000000
	//   wasm trap: out of bounds memory access
	//
	// 0xfffffef0 is exactly 0 - 272, a 272-byte first frame subtracted from a zero
	// stack pointer, not a genuine 4 GB access. With a valid stack top the same
	// boot proceeds 8 frames deep instead of trapping in the first.
	//
	// The stack starts at the end of the declared initial memory and grows down
	// toward the data/heap region, which starts at __memory_base (0). This is the
	// standard Emscripten layout and keeps maximum separation between the two.
	const stackTop = initialPages * pageSize // 128 MB, one past the last valid byte
	stackPtr, err := wasmtime.NewGlobal(store, i32Mut, wasmtime.ValI32(stackTop))
	if err != nil {
		return nil, fmt.Errorf("embedded/abi: __stack_pointer: %w", err)
	}
	if err := linker.Define(store, "env", "__stack_pointer", stackPtr); err != nil {
		return nil, fmt.Errorf("embedded/abi: define __stack_pointer: %w", err)
	}

	// ── 2. Linear Memory ────────────────────────────────────────────────────
	//
	// pglite v0.2.17 imports env::memory with min=2048 pages (128 MB) and
	// max=32768 pages (2 GB), as declared in the WASM binary's import section
	// (confirmed via Node.js WebAssembly.Module.imports + binary parse).
	// Providing fewer initial pages or a larger maximum causes an
	// "incompatible import type" trap at instantiation.
	const (
		maxPages = 32768 // 2 GB (32768 × 65536 bytes) — declared maximum
	)
	mem, err = wasmtime.NewMemory(store, wasmtime.NewMemoryType(initialPages, true, maxPages))
	if err != nil {
		return nil, fmt.Errorf("embedded/abi: env::memory: %w", err)
	}
	if err := linker.Define(store, "env", "memory", mem); err != nil {
		return nil, fmt.Errorf("embedded/abi: define memory: %w", err)
	}

	// ── 3. Indirect Function Table ───────────────────────────────────────────
	//
	// pglite v0.2.17 imports env::__indirect_function_table with min=5359 elements
	// and no declared maximum (confirmed via binary parse of the import section).
	// The invoke_* trampolines index into this table; pglite's dynamic linker
	// populates the entries before calling _start. The initial size must be at
	// least 5359 or instantiation fails with "table import is smaller than initial".
	const tableInitialElems = 5359 // declared minimum in pglite v0.2.17 import section
	tbl, err := wasmtime.NewTable(
		store,
		wasmtime.NewTableType(wasmtime.NewValType(wasmtime.KindFuncref), tableInitialElems, false /* no max */, 0),
		wasmtime.ValFuncref(nil),
	)
	if err != nil {
		return nil, fmt.Errorf("embedded/abi: __indirect_function_table: %w", err)
	}
	if err := linker.Define(store, "env", "__indirect_function_table", tbl); err != nil {
		return nil, fmt.Errorf("embedded/abi: define __indirect_function_table: %w", err)
	}

	return mem, nil
}
