// bc_compat_test.go — Backward-compatibility tests for the license cache format.
//
// S03-M7: verifies that v1.1.1 CLI code can open and interpret cache files written
// by v1.1.0, and that the graceful-regen path fires when signatures don't match.
//
// Strategy: load fixture from testdata/v1.1.0/license.json, run through the
// v1.1.1 ReadCache + VerifySignature code paths, assert fail-open regen semantics.
package license

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const v110FixtureDir = "testdata/v1.1.0"

// TestBC_LicenseCache_V110CanBeRead verifies that v1.1.1 ReadCache can parse a
// v1.1.0 cache file without errors. The JSON schema is identical between
// versions; any unknown fields must be silently ignored.
func TestBC_LicenseCache_V110CanBeRead(t *testing.T) {
	t.Parallel()

	fixture := filepath.Join(v110FixtureDir, "license.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("reading v1.1.0 fixture: %v", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("v1.1.1 CacheEntry cannot parse v1.1.0 fixture: %v", err)
	}

	// Spot-check required fields preserved.
	if entry.KeyHash == "" {
		t.Error("key_hash must not be empty after parsing v1.1.0 fixture")
	}
	if entry.Tier == "" {
		t.Error("tier must not be empty after parsing v1.1.0 fixture")
	}
	if len(entry.PluginsAllowed) == 0 {
		t.Error("plugins_allowed must not be empty in v1.1.0 fixture")
	}
	if entry.FetchedAt == 0 {
		t.Error("fetched_at must not be zero in v1.1.0 fixture")
	}
}

// TestBC_LicenseCache_V110SigFailsGracefully verifies the fail-open regen
// semantic: a v1.1.0 cache file with a mock/invalid signature will fail
// VerifySignature in production builds but not panic or return a hard error.
// In dev builds (IsZeroPubKey true), VerifySignature returns false and the
// caller must treat this as "needs re-fetch" — not as a crash or fatal error.
func TestBC_LicenseCache_V110SigFailsGracefully(t *testing.T) {
	t.Parallel()

	fixture := filepath.Join(v110FixtureDir, "license.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("reading v1.1.0 fixture: %v", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// VerifySignature must return a bool, never panic.
	// In dev builds (zero pubkey) this returns false — which is the correct
	// signal for the caller to trigger a re-fetch.
	result := entry.VerifySignature()

	// In dev builds (licensePubKeyHex == ""), IsZeroPubKey() is true and
	// VerifySignature returns false. The test only asserts the call doesn't panic.
	_ = result
	// Verify fail-open: even with a bad sig, the CacheEntry fields are readable.
	if len(entry.PluginsAllowed) == 0 {
		t.Error("plugins_allowed must remain readable even when sig fails")
	}
}

// TestBC_LicenseCache_V110RegenPath verifies the regen trigger condition:
// IsZeroPubKey returns true in dev builds (no goreleaser ldflags), so callers
// that check IsZeroPubKey() skip sig verification altogether. This is the
// "dev-build safe path" that also serves as a model for what happens when a
// v1.1.0 cache file is opened on a v1.1.1 binary with a different key_id.
func TestBC_LicenseCache_V110RegenPath(t *testing.T) {
	// t.Setenv requires sequential execution — do NOT call t.Parallel here.

	// In dev builds, licensePubKeyHex is empty → IsZeroPubKey() == true.
	// The BC strategy: callers check IsZeroPubKey() first and bypass sig
	// verification. This ensures no hard crash on v1.1.0 → v1.1.1 upgrade.
	if !IsZeroPubKey() {
		// In goreleaser builds, skip: sig would fail against the mock fixture
		// and the regen path is exercised by production integration tests.
		t.Skip("skipping regen-path test in goreleaser build (non-zero pubkey)")
	}

	// Simulate what the CLI does after upgrading from v1.1.0:
	// 1. Read the old cache file.
	// 2. IsZeroPubKey() → skip sig check → use entry as-is or re-fetch.
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "license.json")
	t.Setenv("LICENSE_CACHE_PATH", cachePath)

	// Write a v1.1.0-style entry.
	v110Entry := &CacheEntry{
		KeyHash:        HashKey("nself_pro_v110testkey"),
		Tier:           "pro",
		PluginsAllowed: []string{"ai", "claw", "mux"},
		FetchedAt:      1715644800,
		ExpiresAt:      1715731200,
		Signature:      "deadbeef", // mock — wouldn't pass real sig verify
		SignatureKeyID: 1,
	}
	if err := WriteCache(v110Entry); err != nil {
		t.Fatalf("WriteCache v1.1.0 style entry: %v", err)
	}

	// v1.1.1 reads it back.
	read, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache on v1.1.0-style entry: %v", err)
	}
	if read == nil {
		t.Fatal("ReadCache returned nil for existing v1.1.0-style entry")
	}

	// Fields survive the round-trip.
	if read.Tier != "pro" {
		t.Errorf("tier mismatch: want pro, got %s", read.Tier)
	}
	if len(read.PluginsAllowed) != 3 {
		t.Errorf("plugins_allowed len: want 3, got %d", len(read.PluginsAllowed))
	}

	// Sig fails (mock data) but no panic — graceful regen signal.
	sigOK := read.VerifySignature()
	// In dev build (IsZeroPubKey true), sig always fails: that's expected and correct.
	if sigOK && IsZeroPubKey() {
		t.Error("VerifySignature should return false in dev build with mock sig")
	}
}

// TestBC_LicenseCache_SchemaFields ensures no required field was removed in
// v1.1.1 that v1.1.0 depended upon. This is a schema regression guard.
func TestBC_LicenseCache_SchemaFields(t *testing.T) {
	t.Parallel()

	// Construct a minimal v1.1.0-era entry using only fields present in v1.1.0.
	raw := `{
		"key_hash":        "abc123",
		"tier":            "basic",
		"plugins_allowed": ["notify"],
		"fetched_at":      1715000000,
		"expires_at":      1715086400,
		"signature":       "00ff",
		"signature_key_id": 1
	}`

	var entry CacheEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		t.Fatalf("v1.1.1 must parse all v1.1.0 JSON fields without error: %v", err)
	}

	if entry.Tier != "basic" {
		t.Errorf("tier: want basic, got %q", entry.Tier)
	}
	if entry.SignatureKeyID != 1 {
		t.Errorf("signature_key_id: want 1, got %d", entry.SignatureKeyID)
	}
}
