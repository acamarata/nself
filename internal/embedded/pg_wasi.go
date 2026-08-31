//go:build cgo

package embedded

// pg_wasi.go — a from-scratch WASI preview1 shim bound to the host-owned
// env::memory extern (defineEmscriptenGlobalsMemoryTable's mem), replacing
// wasmtime's built-in linker.DefineWasi().
//
// Why: wasmtime's linker.DefineWasi() resolves guest memory by looking up
// the INSTANCE's exported memory named "memory" the first time any WASI
// function actually runs. pglite v0.2.17 is an Emscripten SIDE_MODULE: it
// IMPORTS env::memory (created host-side, see emscripten_abi_globals.go)
// and exports nothing named "memory" at all. The first WASI call then fails
// with "missing required memory export" — the exact next blocker documented
// in project_ci_findings_2026_08_17.md and reproduced live in
// P6-E11-W2-S1-T5 (2026-08-31): `linker.DefineWasi()` + Instantiate()
// succeed, but the first call into a WASI-provided function during
// __main_argc_argv traps with that message.
//
// pglite v0.2.17 imports only 12 wasi_snapshot_preview1 functions (confirmed
// via wasmtime.Module.Imports() against the cached pglite.wasm, 2026-08-31 —
// NOT the ~40-function preview1 surface DefineWasi() provides): clock_time_get,
// environ_get, environ_sizes_get, fd_close, fd_fdstat_get, fd_pread,
// fd_pwrite, fd_read, fd_seek, fd_sync, fd_write, proc_exit. Notably absent:
// path_open, fd_prestat_get, fd_prestat_dir_name, args_get/args_sizes_get,
// random_get. This matches emscripten_abi_syscalls.go's __syscall_openat
// (and every other path-taking syscall) being permanently stubbed to
// ENOSYS: real file-path I/O never reaches WASI at all in this build —
// pglite's own in-WASM VFS keeps Postgres's data purely in linear memory.
// The only fds actually exercised are 0 (stdin, unused), 1 (stdout), and 2
// (stderr); every other fd returns EBADF, matching the no-path-open reality
// rather than pretending general file I/O works.
//
// Inputs: *wasmtime.Linker, *wasmtime.Store, the host-owned *wasmtime.Memory
// from defineEmscriptenABI, and the stdout/stderr writers boot() opens
// (replacing the WasiConfig-based file redirection this shim makes
// unreachable — WasiConfig only feeds wasmtime's own DefineWasi()).
// Outputs: error — first definition failure aborts, matching
// defineEmscriptenABI's convention.
// Constraints: never calls linker.DefineWasi(); every function below is a
// full replacement, not a wrapper around it.

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v25"
)

// WASI preview1 errno values this shim returns. Full table:
// https://github.com/WebAssembly/WASI/blob/main/legacy/preview1/docs.md
const (
	wasiErrnoSuccess = int32(0)
	wasiErrnoBadf    = int32(8)  // EBADF — fd is not 0/1/2
	wasiErrnoSpipe   = int32(29) // ESPIPE — seek on a non-seekable fd (stdio)
)

// definePGWasi registers the 12 wasi_snapshot_preview1 imports pglite
// v0.2.17 actually declares, bound to mem instead of an instance export.
// stdout and stderr receive fd_write output for fds 1 and 2 respectively;
// every other fd is treated as absent (EBADF) since pglite never opens one
// through WASI (see file header).
func definePGWasi(linker *wasmtime.Linker, store *wasmtime.Store, mem *wasmtime.Memory, stdout, stderr io.Writer) error {
	const mod = "wasi_snapshot_preview1"

	writerFor := func(fd int32) io.Writer {
		switch fd {
		case 1:
			return stdout
		case 2:
			return stderr
		default:
			return nil
		}
	}

	// fd_write(fd, iovs_ptr, iovs_len, nwritten_ptr) -> errno. Gathers each
	// iovec (8 bytes: buf_ptr u32, buf_len u32) and writes it to the fd's
	// backing writer, matching the WASI scatter/gather contract.
	if err := linker.DefineFunc(store, mod, "fd_write",
		func(fd, iovsPtr, iovsLen, nwrittenPtr int32) int32 {
			w := writerFor(fd)
			if w == nil {
				return wasiErrnoBadf
			}
			data := mem.UnsafeData(store)
			var total uint32
			for i := int32(0); i < iovsLen; i++ {
				base := int(iovsPtr) + int(i)*8
				if base < 0 || base+8 > len(data) {
					return wasiErrnoBadf
				}
				bufPtr := binary.LittleEndian.Uint32(data[base : base+4])
				bufLen := binary.LittleEndian.Uint32(data[base+4 : base+8])
				if int(bufPtr+bufLen) > len(data) {
					return wasiErrnoBadf
				}
				n, err := w.Write(data[bufPtr : bufPtr+bufLen])
				total += uint32(n) //nolint:gosec // n <= bufLen, a WASM-address-range value
				if err != nil {
					return wasiErrnoBadf
				}
			}
			if int(nwrittenPtr)+4 > len(data) {
				return wasiErrnoBadf
			}
			binary.LittleEndian.PutUint32(data[nwrittenPtr:nwrittenPtr+4], total)
			return wasiErrnoSuccess
		}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_write: %w", err)
	}

	// fd_read(fd, iovs_ptr, iovs_len, nread_ptr) -> errno. Only fd 0 (stdin)
	// is a valid target and always reports EOF (0 bytes) — the embedded
	// runtime has no interactive stdin to serve reads from.
	if err := linker.DefineFunc(store, mod, "fd_read",
		func(fd, iovsPtr, iovsLen, nreadPtr int32) int32 {
			if fd != 0 {
				return wasiErrnoBadf
			}
			data := mem.UnsafeData(store)
			if int(nreadPtr)+4 > len(data) {
				return wasiErrnoBadf
			}
			binary.LittleEndian.PutUint32(data[nreadPtr:nreadPtr+4], 0)
			return wasiErrnoSuccess
		}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_read: %w", err)
	}

	// fd_pread / fd_pwrite: no fd this runtime exposes supports positional
	// I/O — no real file is ever opened through WASI (see file header) — so
	// both always report EBADF.
	if err := linker.DefineFunc(store, mod, "fd_pread",
		func(fd, iovsPtr, iovsLen int32, offset int64, nreadPtr int32) int32 {
			return wasiErrnoBadf
		}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_pread: %w", err)
	}
	if err := linker.DefineFunc(store, mod, "fd_pwrite",
		func(fd, iovsPtr, iovsLen int32, offset int64, nwrittenPtr int32) int32 {
			return wasiErrnoBadf
		}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_pwrite: %w", err)
	}

	// fd_seek(fd, offset, whence, newoffset_ptr) -> errno. stdio fds are
	// non-seekable (ESPIPE); anything else is EBADF.
	if err := linker.DefineFunc(store, mod, "fd_seek",
		func(fd int32, offset int64, whence int32, newoffsetPtr int32) int32 {
			if fd >= 0 && fd <= 2 {
				return wasiErrnoSpipe
			}
			return wasiErrnoBadf
		}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_seek: %w", err)
	}

	// fd_close / fd_sync: stdio fds no-op successfully (this runtime does
	// not actually own fds 0-2 to close/flush); anything else is EBADF.
	if err := linker.DefineFunc(store, mod, "fd_close", func(fd int32) int32 {
		if fd >= 0 && fd <= 2 {
			return wasiErrnoSuccess
		}
		return wasiErrnoBadf
	}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_close: %w", err)
	}
	if err := linker.DefineFunc(store, mod, "fd_sync", func(fd int32) int32 {
		if fd >= 0 && fd <= 2 {
			return wasiErrnoSuccess
		}
		return wasiErrnoBadf
	}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_sync: %w", err)
	}

	// fd_fdstat_get(fd, stat_ptr) -> errno. Writes a 24-byte __wasi_fdstat_t:
	// fs_filetype(u8) + 3 pad + fs_flags(u16) + 2 pad + fs_rights_base(u64) +
	// fs_rights_inheriting(u64). filetype 2 = character_device, matching a
	// terminal/log stream; rights are set to all bits since nothing in this
	// shim enforces rights checks on a call.
	if err := linker.DefineFunc(store, mod, "fd_fdstat_get",
		func(fd, statPtr int32) int32 {
			if fd < 0 || fd > 2 {
				return wasiErrnoBadf
			}
			data := mem.UnsafeData(store)
			if statPtr < 0 || int(statPtr)+24 > len(data) {
				return wasiErrnoBadf
			}
			buf := data[statPtr : statPtr+24]
			for i := range buf {
				buf[i] = 0
			}
			buf[0] = 2                                            // fs_filetype = character_device
			binary.LittleEndian.PutUint64(buf[8:16], ^uint64(0))  // fs_rights_base
			binary.LittleEndian.PutUint64(buf[16:24], ^uint64(0)) // fs_rights_inheriting
			return wasiErrnoSuccess
		}); err != nil {
		return fmt.Errorf("embedded/wasi: fd_fdstat_get: %w", err)
	}

	// clock_time_get(clock_id, precision, result_ptr) -> errno. clock_id is
	// ignored — realtime and monotonic both collapse to the host wall
	// clock, which is all pglite needs for internal timestamps here.
	if err := linker.DefineFunc(store, mod, "clock_time_get",
		func(clockID int32, precision int64, resultPtr int32) int32 {
			data := mem.UnsafeData(store)
			if resultPtr < 0 || int(resultPtr)+8 > len(data) {
				return wasiErrnoBadf
			}
			binary.LittleEndian.PutUint64(data[resultPtr:resultPtr+8], uint64(time.Now().UnixNano())) //nolint:gosec // wall-clock value, not security sensitive
			return wasiErrnoSuccess
		}); err != nil {
		return fmt.Errorf("embedded/wasi: clock_time_get: %w", err)
	}

	// environ_sizes_get / environ_get: the embedded runtime passes no
	// environment to pglite (matching __main_argc_argv(0, 0) — no argv
	// either), so both report zero entries.
	if err := linker.DefineFunc(store, mod, "environ_sizes_get",
		func(countPtr, bufSizePtr int32) int32 {
			data := mem.UnsafeData(store)
			if countPtr < 0 || int(countPtr)+4 > len(data) || bufSizePtr < 0 || int(bufSizePtr)+4 > len(data) {
				return wasiErrnoBadf
			}
			binary.LittleEndian.PutUint32(data[countPtr:countPtr+4], 0)
			binary.LittleEndian.PutUint32(data[bufSizePtr:bufSizePtr+4], 0)
			return wasiErrnoSuccess
		}); err != nil {
		return fmt.Errorf("embedded/wasi: environ_sizes_get: %w", err)
	}
	if err := linker.DefineFunc(store, mod, "environ_get",
		func(environPtr, environBufPtr int32) int32 {
			return wasiErrnoSuccess // nothing to write; sizes_get already reported 0
		}); err != nil {
		return fmt.Errorf("embedded/wasi: environ_get: %w", err)
	}

	// proc_exit(code) never returns to its caller per spec. code 0 is a
	// graceful shutdown (mirrors env::exit's convention in
	// emscripten_abi_runtime.go); any code panics so wasmtime surfaces it as
	// a trap the boot goroutine reports through exitCh instead of silently
	// discarding it.
	if err := linker.DefineFunc(store, mod, "proc_exit", func(code int32) {
		panic(fmt.Sprintf("pglite: proc_exit(%d)", code))
	}); err != nil {
		return fmt.Errorf("embedded/wasi: proc_exit: %w", err)
	}

	return nil
}
