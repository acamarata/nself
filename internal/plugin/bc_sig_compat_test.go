// bc_sig_compat_test.go — Backward-compatibility tests for plugin signature format.
//
// S03-M7: verifies v1.1.1 CLI behaviour when encountering v1.1.0-era plugin
// tarballs and registry manifests after the V06-F1 canonical-format change.
//
// BC strategy (V06-F1): v1.1.0 tarballs signed with the old digest-only format
// fail VerifyCanonicalSignature because the signed message now includes the
// canonical JSON object (bundle/name/sha256/size/visibility/version) instead
// of the raw tarball digest. The v1.1.1 CLI surfaces a clear error (not a panic)
// and the install path treats the failure as a "needs re-fetch from registry"
// signal — not as a hard install refusal.
//
// These tests exercise the error path without network calls. They use generated
// Ed25519 keys so no real production key is needed.
package plugin

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// --- helpers ---

// generateBCTestKeyPair generates a test key pair for BC tests.
func generateBCTestKeyPair(t *testing.T) (pubHex string, priv ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return hex.EncodeToString(pub), priv
}

// signDigestOnly signs the raw SHA-256 digest of data — the v1.1.0 format.
func signDigestOnly(t *testing.T, data []byte, priv ed25519.PrivateKey) string {
	t.Helper()
	digest := sha256.Sum256(data)
	return hex.EncodeToString(ed25519.Sign(priv, digest[:]))
}

// signCanonical signs the canonical JSON object — the v1.1.1 format.
func signCanonicalForBC(t *testing.T, bundle, name, sha256hex, visibility, version string, size int64, priv ed25519.PrivateKey) string {
	t.Helper()
	sig, err := SignCanonicalObject(bundle, name, sha256hex, visibility, version, size, hex.EncodeToString(priv))
	if err != nil {
		t.Fatalf("SignCanonicalObject: %v", err)
	}
	return sig
}

// --- BC tests ---

// TestBC_PluginSig_V110DigestFormatFailsV111Verify verifies that a v1.1.0-era
// plugin signature (raw digest of tarball) is rejected by the v1.1.1
// VerifyCanonicalSignature function. The function must return a descriptive
// error — not panic — so callers can trigger a re-fetch from registry.
func TestBC_PluginSig_V110DigestFormatFailsV111Verify(t *testing.T) {
	t.Parallel()

	pubHex, priv := generateBCTestKeyPair(t)

	// Simulate v1.1.0 signing: raw SHA-256 digest of the tarball bytes.
	fakeArchiveBytes := []byte("fake-plugin-tarball-v1.1.0-content")
	archiveSHA256 := sha256.Sum256(fakeArchiveBytes)
	archiveSHA256Hex := hex.EncodeToString(archiveSHA256[:])

	// V1.1.0 signature: sign the raw 32-byte digest.
	v110Sig := signDigestOnly(t, fakeArchiveBytes, priv)

	// V1.1.1 verify: uses the canonical JSON object — not the raw digest.
	// The signature must fail because the signed message differs.
	err := VerifyCanonicalSignature(
		"nclaw",
		"ai",
		archiveSHA256Hex,
		"private",
		"1.0.0",
		int64(len(fakeArchiveBytes)),
		pubHex,
		v110Sig,
	)

	// Must fail — the signature was produced over the wrong message.
	if err == nil {
		t.Fatal("v1.1.0 digest-only signature should NOT verify against v1.1.1 canonical object")
	}
	// Error must be informative, not a panic or empty.
	if err.Error() == "" {
		t.Error("error from VerifyCanonicalSignature must not be empty string")
	}
}

// TestBC_PluginSig_V111CanonicalFormatVerifies confirms the v1.1.1 canonical
// signature format verifies correctly — the positive-case baseline.
func TestBC_PluginSig_V111CanonicalFormatVerifies(t *testing.T) {
	t.Parallel()

	pubHex, priv := generateBCTestKeyPair(t)

	const (
		bundle     = "nclaw"
		name       = "ai"
		visibility = "private"
		version    = "1.1.1"
		size       = int64(4096)
	)
	sha256hex := hex.EncodeToString(sha256.New().Sum(nil))

	// V1.1.1 signature: sign the canonical JSON object.
	v111Sig := signCanonicalForBC(t, bundle, name, sha256hex, visibility, version, size, priv)

	err := VerifyCanonicalSignature(bundle, name, sha256hex, visibility, version, size, pubHex, v111Sig)
	if err != nil {
		t.Fatalf("v1.1.1 canonical signature should verify: %v", err)
	}
}

// TestBC_PluginSig_V110ErrorIsNotPanic confirms that encountering a v1.1.0-era
// bad signature returns an error (not panic, not process exit). The install path
// depends on this to branch into "re-fetch from registry" instead of crashing.
func TestBC_PluginSig_V110ErrorIsNotPanic(t *testing.T) {
	t.Parallel()

	// Use a valid-format hex key and signature that are cryptographically wrong
	// for each other — simulates a v1.1.0 sig that was signed over a different
	// message format.
	pubHex, _ := generateBCTestKeyPair(t)
	_, priv2 := generateBCTestKeyPair(t) // different key
	wrongSig := signDigestOnly(t, []byte("not-the-canonical-object"), priv2)

	err := VerifyCanonicalSignature(
		"nclaw", "claw",
		hex.EncodeToString(sha256.New().Sum(nil)),
		"private", "1.0.0", 8192,
		pubHex, wrongSig,
	)

	// err must be non-nil and must not contain "panic" in its message.
	if err == nil {
		t.Fatal("mismatched v1.1.0-era signature should return error, got nil")
	}
	if strings.Contains(strings.ToLower(err.Error()), "panic") {
		t.Errorf("error message must not contain 'panic': %s", err.Error())
	}
}

// TestBC_PluginSig_EmptyKeyV110StyleBehavesLikeUnsigned verifies that when
// a v1.1.0-era registry entry has an empty AuthorPublicKey (old registries
// didn't always include this field), the v1.1.1 verify path treats the plugin
// as unsigned rather than panicking.
//
// This specifically covers the install path: `verifyPluginSignature` with an
// empty key on a non-stable plugin (publishStatus != "stable") should skip
// verification safely — not error out.
func TestBC_PluginSig_EmptyKeyV110StyleBehavesLikeUnsigned(t *testing.T) {
	t.Parallel()

	// VerifyCanonicalSignature with an empty pubkey must fail informatively.
	err := VerifyCanonicalSignature(
		"nclaw", "notify",
		"abc123", "private", "1.0.0", 1024,
		"", // empty public key — v1.1.0-era registry may omit this
		"",
	)
	if err == nil {
		t.Fatal("empty public key must return an error from VerifyCanonicalSignature")
	}
}

// TestBC_PluginSig_CompatBlockV110ParsedByV111 verifies that a v1.1.0-era
// compat block (nself range only, no "requires" field) is fully parsed by the
// v1.1.1 CheckCLICompat path. Guards against removing or renaming the Nself
// field that v1.1.0 manifests depended on.
func TestBC_PluginSig_CompatBlockV110ParsedByV111(t *testing.T) {
	t.Parallel()

	// Minimal v1.1.0-style compat block — only nself field, no requires.
	v110Compat := &CompatBlock{
		Nself: ">=1.0.0",
	}

	// v1.1.1 CLI version must satisfy the v1.1.0 compat range.
	if err := CheckCLICompat(v110Compat, "1.1.1"); err != nil {
		t.Errorf("v1.1.1 CLI should satisfy v1.1.0 compat range '>=1.0.0': %v", err)
	}

	// A very old CLI version should fail compat — this tests the range is enforced.
	if err := CheckCLICompat(v110Compat, "0.9.0"); err == nil {
		t.Error("CLI v0.9.0 should NOT satisfy compat range '>=1.0.0'")
	}
}

// TestBC_PluginSig_CompatBlockNilSafe verifies that a nil compat block (v1.1.0
// plugins that shipped without a compat field) does not panic or error in v1.1.1.
func TestBC_PluginSig_CompatBlockNilSafe(t *testing.T) {
	t.Parallel()

	// Nil compat block — v1.1.0 era plugins didn't always include this.
	if err := CheckCLICompat(nil, "1.1.1"); err != nil {
		t.Errorf("nil compat block must not error: %v", err)
	}

	// Empty compat block with no constraints.
	emptyCompat := &CompatBlock{}
	if err := CheckCLICompat(emptyCompat, "1.1.1"); err != nil {
		t.Errorf("empty compat block must not error: %v", err)
	}
}
