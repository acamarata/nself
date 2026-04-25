package license

import "testing"

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
