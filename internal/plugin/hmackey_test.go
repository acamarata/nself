package plugin

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestLoadHMACKey_MissingFile_GeneratesAndPersistsKey verifies that when the
// HMAC key file does not exist, loadHMACKey generates a 32-byte random key,
// persists it with 0600 permissions, and returns it. (V03-F02 acceptance
// criterion: "Random 32-byte key generated on first run; persisted at
// ~/.nself/license/.hmac-key with 0600 perms".)
func TestLoadHMACKey_MissingFile_GeneratesAndPersistsKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, ".hmac-key")
	t.Setenv("HMAC_KEY_PATH", keyPath)

	// File must not exist before first load.
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatal("precondition: key file should not exist")
	}

	key, err := loadHMACKey()
	if err != nil {
		t.Fatalf("loadHMACKey: %v", err)
	}

	// Key must be exactly 32 bytes.
	if len(key) != hmacKeySize {
		t.Errorf("key length = %d, want %d", len(key), hmacKeySize)
	}

	// Key must not be all zeros (crypto/rand should produce a non-zero key
	// with overwhelming probability).
	allZero := true
	for _, b := range key {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("generated key is all zeros — crypto/rand likely failed")
	}

	// File must now exist with 0600 permissions (Unix only).
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key file not found after generation: %v", err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("key file perms = %04o, want 0600", perm)
		}
	}

	// File contents must match the returned key.
	stored, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading key file: %v", err)
	}
	if !bytes.Equal(stored, key) {
		t.Error("stored key does not match returned key")
	}
}

// TestLoadHMACKey_ExistingFile_ReloadsIdentical verifies that when the key
// file already exists, loadHMACKey returns the same 32-byte value without
// regenerating. (V03-F02 acceptance criterion: "Reload uses the persisted
// key".)
func TestLoadHMACKey_ExistingFile_ReloadsIdentical(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, ".hmac-key")
	t.Setenv("HMAC_KEY_PATH", keyPath)

	// First load — generates the key.
	key1, err := loadHMACKey()
	if err != nil {
		t.Fatalf("first loadHMACKey: %v", err)
	}

	// Second load — must return the same key.
	key2, err := loadHMACKey()
	if err != nil {
		t.Fatalf("second loadHMACKey: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Errorf("second load returned different key: %x != %x", key1, key2)
	}
}

// TestLoadHMACKey_CorruptFile_Regenerates verifies that when the key file
// exists but contains the wrong number of bytes (corrupted / truncated),
// loadHMACKey regenerates a valid 32-byte key and overwrites the file.
func TestLoadHMACKey_CorruptFile_Regenerates(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, ".hmac-key")
	t.Setenv("HMAC_KEY_PATH", keyPath)

	// Write a corrupt key file (wrong length).
	if err := os.WriteFile(keyPath, []byte("tooshort"), 0600); err != nil {
		t.Fatalf("writing corrupt key file: %v", err)
	}

	key, err := loadHMACKey()
	if err != nil {
		t.Fatalf("loadHMACKey with corrupt file: %v", err)
	}
	if len(key) != hmacKeySize {
		t.Errorf("regenerated key length = %d, want %d", len(key), hmacKeySize)
	}

	// The stored file must now contain the correct 32-byte key.
	stored, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading regenerated key file: %v", err)
	}
	if !bytes.Equal(stored, key) {
		t.Error("stored regenerated key does not match returned key")
	}
}

// TestHMACCache_TamperedWithOldObservableKey_FailsVerification verifies that
// a cache entry HMAC-signed with the old observable-derived machineID string
// (SHA-256(hostname:homeDir:GOOS:GOARCH)[0:16]) fails verification when the
// new random-key scheme is in use. This is the regression test for SIEGE
// V03-F02: "tampered cache (forged with old observable-derived key) FAILS HMAC
// verification under new random-key scheme".
func TestHMACCache_TamperedWithOldObservableKey_FailsVerification(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, ".hmac-key")
	t.Setenv("HMAC_KEY_PATH", keyPath)

	// Load (generate) the random key.
	randomKey, err := loadHMACKey()
	if err != nil {
		t.Fatalf("loadHMACKey: %v", err)
	}

	// Compute the old observable-derived key as an attacker would.
	oldObservableKey := []byte(machineID()) // SHA-256(hostname:home:GOOS:GOARCH)[0:16]

	// Sanity: old key must differ from new random key.
	if bytes.Equal(oldObservableKey, randomKey) {
		t.Fatal("old observable key equals random key — test is invalid")
	}

	// Sign a cache data string with the old observable-derived key.
	data := "nself_pro_|valid|1700000000"
	forgeddSig := hmacSign(data, oldObservableKey)

	// Verify against the new random key — must fail.
	if hmacVerify(data, forgeddSig, randomKey) {
		t.Error("SIEGE V03-F02: hmacVerify returned true for a signature " +
			"forged with the observable-derived machineID key — " +
			"random-key scheme does not reject old forged entries")
	}

	// Sanity: same data signed with the correct random key must pass.
	correctSig := hmacSign(data, randomKey)
	if !hmacVerify(data, correctSig, randomKey) {
		t.Error("hmacVerify returned false for a correctly signed entry — regression in happy path")
	}
}
