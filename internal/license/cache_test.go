package license

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsZeroPubKey verifies IsZeroPubKey detection for dev-build vs
// goreleaser-built binaries.
func TestIsZeroPubKey_EmptyString(t *testing.T) {
	orig := licensePubKeyHex
	defer func() { licensePubKeyHex = orig }()

	licensePubKeyHex = ""
	if !IsZeroPubKey() {
		t.Error("empty string should be detected as zero pubkey (dev build)")
	}
}

func TestIsZeroPubKey_AllZeroHex(t *testing.T) {
	orig := licensePubKeyHex
	defer func() { licensePubKeyHex = orig }()

	// 64 zero chars — common placeholder shape
	licensePubKeyHex = "0000000000000000000000000000000000000000000000000000000000000000"
	if !IsZeroPubKey() {
		t.Error("all-zero hex string should be detected as zero pubkey")
	}
}

func TestIsZeroPubKey_ValidNonZeroHex(t *testing.T) {
	orig := licensePubKeyHex
	defer func() { licensePubKeyHex = orig }()

	// A realistic non-zero Ed25519 pubkey hex (64 hex chars = 32 bytes)
	licensePubKeyHex = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	if IsZeroPubKey() {
		t.Error("non-zero hex string should NOT be detected as zero pubkey (goreleaser build)")
	}
}

func TestIsZeroPubKey_SingleNonZeroChar(t *testing.T) {
	orig := licensePubKeyHex
	defer func() { licensePubKeyHex = orig }()

	// Mostly zeros but one non-zero digit — must return false
	licensePubKeyHex = "0000000000000000000000000000000000000000000000000000000000000001"
	if IsZeroPubKey() {
		t.Error("string with one non-zero char should NOT be detected as zero pubkey")
	}
}

// TestWriteCache_AtomicTmpfileNoLeftover ensures the atomic tmpfile + rename
// pattern leaves no stray ".tmp" files in the cache directory after success
// (D3-T10).
func TestWriteCache_AtomicTmpfileNoLeftover(t *testing.T) {
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "license.json")
	t.Setenv("LICENSE_CACHE_PATH", cachePath)

	entry := &CacheEntry{
		KeyHash: HashKey("nself_pro_testkey1234567890abcdef12345"),
		Tier:    "pro",
	}
	if err := WriteCache(entry); err != nil {
		t.Fatalf("WriteCache: %v", err)
	}

	dir := filepath.Dir(cachePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover tmpfile after atomic write: %s", e.Name())
		}
	}

	// Verify the final file is parseable + has 0600 perms.
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("cache file perms = %o, want 0600", perm)
	}
	data, _ := os.ReadFile(cachePath)
	var got CacheEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("cache not parseable: %v", err)
	}
	if got.KeyHash != entry.KeyHash {
		t.Errorf("cache content drift: got %q, want %q", got.KeyHash, entry.KeyHash)
	}
}

// TestWriteCache_AtomicOverwritePreservesContent ensures that overwriting an
// existing cache file with a new entry succeeds and the resulting file is
// fully replaced (no torn write of merged content).
func TestWriteCache_AtomicOverwritePreservesContent(t *testing.T) {
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "license.json")
	t.Setenv("LICENSE_CACHE_PATH", cachePath)

	first := &CacheEntry{KeyHash: HashKey("k1"), Tier: "pro"}
	if err := WriteCache(first); err != nil {
		t.Fatalf("WriteCache 1: %v", err)
	}
	second := &CacheEntry{KeyHash: HashKey("k2"), Tier: "enterprise"}
	if err := WriteCache(second); err != nil {
		t.Fatalf("WriteCache 2: %v", err)
	}

	data, _ := os.ReadFile(cachePath)
	var got CacheEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("cache parse: %v", err)
	}
	if got.Tier != "enterprise" || got.KeyHash != HashKey("k2") {
		t.Errorf("overwrite did not take effect: got %+v", got)
	}
}
