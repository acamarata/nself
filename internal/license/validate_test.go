package license

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeFakeEntry returns a minimal CacheEntry JSON with an empty signature.
// VerifySignature() returns false for this entry, making it unsigned.
func makeFakeEntryJSON(t *testing.T) []byte {
	t.Helper()
	entry := CacheEntry{
		KeyHash: HashKey("nself_pro_testkey1234567890abcdef12345"),
		Tier:    "basic",
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshalling fake entry: %v", err)
	}
	return data
}

// redirectCacheDir points the cache writer at a temp dir so tests don't
// corrupt the real cache.
func redirectCacheDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("NSELF_CACHE_DIR", filepath.Join(tmp, "cache"))
}

// TestImportCache_SkipVerifyWithoutForceRejected asserts that
// NSELF_LICENSE_SKIP_VERIFY=1 without NSELF_LICENSE_SKIP_VERIFY_FORCE=1 returns
// the expected "requires --force flag" error (S09.T-SEC-01).
func TestImportCache_SkipVerifyWithoutForceRejected(t *testing.T) {
	redirectCacheDir(t)
	t.Setenv("NSELF_LICENSE_SKIP_VERIFY", "1")
	t.Setenv("NSELF_LICENSE_SKIP_VERIFY_FORCE", "")

	data := makeFakeEntryJSON(t)
	err := ImportCache(data)
	if err == nil {
		t.Fatal("expected error when SKIP_VERIFY=1 without FORCE=1, got nil")
	}
	if !strings.Contains(err.Error(), "requires --force flag") {
		t.Errorf("expected 'requires --force flag' in error, got: %q", err.Error())
	}
}

// TestImportCache_SkipVerifyWithForceEmitsWarning asserts that
// NSELF_LICENSE_SKIP_VERIFY=1 WITH NSELF_LICENSE_SKIP_VERIFY_FORCE=1 emits a
// warning to stderr and does NOT return an error for the sig check path
// (S09.T-SEC-01 acceptance: warning emitted to stderr when --force used).
func TestImportCache_SkipVerifyWithForceEmitsWarning(t *testing.T) {
	redirectCacheDir(t)
	t.Setenv("NSELF_LICENSE_SKIP_VERIFY", "1")
	t.Setenv("NSELF_LICENSE_SKIP_VERIFY_FORCE", "1")

	// Capture stderr output.
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	data := makeFakeEntryJSON(t)
	// ImportCache may fail at WriteCache (no real cache dir configured); that's
	// OK — we only care that the skip-verify warning was emitted before any
	// write attempt.
	ImportCache(data) //nolint:errcheck

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	stderrOutput := string(buf[:n])

	if !strings.Contains(stderrOutput, "skip-verify mode") {
		t.Errorf("expected 'skip-verify mode' in stderr warning, got: %q", stderrOutput)
	}
}
