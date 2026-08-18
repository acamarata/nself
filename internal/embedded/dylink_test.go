package embedded

// dylink_test.go — Tests for the dylink.0 reader.
//
// Purpose: Verify WASM_DYLINK_MEM_INFO is decoded correctly, that heap-base
//          alignment matches the module's declared requirement, and that
//          malformed or non-SIDE_MODULE input is rejected rather than silently
//          producing a plausible-but-wrong address.
// Inputs:  hand-built minimal wasm modules; optionally the real cached pglite.
// Outputs: assertions on dylinkMemInfo and HeapBase().
// Constraints: no network; the pglite case skips when the module is not cached.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// uleb encodes v as LEB128, matching the wasm encoding the parser expects.
func uleb(v uint32) []byte {
	var out []byte
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			out = append(out, c|0x80)
			continue
		}
		out = append(out, c)
		return out
	}
}

// buildModuleWithDylink assembles a minimal valid wasm module carrying a
// dylink.0 custom section with the given MEM_INFO values.
func buildModuleWithDylink(memSize, memAlign, tabSize, tabAlign uint32) []byte {
	var memInfo []byte
	memInfo = append(memInfo, uleb(memSize)...)
	memInfo = append(memInfo, uleb(memAlign)...)
	memInfo = append(memInfo, uleb(tabSize)...)
	memInfo = append(memInfo, uleb(tabAlign)...)

	var sub []byte
	sub = append(sub, wasmDylinkMemInfo)
	sub = append(sub, uleb(uint32(len(memInfo)))...)
	sub = append(sub, memInfo...)

	name := "dylink.0"
	var body []byte
	body = append(body, uleb(uint32(len(name)))...)
	body = append(body, name...)
	body = append(body, sub...)

	mod := []byte{0x00, 'a', 's', 'm', 0x01, 0x00, 0x00, 0x00}
	mod = append(mod, 0x00) // custom section id
	mod = append(mod, uleb(uint32(len(body)))...)
	mod = append(mod, body...)
	return mod
}

func TestParseDylinkMemInfo_ReadsDeclaredValues(t *testing.T) {
	t.Parallel()

	mod := buildModuleWithDylink(2172900, 12, 5359, 0)
	got, err := parseDylinkMemInfo(mod)
	if err != nil {
		t.Fatalf("parseDylinkMemInfo: %v", err)
	}
	if got.MemorySize != 2172900 || got.MemoryAlignment != 12 ||
		got.TableSize != 5359 || got.TableAlignment != 0 {
		t.Errorf("got %+v, want {2172900 12 5359 0}", got)
	}
}

func TestDylinkMemInfo_HeapBaseAlignsUp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		size, align, want uint32
	}{
		// pglite v0.2.17: 2172900 is not 4096-aligned, so it rounds up.
		{"pglite-v0.2.17", 2172900, 12, 2174976},
		{"already-aligned", 8192, 12, 8192},
		{"align-1-byte", 1234, 0, 1234},
		{"align-16", 1000, 4, 1008},
		{"zero-size", 0, 12, 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := dylinkMemInfo{MemorySize: tc.size, MemoryAlignment: tc.align}
			if got := d.HeapBase(); got != tc.want {
				t.Errorf("HeapBase() = %d (0x%x), want %d (0x%x)", got, got, tc.want, tc.want)
			}
		})
	}
}

// TestDylinkMemInfo_HeapBaseIsPastStaticData is the property that actually
// matters: the heap must never start inside the module's static data, or the
// allocator hands out addresses that overwrite it.
func TestDylinkMemInfo_HeapBaseIsPastStaticData(t *testing.T) {
	t.Parallel()

	for _, size := range []uint32{1, 4095, 4096, 4097, 2172900} {
		d := dylinkMemInfo{MemorySize: size, MemoryAlignment: 12}
		if hb := d.HeapBase(); hb < size {
			t.Errorf("HeapBase() = %d for memorysize %d — heap would overlap static data", hb, size)
		}
	}
}

func TestParseDylinkMemInfo_NoDylinkSection(t *testing.T) {
	t.Parallel()

	// A bare, valid module header with no sections at all.
	mod := []byte{0x00, 'a', 's', 'm', 0x01, 0x00, 0x00, 0x00}
	if _, err := parseDylinkMemInfo(mod); !errors.Is(err, errNoDylink) {
		t.Errorf("err = %v, want errNoDylink", err)
	}
}

func TestParseDylinkMemInfo_RejectsMalformed(t *testing.T) {
	t.Parallel()

	cases := map[string][]byte{
		"empty":        {},
		"bad magic":    {0x01, 0x02, 0x03, 0x04, 0x01, 0x00, 0x00, 0x00},
		"bad version":  {0x00, 'a', 's', 'm', 0x09, 0x00, 0x00, 0x00},
		"truncated":    {0x00, 'a', 's', 'm'},
		"overrun size": {0x00, 'a', 's', 'm', 0x01, 0x00, 0x00, 0x00, 0x00, 0xff, 0x7f},
	}
	for name, mod := range cases {
		name, mod := name, mod
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseDylinkMemInfo(mod); err == nil {
				t.Error("expected an error for malformed input, got nil")
			}
		})
	}
}

// TestParseDylinkMemInfo_SkipsUnknownSubsections ensures a future dylink
// revision adding subsections before MEM_INFO does not break parsing.
func TestParseDylinkMemInfo_SkipsUnknownSubsections(t *testing.T) {
	t.Parallel()

	var memInfo []byte
	for _, v := range []uint32{4096, 12, 10, 0} {
		memInfo = append(memInfo, uleb(v)...)
	}

	var sub []byte
	// Unknown subsection id 7 with a 3-byte payload, ahead of MEM_INFO.
	sub = append(sub, 0x07)
	sub = append(sub, uleb(3)...)
	sub = append(sub, 0xAA, 0xBB, 0xCC)
	sub = append(sub, wasmDylinkMemInfo)
	sub = append(sub, uleb(uint32(len(memInfo)))...)
	sub = append(sub, memInfo...)

	name := "dylink.0"
	var body []byte
	body = append(body, uleb(uint32(len(name)))...)
	body = append(body, name...)
	body = append(body, sub...)

	mod := []byte{0x00, 'a', 's', 'm', 0x01, 0x00, 0x00, 0x00, 0x00}
	mod = append(mod, uleb(uint32(len(body)))...)
	mod = append(mod, body...)

	got, err := parseDylinkMemInfo(mod)
	if err != nil {
		t.Fatalf("parseDylinkMemInfo: %v", err)
	}
	if got.MemorySize != 4096 || got.TableSize != 10 {
		t.Errorf("got %+v, want memsize 4096 / tablesize 10", got)
	}
}

// TestParseDylinkMemInfo_RealPglite pins the values against the actual module
// when it is cached locally. These are the numbers the host must reproduce.
func TestParseDylinkMemInfo_RealPglite(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	path := filepath.Join(home, ".nself", "cache", "pglite", DefaultPGliteVersion, "pglite.wasm")
	raw, err := os.ReadFile(path) //nolint:gosec // developer-local cache path
	if err != nil {
		t.Skipf("pglite not cached at %s", path)
	}

	got, err := parseDylinkMemInfo(raw)
	if err != nil {
		t.Fatalf("parseDylinkMemInfo(real pglite): %v", err)
	}
	if got.MemorySize == 0 {
		t.Fatal("memorysize 0 — pglite is a SIDE_MODULE and must declare static memory")
	}
	if hb := got.HeapBase(); hb <= got.MemorySize-1 {
		t.Errorf("HeapBase() = %d does not clear memorysize %d", hb, got.MemorySize)
	}
	t.Logf("pglite %s: memorysize=%d memalign=%d tablesize=%d heapBase=0x%x",
		DefaultPGliteVersion, got.MemorySize, got.MemoryAlignment, got.TableSize, got.HeapBase())
}
