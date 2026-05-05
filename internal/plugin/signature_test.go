package plugin

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"testing"
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
	defer f.Close()
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

	if err := verifyPluginSignature(archivePath, pubKeyHex, sigHex); err != nil {
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

	err := verifyPluginSignature(archivePath, pubKeyHex, sigHex)
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

	err := verifyPluginSignature(archivePath, pubKeyHexA, sigHex) // verify with A
	if err == nil {
		t.Fatal("wrong public key should fail verification, got nil error")
	}
}

// TestVerifyPluginSignature_EmptyInputsSkip verifies that empty key or signature
// causes the function to return nil (skip verification, not error).
func TestVerifyPluginSignature_EmptyInputsSkip(t *testing.T) {
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
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := verifyPluginSignature(archivePath, tc.pubKey, tc.sigHex); err != nil {
				t.Errorf("empty inputs should skip verification (return nil), got: %v", err)
			}
		})
	}
}

// TestVerifyPluginSignature_MalformedKey verifies that a malformed hex public key
// returns a descriptive error.
func TestVerifyPluginSignature_MalformedKey(t *testing.T) {
	content := []byte("plugin content")
	archivePath := writeTempFile(t, content)

	err := verifyPluginSignature(archivePath, "not-valid-hex!!!", "aabbcc")
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

	err := verifyPluginSignature(archivePath, shortKey, sigHex)
	if err == nil {
		t.Fatal("wrong key length should return error, got nil")
	}
}

// TestVerifyPluginSignature_MissingArchive verifies that a missing archive file
// returns an error.
func TestVerifyPluginSignature_MissingArchive(t *testing.T) {
	pubKeyHex, _, _, priv := generateTestKeyPair(t)
	sigHex := signFileContent(t, []byte("data"), priv)

	err := verifyPluginSignature("/nonexistent/path/archive.tar.gz", pubKeyHex, sigHex)
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

	if err := verifyPluginSignature(archivePath, pubKeyHex, sigHex); err != nil {
		t.Fatalf("large file signature verification failed: %v", err)
	}
}
