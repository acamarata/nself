package plugin

// identity.go — Q01 Phase B-1: per-plugin Ed25519 identity management.
//
// Each plugin gets a unique Ed25519 keypair generated at install time. The
// private key is encrypted at rest using AES-256-GCM with a per-plugin key
// derived via HKDF-SHA256 from PLUGIN_INTERNAL_SECRET and the plugin name.
// The public key is registered with ping_api after generation.
//
// Key derivation:
//   encKey = HKDF-SHA256(ikm: PLUGIN_INTERNAL_SECRET, info: pluginName, len: 32)
//
// Storage:
//   private key → PLUGIN_DATA_DIR/<name>/identity.key (AES-256-GCM, 0600)
//   registration → ping_api POST /plugin/identity/register

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// identityKeyFile is the filename for the encrypted private key inside
// the per-plugin identity directory.
const identityKeyFile = "identity.key"

// encryptedIdentity is the on-disk format for a stored plugin private key.
type encryptedIdentity struct {
	// Version allows future key-format migrations without breaking existing keys.
	Version int    `json:"v"`
	Nonce   []byte `json:"nonce"` // 12-byte AES-GCM nonce
	Data    []byte `json:"data"`  // AES-GCM ciphertext of the 64-byte Ed25519 private key
}

// GenerateEd25519Keypair generates a new Ed25519 keypair and writes the
// encrypted private key to PLUGIN_DATA_DIR/<name>/identity.key.
// It returns the public key bytes (32 bytes) for registration with ping_api.
//
// The encryption key is derived from PLUGIN_INTERNAL_SECRET and the plugin
// name using HKDF-SHA256 so each plugin has a unique encryption key.
func GenerateEd25519Keypair(pluginDataDir, pluginName string) (pubKey ed25519.PublicKey, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 keypair for %q: %w", pluginName, err)
	}

	encKey, err := deriveEncryptionKey(pluginName)
	if err != nil {
		return nil, err
	}

	ciphertext, nonce, err := aesGCMEncrypt(encKey, priv)
	if err != nil {
		return nil, fmt.Errorf("encrypting private key for %q: %w", pluginName, err)
	}

	identity := encryptedIdentity{
		Version: 1,
		Nonce:   nonce,
		Data:    ciphertext,
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return nil, fmt.Errorf("marshalling identity for %q: %w", pluginName, err)
	}

	keyDir := filepath.Join(pluginDataDir, pluginName)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating identity dir for %q: %w", pluginName, err)
	}

	keyPath := filepath.Join(keyDir, identityKeyFile)
	if err := os.WriteFile(keyPath, encoded, 0o600); err != nil {
		return nil, fmt.Errorf("writing identity key for %q: %w", pluginName, err)
	}

	return pub, nil
}

// LoadPrivateKey decrypts and returns the Ed25519 private key stored at
// PLUGIN_DATA_DIR/<name>/identity.key. Returns an error if the key does not
// exist, cannot be decrypted, or the file has been corrupted.
func LoadPrivateKey(pluginDataDir, pluginName string) (ed25519.PrivateKey, error) {
	keyPath := filepath.Join(pluginDataDir, pluginName, identityKeyFile)
	encoded, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin %q has no identity key (run 'nself plugin install %s' to generate one)", pluginName, pluginName)
		}
		return nil, fmt.Errorf("reading identity key for %q: %w", pluginName, err)
	}

	var identity encryptedIdentity
	if err := json.Unmarshal(encoded, &identity); err != nil {
		return nil, fmt.Errorf("parsing identity file for %q: %w", pluginName, err)
	}

	encKey, err := deriveEncryptionKey(pluginName)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesGCMDecrypt(encKey, identity.Nonce, identity.Data)
	if err != nil {
		return nil, fmt.Errorf("decrypting identity key for %q: %w", pluginName, err)
	}

	if len(plaintext) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("identity key for %q has unexpected length %d", pluginName, len(plaintext))
	}

	return ed25519.PrivateKey(plaintext), nil
}

// IdentityKeyExists reports whether a plugin identity key file is present on
// disk. It does NOT validate the key contents.
func IdentityKeyExists(pluginDataDir, pluginName string) bool {
	keyPath := filepath.Join(pluginDataDir, pluginName, identityKeyFile)
	_, err := os.Stat(keyPath)
	return err == nil
}

// RemoveIdentityKey deletes the stored identity key for a plugin. This is
// called during uninstall so the revocation endpoint can be called before the
// key is removed from disk.
func RemoveIdentityKey(pluginDataDir, pluginName string) error {
	keyPath := filepath.Join(pluginDataDir, pluginName, identityKeyFile)
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing identity key for %q: %w", pluginName, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Crypto helpers
// ---------------------------------------------------------------------------

// deriveEncryptionKey returns a 32-byte AES key derived from
// PLUGIN_INTERNAL_SECRET and pluginName using HKDF-SHA256.
func deriveEncryptionKey(pluginName string) ([]byte, error) {
	secret := os.Getenv("PLUGIN_INTERNAL_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("PLUGIN_INTERNAL_SECRET is not set — cannot derive plugin identity encryption key")
	}

	// HKDF-SHA256 (RFC 5869) implemented with stdlib crypto/hmac + crypto/sha256.
	// Extract: PRK = HMAC-SHA256(salt=zeroes, IKM=secret)
	// Expand:  OKM = HMAC-SHA256(PRK, info || 0x01)  [single block, len=32]
	salt := make([]byte, sha256.Size) // zero salt per RFC 5869 §2.2
	prk := hmacSHA256(salt, []byte(secret))
	// Single expand block: T(1) = HMAC-SHA256(PRK, info || 0x01)
	info := append([]byte(pluginName), 0x01)
	key := hmacSHA256(prk, info)
	return key, nil
}

// hmacSHA256 returns HMAC-SHA256(key, data).
func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// aesGCMEncrypt encrypts plaintext using AES-256-GCM with a random 12-byte nonce.
// Returns (ciphertext, nonce, error).
func aesGCMEncrypt(key, plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// aesGCMDecrypt decrypts ciphertext using AES-256-GCM.
func aesGCMDecrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}
	return plaintext, nil
}
