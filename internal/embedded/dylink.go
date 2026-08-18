package embedded

// dylink.go — Minimal reader for the WebAssembly `dylink.0` custom section.
//
// Purpose: Recover the memory layout an Emscripten SIDE_MODULE expects, so the
//          host can place __heap_base correctly instead of leaving it at 0.
// Inputs:  raw pglite.wasm bytes.
// Outputs: dylinkMemInfo{MemorySize, MemoryAlignment, TableSize, TableAlignment}.
// Constraints: read-only, allocation-light, tolerant of unknown subsections and
//              of modules that carry no dylink section at all.
//
// WHY this exists
//
// pglite v0.2.17 ships as a SIDE_MODULE. Its imports include GOT.mem::__heap_base,
// which a real Emscripten dynamic linker writes before any module code runs.
// wasmtime has no such linker, so whatever the host defines is what the module
// gets — permanently. Defining it as 0 puts the heap on top of the static data
// segment at address 0, and address arithmetic derived from it lands outside
// linear memory.
//
// The module states its own requirement in the `dylink.0` section
// (WASM_DYLINK_MEM_INFO): how many bytes of static memory it needs and at what
// alignment. Reading it keeps the layout correct across pglite upgrades rather
// than hardcoding a constant that silently rots on the next bump.
//
// Format (https://github.com/WebAssembly/tool-conventions/blob/main/DynamicLinking.md):
//
//	custom section, name "dylink.0", body = sequence of subsections
//	subsection: u8 id, uleb128 size, payload
//	id 1 = WASM_DYLINK_MEM_INFO:
//	    uleb128 memorysize, memoryalignment, tablesize, tablealignment
//	(alignments are log2 values)

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// wasmDylinkMemInfo is the subsection id for WASM_DYLINK_MEM_INFO.
const wasmDylinkMemInfo = 1

// errNoDylink reports that the module carries no dylink.0 section, i.e. it is
// not a SIDE_MODULE and needs no host-side memory placement.
var errNoDylink = errors.New("embedded/dylink: no dylink.0 section")

// dylinkMemInfo is the memory/table footprint a SIDE_MODULE declares.
type dylinkMemInfo struct {
	MemorySize      uint32 // bytes of static memory the module occupies
	MemoryAlignment uint32 // log2 alignment for that region
	TableSize       uint32 // function-table entries required
	TableAlignment  uint32 // log2 alignment for the table
}

// HeapBase returns the first address past the module's static data, rounded up
// to the declared alignment. This is the value an Emscripten dynamic linker
// would write into GOT.mem::__heap_base.
func (d dylinkMemInfo) HeapBase() uint32 {
	align := uint32(1) << d.MemoryAlignment
	if align == 0 {
		return d.MemorySize
	}
	// Round up to the next multiple of align.
	return (d.MemorySize + align - 1) &^ (align - 1)
}

// readUleb128 decodes a LEB128 unsigned integer at off, returning the value and
// the offset just past it.
func readUleb128(b []byte, off int) (uint32, int, error) {
	var result uint32
	var shift uint
	for {
		if off >= len(b) {
			return 0, 0, fmt.Errorf("embedded/dylink: truncated LEB128 at offset %d", off)
		}
		c := b[off]
		off++
		result |= uint32(c&0x7f) << shift
		if c&0x80 == 0 {
			return result, off, nil
		}
		shift += 7
		if shift > 31 {
			return 0, 0, fmt.Errorf("embedded/dylink: LEB128 overflows uint32 at offset %d", off)
		}
	}
}

// parseDylinkMemInfo extracts WASM_DYLINK_MEM_INFO from a wasm module.
// It returns errNoDylink when the module has no dylink.0 section.
func parseDylinkMemInfo(mod []byte) (dylinkMemInfo, error) {
	var zero dylinkMemInfo

	if len(mod) < 8 || string(mod[0:4]) != "\x00asm" {
		return zero, fmt.Errorf("embedded/dylink: not a wasm module")
	}
	if v := binary.LittleEndian.Uint32(mod[4:8]); v != 1 {
		return zero, fmt.Errorf("embedded/dylink: unsupported wasm version %d", v)
	}

	off := 8
	for off < len(mod) {
		if off >= len(mod) {
			break
		}
		sectionID := mod[off]
		off++

		size, next, err := readUleb128(mod, off)
		if err != nil {
			return zero, err
		}
		off = next
		end := off + int(size)
		if end > len(mod) || end < off {
			return zero, fmt.Errorf("embedded/dylink: section length %d overruns module", size)
		}

		// Only custom sections (id 0) carry dylink.0.
		if sectionID != 0 {
			off = end
			continue
		}

		nameLen, p, err := readUleb128(mod, off)
		if err != nil {
			return zero, err
		}
		if p+int(nameLen) > end {
			return zero, fmt.Errorf("embedded/dylink: custom section name overruns section")
		}
		name := string(mod[p : p+int(nameLen)])
		p += int(nameLen)

		if name != "dylink.0" {
			off = end
			continue
		}

		// Walk subsections looking for MEM_INFO.
		for p < end {
			subID := mod[p]
			p++
			subLen, q, err := readUleb128(mod, p)
			if err != nil {
				return zero, err
			}
			p = q
			subEnd := p + int(subLen)
			if subEnd > end || subEnd < p {
				return zero, fmt.Errorf("embedded/dylink: subsection length %d overruns section", subLen)
			}

			if subID != wasmDylinkMemInfo {
				p = subEnd
				continue
			}

			var info dylinkMemInfo
			cur := p
			for _, field := range []*uint32{
				&info.MemorySize, &info.MemoryAlignment,
				&info.TableSize, &info.TableAlignment,
			} {
				v, nxt, err := readUleb128(mod, cur)
				if err != nil {
					return zero, err
				}
				*field = v
				cur = nxt
			}
			return info, nil
		}

		off = end
	}

	return zero, errNoDylink
}
