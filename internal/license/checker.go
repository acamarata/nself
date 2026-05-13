// Package license — checker.go: bundle-level entitlement check.
//
// BundleEntitled calls ping_api /license/validate?bundle=<name> with the
// operator's license key. This is a DISTINCT call from per-plugin validation:
// a user who has individual plugin entitlements does NOT automatically get
// bundle-level access (bundles are a separate SKU on the ping_api side).
//
// S2.T03 CR-C security requirement: bundle validation MUST use the bundle
// query parameter, NOT individual plugin checks. An operator with a la-carte
// plugin licenses must NOT gain bundle install access for free.
package license

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// bundleValidateResponse is the JSON body from /license/validate?bundle=<name>.
type bundleValidateResponse struct {
	Valid   bool   `json:"valid"`
	Tier    string `json:"tier"`
	Reason  string `json:"reason,omitempty"`
	Bundle  string `json:"bundle,omitempty"`
	Plugins []string `json:"plugins,omitempty"`
}

// BundleEntitled reports whether the operator's license key grants access to
// the named bundle. It calls ping_api /license/validate?bundle=<bundleName>
// with the key. Returns false on any error (network, auth, invalid key) so
// the installer can surface a precise message rather than silently proceeding.
//
// FAIL-OPEN exception: when NSELF_LICENSE_FAIL_OPEN=1 is set (CI / air-gap
// environments) and the network is unreachable, falls back to the local cache
// to check tier coverage. Production deployments MUST NOT set this var.
func BundleEntitled(ctx context.Context, key, bundleName string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("no license key configured; run 'nself license set <key>' or visit nself.org/pricing")
	}

	pingBase := PingURL()
	url := strings.TrimRight(pingBase, "/") + "/license/validate?bundle=" + bundleName

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(""))
	if err != nil {
		return false, fmt.Errorf("building bundle validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NSelf-License-Key", key)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Network unreachable — attempt FAIL-OPEN via cache if explicitly enabled.
		if os.Getenv("NSELF_LICENSE_FAIL_OPEN") == "1" {
			return bundleEntitledFromCache(key, bundleName)
		}
		return false, fmt.Errorf("license validation network error (bundle=%q): %w", bundleName, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// parse and return.
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, fmt.Errorf("license key rejected by server for bundle %q (HTTP %d)", bundleName, resp.StatusCode)
	default:
		return false, fmt.Errorf("unexpected status %d from license server for bundle %q", resp.StatusCode, bundleName)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		return false, fmt.Errorf("reading bundle validation response: %w", err)
	}

	var vr bundleValidateResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		return false, fmt.Errorf("decoding bundle validation response: %w", err)
	}

	if !vr.Valid {
		reason := vr.Reason
		if reason == "" {
			reason = "not entitled to this bundle"
		}
		return false, fmt.Errorf("bundle %q: %s (tier=%q)", bundleName, reason, vr.Tier)
	}
	return true, nil
}

// bundleEntitledFromCache is the FAIL-OPEN fallback for air-gap environments.
// It reads the local license cache and checks whether the recorded tier
// covers the requested bundle. This is less authoritative than a live server
// check and must only be used when NSELF_LICENSE_FAIL_OPEN=1 is set.
func bundleEntitledFromCache(key, bundleName string) (bool, error) {
	entry, err := ReadCache()
	if err != nil || entry == nil {
		return false, fmt.Errorf("no license cache available (offline, fail-open attempted for bundle %q)", bundleName)
	}
	if entry.KeyHash != HashKey(key) {
		return false, fmt.Errorf("cached license does not match current key (bundle=%q)", bundleName)
	}

	tier := strings.ToLower(entry.Tier)
	// ɳSelf+ / owner / enterprise covers all bundles.
	if tier == "plus" || tier == "enterprise" || tier == "owner" {
		return true, nil
	}

	// Heuristic: check if every plugin in a typical bundle is in PluginsAllowed.
	// This is approximate — the authoritative check is the live server call.
	// If the cache says at least one bundle plugin is allowed, proceed with a warning.
	for _, allowed := range entry.PluginsAllowed {
		// "bundle:<name>" sentinel written by a previous live validation.
		if strings.EqualFold(allowed, "bundle:"+bundleName) {
			return true, nil
		}
	}

	return false, fmt.Errorf("bundle %q not found in cached license (tier=%q); connect to validate", bundleName, tier)
}

// CollectLicenseKey returns the first available license key from env vars or
// key files. This is a convenience helper for callers that need the raw key
// for BundleEntitled without going through the full manager flow.
func CollectLicenseKey() string {
	if k := os.Getenv("NSELF_PLUGIN_LICENSE_KEY_OWNER"); k != "" {
		return k
	}
	if k := os.Getenv("NSELF_PLUGIN_LICENSE_KEY"); k != "" {
		return k
	}
	// Fall back to stored key via manager.
	k, _ := GetKey()
	return k
}
