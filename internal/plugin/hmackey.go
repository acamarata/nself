package plugin

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// hmacKeyFile is the filename for the persisted random HMAC key, stored under
// the same directory as the license key (~/.nself/license/).
const hmacKeyFile = ".hmac-key"

// hmacKeySize is the number of random bytes used for the HMAC key.
const hmacKeySize = 32

// hmacKeyPath returns the absolute path to ~/.nself/license/.hmac-key.
// The HMAC_KEY_PATH environment variable overrides the path for testing.
func hmacKeyPath() (string, error) {
	if p := os.Getenv("HMAC_KEY_PATH"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".nself", "license", hmacKeyFile), nil
}

// loadHMACKey returns the persisted 32-byte random HMAC key, generating and
// persisting it on first run. The key is stored at ~/.nself/license/.hmac-key
// with mode 0600.
//
// If the key file is missing (e.g. after a clean install or manual deletion),
// a fresh key is generated and written. Any existing secondary license cache
// signed with the old key will fail HMAC verification on next read, causing
// a re-fetch from the license server. This is the intended behavior: missing
// key file → forced re-validation (fail-open, 7-day grace still applies via
// the primary Ed25519-signed cache in internal/license/cache.go).
func loadHMACKey() ([]byte, error) {
	path, err := hmacKeyPath()
	if err != nil {
		return nil, err
	}

	// Attempt to load existing key.
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) == hmacKeySize {
			return data, nil
		}
		// File exists but has wrong length (truncated / corrupted) — regenerate.
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading HMAC key file: %w", err)
	}

	// Generate a fresh 32-byte random key.
	key := make([]byte, hmacKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating HMAC key: %w", err)
	}

	// Persist atomically with 0600 permissions.
	if err := writeHMACKey(path, key); err != nil {
		return nil, err
	}
	return key, nil
}

// writeHMACKey writes key to path with 0600 permissions using a sibling
// tmpfile + rename pattern to avoid torn writes.
func writeHMACKey(path string, key []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating HMAC key directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".hmac-key.tmp.*")
	if err != nil {
		return fmt.Errorf("creating temp HMAC key file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod HMAC key temp file: %w", err)
	}
	if _, err := tmp.Write(key); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("writing HMAC key temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("syncing HMAC key temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("closing HMAC key temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("persisting HMAC key file: %w", err)
	}
	return nil
}
