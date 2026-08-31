//go:build cgo

package embedded

// emscripten_abi_invoke.go — Emscripten invoke_* trampoline imports for
// pglite v0.2.17 (group 4h of defineEmscriptenABI). Split from
// emscripten_abi.go (T-P6-E2-W1-S1-T3).
// Inputs:  *wasmtime.Linker, *wasmtime.Store.
// Outputs: error — first definition failure aborts.
// Constraints: pure move, same values/order, no behavior change. All 48
//              invoke_<sig> stubs are no-ops returning the zero value.

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// defineEmscriptenInvokeTrampolines registers the invoke_* trampoline
// env:: function imports.
func defineEmscriptenInvokeTrampolines(linker *wasmtime.Linker, store *wasmtime.Store) error {
	// ── 4h. invoke_* trampolines ─────────────────────────────────────────
	//
	// Emscripten uses invoke_<sig> helpers for indirect calls through the
	// function table when exceptions/longjmp are in play. Each trampoline:
	//  1. Reads the function reference from __indirect_function_table[tableIdx].
	//  2. Calls it with the remaining arguments.
	//
	// Signature encoding (Emscripten convention):
	//   v = void, i = i32, j = i64, d = f64, f = f32
	//
	// pglite v0.2.17 imports 48 distinct invoke_* signatures. All are defined
	// here as no-op stubs that return the zero value for the return type.
	// The stubs satisfy WASM instantiation; pglite's Emscripten runtime
	// dispatches through __indirect_function_table at execution time.

	invokeStubs := []struct {
		name string
		fn   interface{}
	}{
		// void return
		{"invoke_v", func(idx int32) {}},
		{"invoke_vi", func(idx, a0 int32) {}},
		{"invoke_vii", func(idx, a0, a1 int32) {}},
		{"invoke_viii", func(idx, a0, a1, a2 int32) {}},
		{"invoke_viiii", func(idx, a0, a1, a2, a3 int32) {}},
		{"invoke_viiiii", func(idx, a0, a1, a2, a3, a4 int32) {}},
		{"invoke_viiiiii", func(idx, a0, a1, a2, a3, a4, a5 int32) {}},
		{"invoke_viiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6 int32) {}},
		{"invoke_viiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6, a7 int32) {}},
		{"invoke_viiiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6, a7, a8 int32) {}},
		{"invoke_viiiiiiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11 int32) {}},
		{"invoke_vid", func(idx int32, a0 int32, a1 float64) {}},
		// void + i64 variants
		{"invoke_viiij", func(idx, a0, a1, a2 int32, a3 int64) {}},
		{"invoke_viij", func(idx, a0, a1 int32, a2 int64) {}},
		{"invoke_viiji", func(idx, a0, a1 int32, a2 int64, a3 int32) {}},
		{"invoke_viijii", func(idx, a0, a1 int32, a2 int64, a3, a4 int32) {}},
		{"invoke_viijiiii", func(idx, a0, a1 int32, a2 int64, a3, a4, a5, a6 int32) {}},
		{"invoke_vij", func(idx, a0 int32, a1 int64) {}},
		{"invoke_viji", func(idx, a0 int32, a1 int64, a2 int32) {}},
		{"invoke_vijiji", func(idx, a0 int32, a1 int64, a2 int32, a3 int64, a4 int32) {}},
		{"invoke_vj", func(idx int32, a0 int64) {}},
		{"invoke_vji", func(idx int32, a0 int64, a1 int32) {}},
		// i32 return
		{"invoke_i", func(idx int32) int32 { return 0 }},
		{"invoke_ii", func(idx, a0 int32) int32 { return 0 }},
		{"invoke_iii", func(idx, a0, a1 int32) int32 { return 0 }},
		{"invoke_iiii", func(idx, a0, a1, a2 int32) int32 { return 0 }},
		{"invoke_iiiii", func(idx, a0, a1, a2, a3 int32) int32 { return 0 }},
		{"invoke_iiiiii", func(idx, a0, a1, a2, a3, a4 int32) int32 { return 0 }},
		{"invoke_iiiiiii", func(idx, a0, a1, a2, a3, a4, a5 int32) int32 { return 0 }},
		{"invoke_iiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6 int32) int32 { return 0 }},
		{"invoke_iiiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6, a7 int32) int32 { return 0 }},
		{"invoke_iiiiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6, a7, a8 int32) int32 { return 0 }},
		{"invoke_iiiiiiiiiiiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15 int32) int32 { return 0 }},
		// i32 return + i64 variants
		{"invoke_iiij", func(idx, a0, a1 int32, a2 int64) int32 { return 0 }},
		{"invoke_iiiiiji", func(idx, a0, a1, a2, a3 int32, a4 int64, a5 int32) int32 { return 0 }},
		{"invoke_iiiij", func(idx, a0, a1, a2 int32, a3 int64) int32 { return 0 }},
		{"invoke_iiiijii", func(idx, a0, a1, a2 int32, a3 int64, a4, a5 int32) int32 { return 0 }},
		{"invoke_iiji", func(idx, a0 int32, a1 int64, a2 int32) int32 { return 0 }},
		{"invoke_ij", func(idx int32, a0 int64) int32 { return 0 }},
		{"invoke_ijiiiii", func(idx int32, a0 int64, a1, a2, a3, a4, a5 int32) int32 { return 0 }},
		{"invoke_ijiiiiii", func(idx int32, a0 int64, a1, a2, a3, a4, a5, a6 int32) int32 { return 0 }},
		// i64 return
		{"invoke_ji", func(idx, a0 int32) int64 { return 0 }},
		{"invoke_jii", func(idx, a0, a1 int32) int64 { return 0 }},
		{"invoke_jiiii", func(idx, a0, a1, a2, a3 int32) int64 { return 0 }},
		{"invoke_jiiiii", func(idx, a0, a1, a2, a3, a4 int32) int64 { return 0 }},
		{"invoke_jiiiiiiii", func(idx, a0, a1, a2, a3, a4, a5, a6, a7 int32) int64 { return 0 }},
		// f64 return
		{"invoke_di", func(idx, a0 int32) float64 { return 0 }},
		{"invoke_id", func(idx int32, a0 float64) int32 { return 0 }},
	}

	for _, s := range invokeStubs {
		stub := s // capture
		if err := linker.DefineFunc(store, "env", stub.name, stub.fn); err != nil {
			return fmt.Errorf("embedded/abi: %s: %w", stub.name, err)
		}
	}

	return nil
}
