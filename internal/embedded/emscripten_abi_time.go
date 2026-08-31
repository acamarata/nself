//go:build cgo

package embedded

// emscripten_abi_time.go — Emscripten time, event-loop, and network-stub
// imports for pglite v0.2.17 (groups 4d-4f of defineEmscriptenABI). Split
// from emscripten_abi.go (T-P6-E2-W1-S1-T3).
// Inputs:  *wasmtime.Linker, *wasmtime.Store.
// Outputs: error — first definition failure aborts.
// Constraints: pure move, same values/order, no behavior change.

import (
	"fmt"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// defineEmscriptenTimeEventNetFuncs registers the time, event-loop, and
// network-stub (ENOSYS) env:: function imports.
func defineEmscriptenTimeEventNetFuncs(linker *wasmtime.Linker, store *wasmtime.Store) error {
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
	// Actual WASM signature: (i64, i32) -> () — timestamp is i64 (not f64).
	if err := linker.DefineFunc(store, "env", "_gmtime_js",
		func(time_ int64, tmPtr int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: _gmtime_js: %w", err)
	}

	// _localtime_js — convert Unix timestamp to local tm struct.
	// Actual WASM signature: (i64, i32) -> () — timestamp is i64 (not f64).
	if err := linker.DefineFunc(store, "env", "_localtime_js",
		func(time_ int64, tmPtr int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: _localtime_js: %w", err)
	}

	// _tzset_js — set timezone info. No-op.
	if err := linker.DefineFunc(store, "env", "_tzset_js",
		func(timezone, daylight, stdName, dstName int32) {}); err != nil {
		return fmt.Errorf("embedded/abi: _tzset_js: %w", err)
	}

	// _setitimer_js — set interval timer. Returns 0 (success).
	// Actual WASM signature: (i32, f64) -> (i32).
	if err := linker.DefineFunc(store, "env", "_setitimer_js",
		func(which int32, timeout_ms float64) int32 { return 0 }); err != nil {
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

	return nil
}
