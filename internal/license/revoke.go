// Package license — revoke.go implements local license revocation and plugin
// dormancy (S133-T03 / T04).
//
// Revoking a license:
//   - Marks the local cache with revoked=true and wiped_at timestamp.
//   - Removes the stored key from disk.
//   - Plugins transition to DORMANT on next build (not uninstalled).
//
// Restoring a license (S133-T04):
//   - Validates a new key before writing.
//   - Clears the revoked marker from the cache.
//   - Signals dormant plugins to re-activate on next build.
package license

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// revokedMarkerFile is stored alongside license.json to signal dormancy to the
// plugin loader.
const revokedMarkerFile = "license.revoked"

// RevokedMarker is written to disk when a license is revoked.
type RevokedMarker struct {
	// WipedAt is the Unix timestamp when the license was revoked.
	WipedAt int64 `json:"wiped_at"`
	// Reason is an optional human-readable reason.
	Reason string `json:"reason,omitempty"`
}

// revokedMarkerPath returns the path to the revoked marker file.
func revokedMarkerPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "nself", revokedMarkerFile), nil
}

// IsRevoked returns true if a revoked marker file is present on disk.
func IsRevoked() bool {
	path, err := revokedMarkerPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// RevokeLicense marks the license as revoked:
//  1. Writes a revoked marker to disk.
//  2. Deletes the local license cache.
//  3. Clears all stored license keys.
//
// Plugins go DORMANT (not uninstalled) on the next `nself build`.
func RevokeLicense() error {
	// Write the revoked marker.
	markerPath, err := revokedMarkerPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(markerPath)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	marker := RevokedMarker{WipedAt: time.Now().Unix()}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling revoked marker: %w", err)
	}
	if err := os.WriteFile(markerPath, data, 0600); err != nil {
		return fmt.Errorf("writing revoked marker: %w", err)
	}

	// Delete the license cache.
	if err := DeleteCache(); err != nil {
		return fmt.Errorf("clearing license cache: %w", err)
	}

	// Clear all stored keys.
	if err := ClearKey(); err != nil {
		return fmt.Errorf("clearing license keys: %w", err)
	}

	return nil
}

// ClearRevokedMarker removes the revoked marker file (called on successful
// restore).
func ClearRevokedMarker() error {
	path, err := revokedMarkerPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing revoked marker: %w", err)
	}
	return nil
}

// RestoreWithKey validates the new key, clears the revoked marker, and stores
// the new key so dormant plugins re-activate on next build.
//
// The key must pass format validation before it is written. Remote validation
// is handled by the caller (license restore command) so that we can surface a
// clear error to the user before touching disk.
func RestoreWithKey(newKey string) error {
	if err := ValidateKeyFormat(newKey); err != nil {
		return fmt.Errorf("invalid license key: %w", err)
	}

	// Clear the revoked marker.
	if err := ClearRevokedMarker(); err != nil {
		return fmt.Errorf("clearing revoked marker: %w", err)
	}

	// Store the new key.
	return SetKey(newKey)
}
