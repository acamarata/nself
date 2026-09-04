package plugin

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/nself-org/cli/internal/errs"
)

// generateTestKeyPair generates a fresh Ed25519 key pair for testing.
func generateTestKeyPair(t *testing.T) (pubKeyHex, privKeyHex string, pub ed25519.PublicKey, priv ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return hex.EncodeToString(pub), hex.EncodeToString(priv), pub, priv
}

// writeTempFile writes content to a temp file and returns its path.
func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "archive-*.bin")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(content); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return f.Name()
}

// signFile computes SHA-256 of content and signs the raw digest with priv.
func signFileContent(t *testing.T, content []byte, priv ed25519.PrivateKey) string {
	t.Helper()
	digest := sha256.Sum256(content)
	sig := ed25519.Sign(priv, digest[:])
	return hex.EncodeToString(sig)
}

// TestVerifyPluginSignature_ValidSignature verifies that a correctly signed
// archive passes verification.
func TestVerifyPluginSignature_ValidSignature(t *testing.T) {
	pubKeyHex, _, _, priv := generateTestKeyPair(t)
	content := []byte("this is fake plugin archive content for testing")
	archivePath := writeTempFile(t, content)
	sigHex := signFileContent(t, content, priv)

	if err := verifyPluginSignature(archivePath, pubKeyHex, sigHex, "stable"); err != nil {
		t.Fatalf("valid signature should pass verification: %v", err)
	}
}

// TestVerifyPluginSignature_TamperedContent verifies that modifying the archive
// after signing causes verification to fail (tamper detection).
func TestVerifyPluginSignature_TamperedContent(t *testing.T) {
	pubKeyHex, _, _, priv := generateTestKeyPair(t)
	originalContent := []byte("original plugin archive content")
	archivePath := writeTempFile(t, originalContent)
	sigHex := signFileContent(t, originalContent, priv)

	// Tamper with the archive after signing.
	if err := os.WriteFile(archivePath, []byte("tampered content!"), 0o644); err != nil {
		t.Fatalf("WriteFile tamper: %v", err)
	}

	err := verifyPluginSignature(archivePath, pubKeyHex, sigHex, "stable")
	if err == nil {
		t.Fatal("tampered archive should fail verification, got nil error")
	}
}

// TestVerifyPluginSignature_WrongKey verifies that a valid signature from a
// different key pair fails verification.
func TestVerifyPluginSignature_WrongKey(t *testing.T) {
	// Sign with key A, verify with key B.
	pubKeyHexA, _, _, _ := generateTestKeyPair(t)
	_, _, _, privB := generateTestKeyPair(t)

	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)
	sigHex := signFileContent(t, content, privB) // signed with B

	err := verifyPluginSignature(archivePath, pubKeyHexA, sigHex, "stable") // verify with A
	if err == nil {
		t.Fatal("wrong public key should fail verification, got nil error")
	}
}

// TestVerifyPluginSignature_MismatchAlwaysRefusesRegardlessOfStatus verifies
// that a PRESENT-but-wrong signature always refuses the install, for every
// publishStatus (including "beta"/"deprecated") — a present-but-wrong
// signature is never acceptable regardless of lifecycle stage, and this
// check does not depend on NSELF_PLUGIN_REQUIRE_CHECKSUM.
func TestVerifyPluginSignature_MismatchAlwaysRefusesRegardlessOfStatus(t *testing.T) {
	pubKeyHexA, _, _, _ := generateTestKeyPair(t)
	_, _, _, privB := generateTestKeyPair(t)
	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)
	wrongSigHex := signFileContent(t, content, privB) // signed with B, verified against A

	for _, status := range []string{"", "stable", "beta", "deprecated"} {
		t.Run("status="+status, func(t *testing.T) {
			err := verifyPluginSignature(archivePath, pubKeyHexA, wrongSigHex, status)
			if err == nil {
				t.Fatalf("present-but-wrong signature must refuse for status %q, got nil", status)
			}
		})
	}
}

// TestVerifyPluginSignature_EmptyInputsDefaultWarnOnly verifies that, by
// default (NSELF_PLUGIN_REQUIRE_CHECKSUM unset), a missing key or signature
// is permitted — never refused — for every status, including an explicit
// "stable" and an absent ("") one. FIX-CLI-6: registry coverage is 47/177 as
// of 2026-09-04, so refusing by default would refuse the other 130 installs.
func TestVerifyPluginSignature_EmptyInputsDefaultWarnOnly(t *testing.T) {
	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)

	cases := []struct {
		name   string
		pubKey string
		sigHex string
	}{
		{"empty public key", "", "aabbcc"},
		{"empty signature", "aabbcc", ""},
		{"both empty", "", ""},
	}
	for _, status := range []string{"stable", "", "beta", "deprecated"} {
		for _, tc := range cases {
			t.Run("status="+status+"/"+tc.name, func(t *testing.T) {
				err := verifyPluginSignature(archivePath, tc.pubKey, tc.sigHex, status)
				if err != nil {
					t.Errorf("default mode: missing signature should warn and proceed (nil), got: %v", err)
				}
			})
		}
	}
}

// TestVerifyPluginSignature_EmptyInputsHardModeRefusesEffectivelyStable
// verifies that with NSELF_PLUGIN_REQUIRE_CHECKSUM=1 set, a missing key or
// signature is refused with ErrPluginUnsigned for both an explicit "stable"
// status and an absent ("") one (EffectiveStatus treats them the same —
// this is the actual FIX-CLI-6 defect: before this fix "" never hit the
// hard-fail branch even in hard mode).
func TestVerifyPluginSignature_EmptyInputsHardModeRefusesEffectivelyStable(t *testing.T) {
	t.Setenv(pluginRequireChecksumEnv, "1")
	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)

	for _, status := range []string{"stable", ""} {
		t.Run("status="+status, func(t *testing.T) {
			err := verifyPluginSignature(archivePath, "", "", status)
			if err == nil {
				t.Fatalf("hard mode: effectively-stable plugin with empty inputs should return ErrPluginUnsigned, got nil")
			}
			if !errors.Is(err, errs.ErrPluginUnsigned) {
				t.Errorf("expected ErrPluginUnsigned, got: %v", err)
			}
		})
	}
}

// TestVerifyPluginSignature_EmptyInputsHardModeNonStableSkip verifies that,
// even in hard mode, beta/deprecated plugins with missing signature still
// skip verification — the hard-mode requirement only applies to
// EffectiveStatus == "stable".
func TestVerifyPluginSignature_EmptyInputsHardModeNonStableSkip(t *testing.T) {
	t.Setenv(pluginRequireChecksumEnv, "1")
	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)

	for _, status := range []string{"alpha", "beta", "deprecated", "experimental"} {
		t.Run("status="+status, func(t *testing.T) {
			if err := verifyPluginSignature(archivePath, "", "", status); err != nil {
				t.Errorf("non-stable plugin with empty signature should skip (return nil) even in hard mode, got: %v", err)
			}
		})
	}
}

// TestVerifyPluginSignature_MalformedKey verifies that a malformed hex public key
// returns a descriptive error.
func TestVerifyPluginSignature_MalformedKey(t *testing.T) {
	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)

	err := verifyPluginSignature(archivePath, "not-valid-hex!!!", "aabbcc", "stable")
	if err == nil {
		t.Fatal("malformed public key hex should return error, got nil")
	}
}

// TestVerifyPluginSignature_WrongKeyLength verifies that a valid hex string with
// wrong byte length returns a descriptive error.
func TestVerifyPluginSignature_WrongKeyLength(t *testing.T) {
	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)

	// Ed25519 public key is 32 bytes; provide 16 bytes.
	shortKey := hex.EncodeToString(make([]byte, 16))
	_, _, _, priv := generateTestKeyPair(t)
	sigHex := signFileContent(t, content, priv)

	err := verifyPluginSignature(archivePath, shortKey, sigHex, "stable")
	if err == nil {
		t.Fatal("wrong key length should return error, got nil")
	}
}

// TestVerifyPluginSignature_MissingArchive verifies that a missing archive file
// returns an error.
func TestVerifyPluginSignature_MissingArchive(t *testing.T) {
	pubKeyHex, _, _, priv := generateTestKeyPair(t)
	sigHex := signFileContent(t, []byte("data"), priv)

	err := verifyPluginSignature("/nonexistent/path/archive.tar.gz", pubKeyHex, sigHex, "stable")
	if err == nil {
		t.Fatal("missing archive should return error, got nil")
	}
}

// TestVerifyPluginSignature_LargeFile verifies correct operation on a larger
// archive (ensures io.Copy path is exercised fully).
func TestVerifyPluginSignature_LargeFile(t *testing.T) {
	pubKeyHex, _, _, priv := generateTestKeyPair(t)

	// 512KB of pseudo-random data to exercise hashing loop.
	content := make([]byte, 512*1024)
	if _, err := io.ReadFull(rand.Reader, content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	archivePath := writeTempFile(t, content)
	sigHex := signFileContent(t, content, priv)

	if err := verifyPluginSignature(archivePath, pubKeyHex, sigHex, "stable"); err != nil {
		t.Fatalf("large file signature verification failed: %v", err)
	}
}

// TestVerifyChecksum_MissingChecksumDefaultWarnOnly verifies that, by
// default (NSELF_PLUGIN_REQUIRE_CHECKSUM unset), a missing checksum is
// permitted — never refused — for every status, including an explicit
// "stable" and an absent ("") one. FIX-CLI-6: registry coverage is 47/177 as
// of 2026-09-04, so refusing by default would refuse the other 130 installs.
func TestVerifyChecksum_MissingChecksumDefaultWarnOnly(t *testing.T) {
	content := []byte("plugin archive content")
	archivePath := writeTempFile(t, content)

	for _, status := range []string{"stable", "", "alpha", "beta", "deprecated", "experimental"} {
		t.Run("status="+status, func(t *testing.T) {
			if err := verifyChecksum(archivePath, "", status); err != nil {
				t.Errorf("default mode: missing checksum should warn and proceed (nil), got: %v", err)
			}
		})
	}
}

// TestVerifyChecksum_MissingChecksumHardModeRefusesEffectivelyStable verifies
// that with NSELF_PLUGIN_REQUIRE_CHECKSUM=1 set, a missing checksum is
// refused with ErrPluginMissingChecksum for both an explicit "stable" status
// and an absent ("") one (EffectiveStatus treats them the same — this is the
// actual FIX-CLI-6 defect: before this fix "" never hit the hard-fail branch
// even in hard mode).
func TestVerifyChecksum_MissingChecksumHardModeRefusesEffectivelyStable(t *testing.T) {
	t.Setenv(pluginRequireChecksumEnv, "1")
	content := []byte("plugin archive content")
	archivePath := writeTempFile(t, content)

	for _, status := range []string{"stable", ""} {
		t.Run("status="+status, func(t *testing.T) {
			err := verifyChecksum(archivePath, "", status)
			if err == nil {
				t.Fatal("hard mode: effectively-stable plugin with empty checksum should return ErrPluginMissingChecksum, got nil")
			}
			if !errors.Is(err, errs.ErrPluginMissingChecksum) {
				t.Errorf("expected ErrPluginMissingChecksum, got: %v", err)
			}
		})
	}
}

// TestVerifyChecksum_MissingChecksumHardModeNonStableSkip verifies that,
// even in hard mode, beta/deprecated plugins with a missing checksum still
// skip verification — the hard-mode requirement only applies to
// EffectiveStatus == "stable".
func TestVerifyChecksum_MissingChecksumHardModeNonStableSkip(t *testing.T) {
	t.Setenv(pluginRequireChecksumEnv, "1")
	content := []byte("plugin archive content")
	archivePath := writeTempFile(t, content)

	for _, status := range []string{"alpha", "beta", "deprecated", "experimental"} {
		t.Run("status="+status, func(t *testing.T) {
			if err := verifyChecksum(archivePath, "", status); err != nil {
				t.Errorf("non-stable plugin with empty checksum should return nil even in hard mode, got: %v", err)
			}
		})
	}
}

// TestVerifyChecksum_ValidChecksum verifies correct checksum passes.
func TestVerifyChecksum_ValidChecksum(t *testing.T) {
	content := []byte("plugin archive content for checksum test")
	archivePath := writeTempFile(t, content)

	h := sha256.New()
	h.Write(content)
	expectedHex := hex.EncodeToString(h.Sum(nil))

	if err := verifyChecksum(archivePath, expectedHex, "stable"); err != nil {
		t.Fatalf("valid checksum should pass: %v", err)
	}
}

// TestVerifyChecksum_MismatchAlwaysRefusesRegardlessOfStatus verifies that a
// PRESENT-but-wrong checksum always refuses the install, for every
// publishStatus (including "beta"/"deprecated") — a present-but-wrong
// checksum is never acceptable regardless of lifecycle stage, and this check
// does not depend on NSELF_PLUGIN_REQUIRE_CHECKSUM.
func TestVerifyChecksum_MismatchAlwaysRefusesRegardlessOfStatus(t *testing.T) {
	content := []byte("plugin archive content for mismatch test")
	archivePath := writeTempFile(t, content)
	wrongHex := hex.EncodeToString(make([]byte, sha256.Size)) // all-zero, guaranteed wrong

	for _, status := range []string{"", "stable", "beta", "deprecated"} {
		t.Run("status="+status, func(t *testing.T) {
			err := verifyChecksum(archivePath, wrongHex, status)
			if err == nil {
				t.Fatalf("present-but-wrong checksum must refuse for status %q, got nil", status)
			}
		})
	}
}
