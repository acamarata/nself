package plugin

// license_cache.go — on-disk license validation cache.
//
// Purpose: cache remote license validation results so plugins keep working offline within the grace period, and decide when a cached entry needs revalidation, used by ValidateLicenseRemote in license.go, split out for file size.
// Inputs: a plugin slug/license key and the cache directory.
// Outputs: cached validation results read from or written to disk, and a revalidation decision.
// Constraints: pure move from license.go (CLI-R12 Batch F); no behaviour change. Does not touch the manager.go plugin load order.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cacheMaxAge is the maximum age of a cache entry before it is pruned.
// Entries older than 30 days are removed on the next CacheLicense call.
const cacheMaxAge = 30 * 24 * time.Hour

// pruneExpiredCacheEntries reads the cache file line by line, drops any entry
// whose timestamp is older than maxAge, and rewrites the file with the
// remaining valid entries. If the file does not exist the function is a no-op.
func pruneExpiredCacheEntries(cachePath string, maxAge time.Duration) error {
	f, err := os.Open(cachePath)
	if err != nil {
		// File missing or unreadable -- nothing to prune.
		return nil
	}
	defer func() { _ = f.Close() }()

	hmacKey := loadHMACKeyOrFallback()
	now := time.Now()
	var kept []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Split on the last "|" to separate data from HMAC.
		lastPipe := strings.LastIndex(line, "|")
		if lastPipe < 0 {
			continue
		}
		data := line[:lastPipe]
		sig := line[lastPipe+1:]

		// Drop tampered entries silently during pruning.
		if !hmacVerify(data, sig, hmacKey) {
			continue
		}

		parts := strings.SplitN(data, "|", 3)
		if len(parts) != 3 {
			continue
		}

		var ts int64
		if _, err := fmt.Sscanf(parts[2], "%d", &ts); err != nil {
			continue
		}
		if now.Sub(time.Unix(ts, 0)) > maxAge {
			continue
		}

		kept = append(kept, line)
	}

	_ = f.Close()

	// Rewrite file with only valid, non-expired entries.
	content := strings.Join(kept, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(cachePath, []byte(content), 0600)
}

// CacheLicense writes the validation result for a license key to a cache file
// inside cacheDir. The cache format is: {data}|{hmac_hex}
// where data is {key_prefix}|{status}|{timestamp} and hmac_hex is the
// HMAC-SHA256 of data keyed by the random 32-byte key persisted at
// ~/.nself/license/.hmac-key (SIEGE V03-F02 fix — replaces observable-derived
// machineID key).
//
// Before writing, it prunes any cache entries older than 30 days.
func CacheLicense(key string, valid bool, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("creating license cache directory: %w", err)
	}

	cachePath := filepath.Join(cacheDir, "cache")

	// Prune stale entries before writing the new one.
	if err := pruneExpiredCacheEntries(cachePath, cacheMaxAge); err != nil {
		return fmt.Errorf("pruning expired cache entries: %w", err)
	}

	prefix := keyPrefix(key)
	status := "invalid"
	if valid {
		status = "valid"
	}
	data := fmt.Sprintf("%s|%s|%d", prefix, status, time.Now().Unix())
	sig := hmacSign(data, loadHMACKeyOrFallback())
	line := data + "|" + sig + "\n"

	return os.WriteFile(cachePath, []byte(line), 0600)
}

// CheckLicenseCache reads the license cache and returns whether the key is
// valid and whether a non-expired cache entry was found. The HMAC signature
// on each entry is verified before trusting the cached result. If the
// signature is missing or invalid the cache file is deleted and the caller
// must re-validate against the server.
func checkLicenseCache(key string, cacheDir string) (valid bool, found bool) {
	cachePath := filepath.Join(cacheDir, "cache")
	f, err := os.Open(cachePath)
	if err != nil {
		return false, false
	}
	defer func() { _ = f.Close() }()

	hmacKey := loadHMACKeyOrFallback()
	prefix := keyPrefix(key)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Split on the last "|" to separate data from HMAC.
		lastPipe := strings.LastIndex(line, "|")
		if lastPipe < 0 {
			continue
		}
		data := line[:lastPipe]
		sig := line[lastPipe+1:]

		// Verify HMAC before trusting anything in the entry.
		if !hmacVerify(data, sig, hmacKey) {
			// Tampered or unsigned entry — nuke cache file.
			_ = f.Close()
			_ = os.Remove(cachePath)
			return false, false
		}

		parts := strings.SplitN(data, "|", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[0] != prefix {
			continue
		}

		var ts int64
		if _, err := fmt.Sscanf(parts[2], "%d", &ts); err != nil {
			continue
		}
		cached := time.Unix(ts, 0)
		// Owner keys never expire from cache.
		if !isOwnerKey(key) && time.Since(cached) > cacheTTL {
			return false, false
		}
		return parts[1] == "valid", true
	}
	return false, false
}

// CheckLicenseCacheOffline reads the license cache using the extended offline
// grace period (offlineGraceTTL). It only returns true for entries that were
// previously validated as "valid". This is used as a fallback when the network
// is unavailable, allowing offline installs for up to 7 days after the last
// successful remote validation. HMAC signatures are verified identically to
// CheckLicenseCache.
func checkLicenseCacheOffline(key string, cacheDir string) (valid bool, found bool) {
	cachePath := filepath.Join(cacheDir, "cache")
	f, err := os.Open(cachePath)
	if err != nil {
		return false, false
	}
	defer func() { _ = f.Close() }()

	hmacKey := loadHMACKeyOrFallback()
	prefix := keyPrefix(key)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Split on the last "|" to separate data from HMAC.
		lastPipe := strings.LastIndex(line, "|")
		if lastPipe < 0 {
			continue
		}
		data := line[:lastPipe]
		sig := line[lastPipe+1:]

		// Verify HMAC before trusting anything in the entry.
		if !hmacVerify(data, sig, hmacKey) {
			_ = f.Close()
			_ = os.Remove(cachePath)
			return false, false
		}

		parts := strings.SplitN(data, "|", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[0] != prefix {
			continue
		}

		var ts int64
		if _, err := fmt.Sscanf(parts[2], "%d", &ts); err != nil {
			continue
		}
		cached := time.Unix(ts, 0)
		// Owner keys never expire from cache (online or offline).
		if !isOwnerKey(key) && time.Since(cached) > offlineGraceTTL {
			return false, false
		}
		// Only trust "valid" entries for offline grace. If the last
		// cached result was "invalid", do not allow offline install.
		return parts[1] == "valid", true
	}
	return false, false
}

// revalidationInterval is the maximum time between periodic license
// revalidation checks during nself start. If the cache is older than this,
// the startup sequence attempts a soft revalidation against the server.
const revalidationInterval = 7 * 24 * time.Hour

// NeedsRevalidation returns true if the license cache in cacheDir is missing,
// tampered with, or if the most recent validation timestamp is older than
// revalidationInterval (7 days). This is used during nself start to trigger a
// periodic heartbeat check against the license server. Owner keys never need
// revalidation.
func NeedsRevalidation(key string, cacheDir string) bool {
	if isOwnerKey(key) {
		return false
	}
	cachePath := filepath.Join(cacheDir, "cache")
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		return true
	}

	line := strings.TrimSpace(string(raw))
	if line == "" {
		return true
	}

	// Cache format: {prefix}|{status}|{timestamp}|{hmac_hex}
	// Split on the last "|" to separate data from HMAC.
	lastPipe := strings.LastIndex(line, "|")
	if lastPipe < 0 {
		return true
	}
	data := line[:lastPipe]
	sig := line[lastPipe+1:]

	// Verify HMAC before trusting the timestamp.
	if !hmacVerify(data, sig, loadHMACKeyOrFallback()) {
		return true
	}

	// Extract timestamp from data ({prefix}|{status}|{timestamp}).
	parts := strings.SplitN(data, "|", 3)
	if len(parts) != 3 {
		return true
	}

	var ts int64
	if _, err := fmt.Sscanf(parts[2], "%d", &ts); err != nil {
		return true
	}

	return time.Since(time.Unix(ts, 0)) > revalidationInterval
}
