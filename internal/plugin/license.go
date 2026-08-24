package plugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/httptimeout"
)

// ErrRateLimited is returned when the license validation server responds with
// HTTP 429 Too Many Requests. RetryAfter holds the value of the Retry-After
// header (in seconds) as a string, defaulting to "60" when the header is absent.
type ErrRateLimited struct {
	RetryAfter string
}

func (e *ErrRateLimited) Error() string {
	return fmt.Sprintf("rate limited — retry after %ss", e.RetryAfter)
}

// IsRateLimited reports whether err is (or wraps) an ErrRateLimited.
func IsRateLimited(err error) (retryAfter string, ok bool) {
	var e *ErrRateLimited
	if errors.As(err, &e) {
		return e.RetryAfter, true
	}
	return "", false
}

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
	"activity-feed":  true,
	"cms":            true,
	"documents":      true,
	"knowledge-base": true,
	"moderation":     true,
	"photos":         true,
	"social":         true,
	"support":        true,
	"calendar":       true,
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

	resp, err := httptimeout.License.Do(req)
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
	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			return false, nil, &ErrRateLimited{RetryAfter: retryAfter}
		}
		return false, nil, &ErrRateLimited{RetryAfter: "60"}
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false, nil, nil
	default:
		return false, nil, fmt.Errorf("unexpected license validation status: %d", resp.StatusCode)
	}
}
