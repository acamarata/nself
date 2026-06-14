package plugin

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestGenerateAndLoadKeypair verifies the full encrypt-store-decrypt round-trip.
func TestGenerateAndLoadKeypair(t *testing.T) {
	t.Setenv("PLUGIN_INTERNAL_SECRET", "test-secret-for-plugin-identity-unit-tests")

	dir := t.TempDir()
	pluginName := "ai"

	pubKey, err := GenerateEd25519Keypair(dir, pluginName)
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair: %v", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		t.Fatalf("expected %d-byte public key, got %d", ed25519.PublicKeySize, len(pubKey))
	}

	// Key file must exist with 0600 permissions (Unix only).
	keyPath := filepath.Join(dir, pluginName, identityKeyFile)
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key file not found: %v", err)
	}
	if runtime.GOOS != "windows" {
		if info.Mode() != 0o600 {
			t.Errorf("expected mode 0600, got %o", info.Mode())
		}
	}

	// Load must recover the same keypair.
	privKey, err := LoadPrivateKey(dir, pluginName)
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	// Public key derived from the loaded private key must match what was returned at generation.
	recoveredPub := privKey.Public().(ed25519.PublicKey)
	if !recoveredPub.Equal(pubKey) {
		t.Errorf("recovered public key does not match generated public key")
	}
}

// TestEncryptDecryptRoundTrip tests the AES-GCM helpers in isolation.
func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("hello from the plugin identity test")

	ct, nonce, err := aesGCMEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if len(nonce) != 12 {
		t.Fatalf("expected 12-byte nonce, got %d", len(nonce))
	}

	recovered, err := aesGCMDecrypt(key, nonce, ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(recovered) != string(plaintext) {
		t.Errorf("decrypted %q != original %q", recovered, plaintext)
	}
}

// TestDeriveEncryptionKeyIsDeterministic verifies that the same secret+name
// always produces the same derived key.
func TestDeriveEncryptionKeyIsDeterministic(t *testing.T) {
	t.Setenv("PLUGIN_INTERNAL_SECRET", "stable-secret")

	k1, err := deriveEncryptionKey("mux")
	if err != nil {
		t.Fatalf("first derivation: %v", err)
	}
	k2, err := deriveEncryptionKey("mux")
	if err != nil {
		t.Fatalf("second derivation: %v", err)
	}
	if string(k1) != string(k2) {
		t.Error("derived key is not deterministic")
	}
}

// TestDeriveEncryptionKeyDiffersPerPlugin verifies that different plugin names
// produce different encryption keys (blast-radius isolation).
func TestDeriveEncryptionKeyDiffersPerPlugin(t *testing.T) {
	t.Setenv("PLUGIN_INTERNAL_SECRET", "stable-secret")

	kAI, _ := deriveEncryptionKey("ai")
	kMux, _ := deriveEncryptionKey("mux")

	if string(kAI) == string(kMux) {
		t.Error("ai and mux should have different encryption keys")
	}
}

// TestIdentityKeyExists verifies the existence check helper.
func TestIdentityKeyExists(t *testing.T) {
	t.Setenv("PLUGIN_INTERNAL_SECRET", "test-secret")

	dir := t.TempDir()
	pluginName := "claw"

	if IdentityKeyExists(dir, pluginName) {
		t.Error("key should not exist before generation")
	}

	if _, err := GenerateEd25519Keypair(dir, pluginName); err != nil {
		t.Fatalf("GenerateEd25519Keypair: %v", err)
	}

	if !IdentityKeyExists(dir, pluginName) {
		t.Error("key should exist after generation")
	}
}

// TestRemoveIdentityKey verifies key deletion.
func TestRemoveIdentityKey(t *testing.T) {
	t.Setenv("PLUGIN_INTERNAL_SECRET", "test-secret")

	dir := t.TempDir()
	pluginName := "notify"

	if _, err := GenerateEd25519Keypair(dir, pluginName); err != nil {
		t.Fatalf("GenerateEd25519Keypair: %v", err)
	}

	if err := RemoveIdentityKey(dir, pluginName); err != nil {
		t.Fatalf("RemoveIdentityKey: %v", err)
	}

	if IdentityKeyExists(dir, pluginName) {
		t.Error("key should not exist after removal")
	}

	// Second removal should be a no-op.
	if err := RemoveIdentityKey(dir, pluginName); err != nil {
		t.Errorf("second removal should be no-op, got: %v", err)
	}
}

// TestMissingSecretReturnsError verifies that missing PLUGIN_INTERNAL_SECRET
// is caught early.
func TestMissingSecretReturnsError(t *testing.T) {
	os.Unsetenv("PLUGIN_INTERNAL_SECRET")

	dir := t.TempDir()
	if _, err := GenerateEd25519Keypair(dir, "ai"); err == nil {
		t.Error("expected error when PLUGIN_INTERNAL_SECRET is unset")
	}
}
