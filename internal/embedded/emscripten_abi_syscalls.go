//go:build cgo

package embedded

// emscripten_abi_syscalls.go — Emscripten __syscall_* stub imports for
// pglite v0.2.17 (group 4g of defineEmscriptenABI). Split from
// emscripten_abi.go (T-P6-E2-W1-S1-T3).
// Inputs:  *wasmtime.Linker, *wasmtime.Store.
// Outputs: error — first definition failure aborts.
// Constraints: pure move, same values/order, no behavior change. All stubs
//              return ENOSYS; pglite uses its own VFS.

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// defineEmscriptenSyscallStubs registers the __syscall_* ENOSYS stub
// env:: function imports.
func defineEmscriptenSyscallStubs(linker *wasmtime.Linker, store *wasmtime.Store) error {
	// ── 4g. __syscall_* stubs ─────────────────────────────────────────────
	//
	// Emscripten maps Linux syscalls to these __syscall_* wrappers.
	// pglite's VFS intercepts file I/O above the syscall layer; these stubs
	// return ENOSYS on direct calls and satisfy the import table only.

	// Uniform-i32 syscall stubs (exact signatures from pglite v0.2.17 binary).
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
		// Uniform i32-only signatures.
		{"__syscall__newselect", syscallEnosys5},
		{"__syscall_chdir", syscallEnosys1},
		{"__syscall_chmod", syscallEnosys2},
		{"__syscall_dup", syscallEnosys1},
		{"__syscall_dup3", syscallEnosys3},
		{"__syscall_faccessat", syscallEnosys4},
		{"__syscall_fcntl64", syscallEnosys3},
		{"__syscall_fdatasync", syscallEnosys1},
		{"__syscall_fstat64", syscallEnosys2},
		{"__syscall_getcwd", syscallEnosys2},
		{"__syscall_getdents64", syscallEnosys3},
		{"__syscall_ioctl", syscallEnosys3},
		{"__syscall_lstat64", syscallEnosys2},
		{"__syscall_mkdirat", syscallEnosys3},
		{"__syscall_newfstatat", syscallEnosys4},
		{"__syscall_openat", syscallEnosys4},
		{"__syscall_pipe", syscallEnosys1},
		{"__syscall_poll", syscallEnosys3},
		{"__syscall_readlinkat", syscallEnosys4},
		{"__syscall_renameat", syscallEnosys4},
		{"__syscall_rmdir", syscallEnosys1},
		{"__syscall_stat64", syscallEnosys2},
		{"__syscall_symlinkat", syscallEnosys3},
		{"__syscall_unlinkat", syscallEnosys3},
		// 6-param socket syscalls: (i32, i32, i32, i32, i32, i32) -> i32
		{"__syscall_bind", syscallEnosys6},
		{"__syscall_connect", syscallEnosys6},
		{"__syscall_getsockname", syscallEnosys6},
		{"__syscall_getsockopt", syscallEnosys6},
		{"__syscall_recvfrom", syscallEnosys6},
		{"__syscall_sendto", syscallEnosys6},
		{"__syscall_socket", syscallEnosys6},
	} {
		// Capture loop variable for the closure.
		d := def
		if err := linker.DefineFunc(store, "env", d.name, d.fn); err != nil {
			return fmt.Errorf("embedded/abi: %s: %w", d.name, err)
		}
	}

	// Syscalls with i64 parameters — cannot use the uniform i32 stubs.
	// Signatures confirmed from pglite v0.2.17 WASM type section.

	// __syscall_fadvise64: (i32, i64, i64, i32) -> i32
	if err := linker.DefineFunc(store, "env", "__syscall_fadvise64",
		func(fd int32, offset, length int64, advice int32) int32 { return enosys }); err != nil {
		return fmt.Errorf("embedded/abi: __syscall_fadvise64: %w", err)
	}
	// __syscall_fallocate: (i32, i32, i64, i64) -> i32
	if err := linker.DefineFunc(store, "env", "__syscall_fallocate",
		func(fd, mode int32, offset, length int64) int32 { return enosys }); err != nil {
		return fmt.Errorf("embedded/abi: __syscall_fallocate: %w", err)
	}
	// __syscall_ftruncate64: (i32, i64) -> i32
	if err := linker.DefineFunc(store, "env", "__syscall_ftruncate64",
		func(fd int32, length int64) int32 { return enosys }); err != nil {
		return fmt.Errorf("embedded/abi: __syscall_ftruncate64: %w", err)
	}
	// __syscall_truncate64: (i32, i64) -> i32
	if err := linker.DefineFunc(store, "env", "__syscall_truncate64",
		func(path int32, length int64) int32 { return enosys }); err != nil {
		return fmt.Errorf("embedded/abi: __syscall_truncate64: %w", err)
	}

	return nil
}
