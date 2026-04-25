// Package license — cache.go implements the local license cache with Ed25519
// signature verification and grace-period-aware freshness checks.
//
// Cache location: ~/.cache/nself/license.json (0600)
// Fields: key_hash, tier, plugins_allowed, fetched_at, expires_at, signature
package license

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// licensePubKeyHex is injected via -X ldflag at goreleaser build time.
// In dev builds without ldflags, it remains empty — license verification is
// disabled and nself version prints a warning banner.
//
//nolint:gochecknoglobals
var licensePubKeyHex = "" //nolint:unused // set via -X github.com/nself-org/cli/internal/license.licensePubKeyHex=<hex>

// IsZeroPubKey reports whether the build was made without an ldflags-injected
// signing key. Returns true when licensePubKeyHex is empty OR consists entirely
// of '0' characters (e.g., a placeholder 64-char zero string).
// goreleaser injects a real non-zero Ed25519 pubkey hex; dev builds leave it empty.
func IsZeroPubKey() bool {
	if licensePubKeyHex == "" {
		return true
	}
	for _, ch := range licensePubKeyHex {
		if ch != '0' {
			return false
		}
	}
	return true
}

// CacheEntry represents a cached license validation response with Ed25519
// signature from the server.
type CacheEntry struct {
	KeyHash        string   `json:"key_hash"`
	Tier           string   `json:"tier"`
	PluginsAllowed []string `json:"plugins_allowed"`
	FetchedAt      int64    `json:"fetched_at"`
	ExpiresAt      int64    `json:"expires_at"`
	Signature      string   `json:"signature"`
	SignatureKeyID int      `json:"signature_key_id"`
}

// defaultCacheDir returns ~/.cache/nself.
func defaultCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "nself"), nil
}

// CachePath returns the full path to the license cache file.
// It respects LICENSE_CACHE_PATH if set.
func CachePath() (string, error) {
	if p := os.Getenv("LICENSE_CACHE_PATH"); p != "" {
		return p, nil
	}
	dir, err := defaultCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "license.json"), nil
}

// ReadCache reads and parses the license cache file.
// Returns nil, nil if the cache file does not exist.
func ReadCache() (*CacheEntry, error) {
	path, err := CachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading license cache: %w", err)
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing license cache: %w", err)
	}
	return &entry, nil
}

// WriteCache writes the cache entry to disk with 0600 permissions.
func WriteCache(entry *CacheEntry) error {
	path, err := CachePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling cache entry: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// DeleteCache removes the license cache file.
func DeleteCache() error {
	path, err := CachePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing license cache: %w", err)
	}
	return nil
}

// HashKey returns the SHA-256 hex digest of a license key.
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// CacheAge returns how long ago the cache was fetched.
func (c *CacheEntry) CacheAge() time.Duration {
	return time.Since(time.Unix(c.FetchedAt, 0))
}

// VerifySignature verifies the cache entry's Ed25519 signature against the
// bundled public keys. It accepts the current key (keyID N) and the previous
// key (keyID N-1) to support rotation windows.
func (c *CacheEntry) VerifySignature() bool {
	keys := GetPublicKeys()
	for _, pk := range keys {
		if c.SignatureKeyID != 0 && pk.ID != c.SignatureKeyID {
			continue
		}
		sigBytes, err := hex.DecodeString(c.Signature)
		if err != nil {
			continue
		}
		// The signed payload is the JSON of the entry without the signature fields.
		payload := c.signablePayload()
		if ed25519.Verify(pk.Key, payload, sigBytes) {
			return true
		}
	}
	// If keyID was specified and didn't match, try all keys (rotation window).
	if c.SignatureKeyID != 0 {
		sigBytes, err := hex.DecodeString(c.Signature)
		if err != nil {
			return false
		}
		payload := c.signablePayload()
		for _, pk := range keys {
			if ed25519.Verify(pk.Key, payload, sigBytes) {
				return true
			}
		}
	}
	return false
}

// signablePayload produces the deterministic byte sequence that was signed by
// the server. This must match the server's signing format exactly.
func (c *CacheEntry) signablePayload() []byte {
	// Canonical format: key_hash|tier|fetched_at|expires_at
	return []byte(fmt.Sprintf("%s|%s|%d|%d", c.KeyHash, c.Tier, c.FetchedAt, c.ExpiresAt))
}

// PublicKeyEntry holds a versioned Ed25519 public key.
type PublicKeyEntry struct {
	ID  int
	Key ed25519.PublicKey
}

// bundledPublicKeys contains the Ed25519 public keys used to verify license
// cache signatures. Key ID 1 is the current key. During rotation, both N and
// N-1 are accepted.
//
// The public key override env var LICENSE_PUBLIC_KEY_OVERRIDE can replace key 1
// for testing.
var bundledPublicKeys []PublicKeyEntry

func init() {
	// Default placeholder key (replaced at build time or by server).
	// This is a valid Ed25519 public key used for development/testing.
	// Production builds embed the real public key via ldflags.
	devKey := make(ed25519.PublicKey, ed25519.PublicKeySize)
	bundledPublicKeys = []PublicKeyEntry{
		{ID: 1, Key: devKey},
	}
}

// GetPublicKeys returns the active public keys, respecting the
// LICENSE_PUBLIC_KEY_OVERRIDE environment variable for testing.
func GetPublicKeys() []PublicKeyEntry {
	if override := os.Getenv("LICENSE_PUBLIC_KEY_OVERRIDE"); override != "" {
		keyBytes, err := hex.DecodeString(override)
		if err == nil && len(keyBytes) == ed25519.PublicKeySize {
			return []PublicKeyEntry{{ID: 1, Key: ed25519.PublicKey(keyBytes)}}
		}
	}
	return bundledPublicKeys
}
