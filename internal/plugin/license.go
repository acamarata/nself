package plugin

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/errs"
)

// validLicensePrefixes lists the accepted license key prefixes.
var validLicensePrefixes = []string{
	"nself_pro_",
	"nself_max_",
	"nself_ent_",
	"nself_owner_",
}

// paidPlugins is the hardcoded set of pro plugin names that require a license.
var paidPlugins = map[string]bool{
	// Auth & Access
	"access-controls": true,
	"auth":            true,
	"idme":            true,
	"compliance":      true,
	"entitlements":    true,
	// Content
	"activity-feed":   true,
	"cms":             true,
	"documents":       true,
	"knowledge-base":  true,
	"moderation":      true,
	"photos":          true,
	"social":          true,
	"support":         true,
	"calendar":        true,
	// AI & Automation
	"ai":        true,
	"claw":      true,
	"claw-web":  true,
	"cron":      true,
	"mux":       true,
	"workflows": true,
	"bots":      true,
	// Communication
	"chat":             true,
	"livekit":          true,
	"notify":           true,
	"streaming":        true,
	"voice":            true,
	"podcast":          true,
	"media-processing": true,
	"epg":              true,
	"game-metadata":    true,
	"retro-gaming":     true,
	"rom-discovery":    true,
	"tmdb":             true,
	// Commerce
	"stripe":         true,
	"paypal":         true,
	"shopify":        true,
	"donorbox":       true,
	"analytics":      true,
	"admin-api":      true,
	"recording":      true,
	"stream-gateway": true,
	// Infrastructure
	"object-storage":  true,
	"cdn":             true,
	"cloudflare":      true,
	"backup":          true,
	"realtime":        true,
	"file-processing": true,
	"browser":         true,
	"observability":   true,
	"google":          true,
	"web3":            true,
	"ddns":            true,
	"geocoding":       true,
	"geolocation":     true,
	// Other
	"devices":  true,
	"meetings": true,
	"sports":   true,
	"home":     true,
	"post":     true,
}

// cacheTTL is the duration a cached license result remains valid during
// normal operation (online mode).
const cacheTTL = 24 * time.Hour

// offlineGraceTTL is the maximum age of a cached "valid" entry that can be
// trusted when the network is unavailable. This gives users a 7-day window
// to work offline without re-validating against the server.
const offlineGraceTTL = 7 * 24 * time.Hour

// IsPaidPlugin returns true if the named plugin requires a license key.
func isPaidPlugin(name string) bool {
	return paidPlugins[name]
}

// ValidateLicenseFormat checks that a license key has a recognized prefix and
// meets the minimum length requirement. Returns errs.ErrInvalidLicenseKey on
// failure.
func validateLicenseFormat(key string) error {
	if len(key) < 32 {
		return errs.ErrInvalidLicenseKey
	}
	for _, prefix := range validLicensePrefixes {
		if strings.HasPrefix(key, prefix) {
			return nil
		}
	}
	return errs.ErrInvalidLicenseKey
}

// MachineID returns a deterministic 16-character hex fingerprint derived from
// the hostname, home directory, OS, and architecture of the current machine.
func machineID() string {
	hostname, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	raw := hostname + ":" + home + ":" + runtime.GOOS + ":" + runtime.GOARCH
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])[:16]
}

// licenseRequest is the JSON body sent to the license validation endpoint.
type licenseRequest struct {
	LicenseKey string `json:"license_key"`
	Product    string `json:"product"`
	MachineID  string `json:"machine_id"`
}

// ValidateLicenseRemote performs a remote license check by POSTing to the
// given pingURL. It returns (true, nil) on HTTP 200, (false, nil) on
// 401/403/404, and (false, error) on network or unexpected failures.
func ValidateLicenseRemote(ctx context.Context, key string, pingURL string) (bool, error) {
	valid, _, err := validateLicenseRemoteWithEntitlements(ctx, key, pingURL)
	return valid, err
}

// ValidateLicenseRemoteWithEntitlements is like ValidateLicenseRemote but also
// returns the parsed response body on HTTP 200. The response includes the tier
// name and list of plugins the license is entitled to, which can be cached
// locally to avoid repeated remote calls.
func validateLicenseRemoteWithEntitlements(ctx context.Context, key string, pingURL string) (bool, *licenseValidateResponse, error) {
	// Owner keys skip machine fingerprint — they are not bound to a
	// specific device.
	mid := ""
	if !isOwnerKey(key) {
		mid = machineID()
	}

	body, err := json.Marshal(licenseRequest{
		LicenseKey: key,
		Product:    "plugins-pro",
		MachineID:  mid,
	})
	if err != nil {
		return false, nil, fmt.Errorf("marshalling license request: %w", err)
	}

	url := strings.TrimRight(pingURL, "/") + "/license/validate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false, nil, fmt.Errorf("creating license request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("license validation request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var lvr licenseValidateResponse
		if decErr := json.NewDecoder(resp.Body).Decode(&lvr); decErr != nil {
			// Unparsable body — treat as invalid to be safe.
			return false, nil, fmt.Errorf("license validation response unreadable: %w", decErr)
		}
		if !lvr.Valid {
			if lvr.Reason != "" {
				return false, nil, fmt.Errorf("license invalid: %s", lvr.Reason)
			}
			return false, nil, fmt.Errorf("license invalid")
		}
		return true, &lvr, nil
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false, nil, nil
	default:
		return false, nil, fmt.Errorf("unexpected license validation status: %d", resp.StatusCode)
	}
}

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
	defer f.Close()

	machineKey := machineID()
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
		if !hmacVerify(data, sig, machineKey) {
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

	f.Close()

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
// HMAC-SHA256 of data keyed by machineID().
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
	sig := hmacSign(data, machineID())
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
	defer f.Close()

	machineKey := machineID()
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
		if !hmacVerify(data, sig, machineKey) {
			// Tampered or unsigned entry — nuke cache file.
			f.Close()
			os.Remove(cachePath)
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
	defer f.Close()

	machineKey := machineID()
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
		if !hmacVerify(data, sig, machineKey) {
			f.Close()
			os.Remove(cachePath)
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
	if !hmacVerify(data, sig, machineID()) {
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

// hmacSign returns the HMAC-SHA256 of data using key, encoded as a hex string.
func hmacSign(data string, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// hmacVerify checks that expectedHMAC matches the HMAC-SHA256 of data using
// key. The comparison is timing-safe.
func hmacVerify(data string, expectedHMAC string, key string) bool {
	expected, err := hex.DecodeString(expectedHMAC)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hmac.Equal(mac.Sum(nil), expected)
}

// isOwnerKey returns true if the license key has the nself_owner_ prefix.
// Owner keys receive special treatment: no machine fingerprint and no cache
// expiry.
func isOwnerKey(key string) bool {
	return strings.HasPrefix(key, "nself_owner_")
}

// entitlementCache is the JSON structure persisted to entitlements.json.
type entitlementCache struct {
	Tier     string   `json:"tier"`
	Plugins  []string `json:"plugins"`
	CachedAt string   `json:"cached_at"`
}

// entitlementCacheTTL is how long a cached entitlements file is considered
// fresh. After this duration, CheckEntitlements reports a cache miss so the
// caller falls through to remote validation.
const entitlementCacheTTL = 24 * time.Hour

// CacheEntitlements writes the tier-to-plugins mapping to
// {cacheDir}/entitlements.json with 0600 permissions.
func cacheEntitlements(cacheDir string, tier string, plugins []string) error {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("creating entitlement cache directory: %w", err)
	}
	c := entitlementCache{
		Tier:     tier,
		Plugins:  plugins,
		CachedAt: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling entitlement cache: %w", err)
	}
	return os.WriteFile(filepath.Join(cacheDir, "entitlements.json"), data, 0600)
}

// CheckEntitlements reads {cacheDir}/entitlements.json and checks whether
// pluginName is in the cached plugins list. It returns two booleans:
//
//   - allowed: true if pluginName appears in the cached list.
//   - found:   true if a valid, non-expired cache file was read.
//
// If the file is missing, unparsable, or older than 24 hours, found is false
// and the caller should fall through to remote validation.
func checkEntitlements(cacheDir string, pluginName string) (allowed bool, found bool) {
	data, err := os.ReadFile(filepath.Join(cacheDir, "entitlements.json"))
	if err != nil {
		return false, false
	}
	var c entitlementCache
	if err := json.Unmarshal(data, &c); err != nil {
		return false, false
	}
	cachedAt, err := time.Parse(time.RFC3339, c.CachedAt)
	if err != nil {
		return false, false
	}
	if time.Since(cachedAt) > entitlementCacheTTL {
		return false, false
	}
	for _, p := range c.Plugins {
		if p == pluginName {
			return true, true
		}
	}
	return false, true
}

// licenseValidateResponse is the JSON body returned by the license validation
// endpoint on HTTP 200. The Tier and Plugins fields are used to populate the
// local entitlement cache.
type licenseValidateResponse struct {
	Valid   bool     `json:"valid"`
	Reason  string   `json:"reason,omitempty"`
	Tier    string   `json:"tier"`
	Plugins []string `json:"plugins"`
}

// keyPrefix extracts the prefix portion of a license key (everything up to and
// including the third underscore-delimited segment). For example,
// "nself_pro_abc123..." returns "nself_pro_".
func keyPrefix(key string) string {
	for _, prefix := range validLicensePrefixes {
		if strings.HasPrefix(key, prefix) {
			return prefix
		}
	}
	// Fallback: first 10 chars or the whole key if shorter.
	if len(key) > 10 {
		return key[:10]
	}
	return key
}
