// Package license — revocation.go implements the CLI-side revocation list
// consumer (D3-T08).
//
// The CLI fetches `GET /license/revocation-list` from ping.nself.org hourly
// and stores the signed payload at `~/.config/nself/revocation-cache.json`.
// Every license validation pass consults the local cache to decide whether
// a presented JWT (jti / user_id / kid) has been revoked.
//
// FAIL-OPEN policy (per PPI § Vendor Stack — license validation fail-mode):
//   - Cache up to 7 days stale: still authoritative; refresh attempts run
//     in the background but failures don't block.
//   - Cache > 7 days stale + remote fetch fails: continue treating items
//     as NOT revoked, log a prominent warning. License-server unreachable
//     must never lock paying users out.
//
// Wire format mirrors web/backend/services/ping_api/src/routes/license/
// revocation-list.ts exactly.  Canonical JSON (sorted keys at every level,
// arrays in declared order) is the bytes the server signs and we verify.
package license

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	saferecover "github.com/nself-org/cli/internal/recover"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── Constants ────────────────────────────────────────────────────────────────

// RevocationCacheFile is the on-disk location of the cached revocation list.
// Lives under ~/.config/nself so it persists across `nself` upgrades.
const revocationCacheFile = "revocation-cache.json"

// RevocationFailOpenStaleness is the cutoff beyond which a stale cache + a
// failed remote fetch logs a prominent warning.  Within the window, stale
// caches are silently authoritative.
const RevocationFailOpenStaleness = 7 * 24 * time.Hour

// RevocationRefreshInterval is the recommended hourly refresh cadence.
const RevocationRefreshInterval = 1 * time.Hour

// revocationHTTPTimeout caps a single revocation-list fetch.
const revocationHTTPTimeout = 10 * time.Second

// ─── Types ────────────────────────────────────────────────────────────────────

// RevocationEntry is one revoked identifier.  Field order matters for the
// canonical JSON encoder; struct tags pin the wire names.
type RevocationEntry struct {
	Type      string `json:"type"`       // "jti" | "user_id" | "kid"
	ID        string `json:"id"`         // identifier value
	Reason    string `json:"reason"`     // human-readable
	RevokedAt string `json:"revoked_at"` // ISO8601
}

// RevocationList is the full signed payload from /license/revocation-list.
type RevocationList struct {
	IssuedAt   string            `json:"issued_at"`
	ValidUntil string            `json:"valid_until"`
	Kid        string            `json:"kid"`
	Revoked    []RevocationEntry `json:"revoked"`
	Signature  string            `json:"signature"`
}

// LicenseRecord is the minimal shape needed to evaluate revocation.
// Each field is optional: zero values are treated as "no match attempted".
type LicenseRecord struct {
	JTI    string
	UserID string
	Kid    string
}

// RevocationCache is the on-disk cache wrapper around RevocationList.
// FetchedAt is set by the CLI whenever a successful refresh completes.
// ETag is captured from the most recent 200 response so subsequent refreshes
// can send `If-None-Match` and let the server short-circuit with 304.
// Older caches (pre-D3-T08a) without an ETag field decode with ETag=""
// and simply skip the conditional-GET header — fully backward compatible.
type RevocationCache struct {
	List      RevocationList `json:"list"`
	FetchedAt int64          `json:"fetched_at"`     // unix seconds
	ETag      string         `json:"etag,omitempty"` // server-emitted strong ETag (sha256 hex)
}

// ─── Disk I/O ────────────────────────────────────────────────────────────────

// RevocationCachePath returns the on-disk cache location, honouring
// LICENSE_REVOCATION_CACHE_PATH for tests and unusual deployments.
func RevocationCachePath() (string, error) {
	if p := os.Getenv("LICENSE_REVOCATION_CACHE_PATH"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".config", "nself", revocationCacheFile), nil
}

// ReadRevocationCache loads the cache from disk.  Returns (nil, nil) when
// the file is absent (cold start).
func ReadRevocationCache() (*RevocationCache, error) {
	path, err := RevocationCachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading revocation cache: %w", err)
	}
	var c RevocationCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing revocation cache: %w", err)
	}
	return &c, nil
}

// WriteRevocationCache persists the cache with 0600 permissions.
func WriteRevocationCache(c *RevocationCache) error {
	path, err := RevocationCachePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling revocation cache: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// ─── Canonical JSON ──────────────────────────────────────────────────────────

// canonicalJSON produces the same byte sequence the server signed:
// objects with sorted keys at every depth, arrays in declared order.
func canonicalJSON(v any) ([]byte, error) {
	var b strings.Builder
	if err := writeCanonical(&b, v); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func writeCanonical(b *strings.Builder, v any) error {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		out, err := json.Marshal(t)
		if err != nil {
			return err
		}
		b.Write(out)
	case float64:
		out, err := json.Marshal(t)
		if err != nil {
			return err
		}
		b.Write(out)
	case json.Number:
		b.WriteString(string(t))
	case []any:
		b.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeCanonical(b, item); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			kj, err := json.Marshal(k)
			if err != nil {
				return err
			}
			b.Write(kj)
			b.WriteByte(':')
			if err := writeCanonical(b, t[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		// Fall back to standard JSON encoding for other types — re-decode
		// through json.Number so nested structures are also canonicalised.
		buf, err := json.Marshal(t)
		if err != nil {
			return err
		}
		dec := json.NewDecoder(strings.NewReader(string(buf)))
		dec.UseNumber()
		var generic any
		if err := dec.Decode(&generic); err != nil {
			return err
		}
		return writeCanonical(b, generic)
	}
	return nil
}

// canonicalForVerify reproduces the bytes the server signed: the full
// payload object minus the signature field.
func canonicalForVerify(list *RevocationList) ([]byte, error) {
	// Marshal then re-decode through json.Number so the canonicalizer
	// preserves numeric precision and key ordering at every depth.
	raw, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var generic map[string]any
	if err := dec.Decode(&generic); err != nil {
		return nil, err
	}
	delete(generic, "signature")
	return canonicalJSON(generic)
}

// ─── Signature verification ──────────────────────────────────────────────────

// VerifyRevocationSignature checks the Ed25519 signature on `list` against
// the bundled license public keys.  Returns true on a valid signature.
func VerifyRevocationSignature(list *RevocationList) bool {
	if list == nil || list.Signature == "" {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(list.Signature)
	if err != nil {
		return false
	}
	canonical, err := canonicalForVerify(list)
	if err != nil {
		return false
	}
	keys := GetPublicKeys()
	for _, pk := range keys {
		// Skip the placeholder zero key (dev builds without ldflags).
		allZero := true
		for _, b := range pk.Key {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			continue
		}
		if ed25519.Verify(pk.Key, canonical, sig) {
			return true
		}
	}
	return false
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

// revocationHTTPClient is shared so transports get reused.
var revocationHTTPClient = &http.Client{Timeout: revocationHTTPTimeout}

// SetRevocationHTTPClient overrides the HTTP client used for refreshes.
// Test-only.
func SetRevocationHTTPClient(c *http.Client) {
	revocationHTTPClient = c
}

// RefreshRevocationList fetches the signed list from ping.nself.org,
// verifies the signature, and persists it on disk.
//
// Behaviour:
//   - Network / decode error → returns the wrapped error; does NOT touch disk.
//   - Bad signature → returns ErrRevocationSignatureInvalid; does NOT persist.
//   - Success → writes cache and returns the new RevocationCache.
func RefreshRevocationList(ctx context.Context) (*RevocationCache, error) {
	url := strings.TrimRight(PingURL(), "/") + "/license/revocation-list"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	// Send conditional GET when we have a stored cache + ETag to allow 304s.
	// The server (revocation-list.ts) honours If-None-Match against a strong
	// sha256-hex ETag; If-Modified-Since is NOT honoured. Cold-start caches
	// (no ETag yet) simply skip the conditional header and pull a full 200.
	if existing, _ := ReadRevocationCache(); existing != nil && existing.ETag != "" {
		req.Header.Set("If-None-Match", existing.ETag)
	}

	resp, err := revocationHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching revocation list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		// Server confirms cache is current; touch FetchedAt so the
		// staleness window resets even though contents didn't change.
		existing, _ := ReadRevocationCache()
		if existing != nil {
			existing.FetchedAt = time.Now().Unix()
			if err := WriteRevocationCache(existing); err != nil {
				return nil, err
			}
			return existing, nil
		}
		// 304 with no local cache is contractually impossible but
		// graceful enough — fall through to a full fetch attempt
		// next call.
		return nil, fmt.Errorf("304 received with no local cache")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf(
			"revocation list HTTP %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("reading revocation list body: %w", err)
	}

	var list RevocationList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("parsing revocation list: %w", err)
	}

	if !VerifyRevocationSignature(&list) {
		// Skip verification only when the build has no signing key AND
		// no test override is in effect.  Production builds always
		// verify; test runs with LICENSE_PUBLIC_KEY_OVERRIDE always
		// verify.
		if !IsZeroPubKey() || os.Getenv("LICENSE_PUBLIC_KEY_OVERRIDE") != "" {
			return nil, ErrRevocationSignatureInvalid
		}
	}

	cache := &RevocationCache{
		List:      list,
		FetchedAt: time.Now().Unix(),
		ETag:      resp.Header.Get("ETag"),
	}
	if err := WriteRevocationCache(cache); err != nil {
		return nil, fmt.Errorf("persisting revocation cache: %w", err)
	}
	return cache, nil
}

// ErrRevocationSignatureInvalid signals that a fetched payload failed
// signature verification.  Callers should treat this as a refresh failure
// and keep the existing cache untouched.
var ErrRevocationSignatureInvalid = fmt.Errorf("revocation list signature invalid")

// ─── Lookup (FAIL-OPEN) ──────────────────────────────────────────────────────

// IsRecordRevoked reports whether the supplied license record is on the
// cached revocation list.  Implements the FAIL-OPEN policy described at
// the top of this file.
//
// Returns false (not revoked) when:
//   - the cache file is absent (cold start);
//   - the cache is older than RevocationFailOpenStaleness (7 days) AND
//     the most recent refresh attempt failed;
//   - the cached list does not contain a matching identifier.
//
// A prominent warning is printed to stderr (once per process) when the
// cache is fail-open-stale.
//
// Disambiguation: this is distinct from the legacy IsRevoked() in
// revoke.go, which reports whether the local "license.revoked" marker
// file is present (the marker is an out-of-band signal written by
// `nself license revoke` for plugin dormancy).
func IsRecordRevoked(rec LicenseRecord) bool {
	cache, err := ReadRevocationCache()
	if err != nil || cache == nil {
		return false
	}
	if isStaleFailOpen(cache) {
		warnFailOpenOnce(cache)
		return false
	}
	return matchAny(&cache.List, rec)
}

// matchAny scans the revocation list for any entry matching the record.
func matchAny(list *RevocationList, rec LicenseRecord) bool {
	for i := range list.Revoked {
		e := &list.Revoked[i]
		switch e.Type {
		case "jti":
			if rec.JTI != "" && e.ID == rec.JTI {
				return true
			}
		case "user_id":
			if rec.UserID != "" && e.ID == rec.UserID {
				return true
			}
		case "kid":
			if rec.Kid != "" && e.ID == rec.Kid {
				return true
			}
		}
	}
	return false
}

// isStaleFailOpen reports whether the cache is past the FAIL-OPEN cutoff.
func isStaleFailOpen(cache *RevocationCache) bool {
	if cache == nil {
		return true
	}
	fetched := time.Unix(cache.FetchedAt, 0)
	return time.Since(fetched) > RevocationFailOpenStaleness
}

// ─── Stale-warning suppression (one log per process) ────────────────────────

var (
	failOpenWarnedMu sync.Mutex
	failOpenWarned   bool
)

// warnFailOpenOnce prints the prominent stderr warning at most once per
// CLI invocation.
func warnFailOpenOnce(cache *RevocationCache) {
	failOpenWarnedMu.Lock()
	defer failOpenWarnedMu.Unlock()
	if failOpenWarned {
		return
	}
	failOpenWarned = true
	age := time.Since(time.Unix(cache.FetchedAt, 0)).Round(time.Hour)
	fmt.Fprintf(os.Stderr,
		"warning: revocation list cache is %s stale (>7d) and last refresh failed; treating licenses as not revoked\n",
		age,
	)
}

// ResetFailOpenWarning clears the once-per-process warning latch.  Test-only.
func ResetFailOpenWarning() {
	failOpenWarnedMu.Lock()
	failOpenWarned = false
	failOpenWarnedMu.Unlock()
}

// ─── Background refresher ───────────────────────────────────────────────────

// StartRevocationRefresher launches a background goroutine that calls
// RefreshRevocationList every RevocationRefreshInterval until ctx is done.
// Refresh failures are logged but do not stop the loop — the FAIL-OPEN
// policy is enforced at lookup time, not here.
//
// Returns a stop function that the caller can defer to halt the loop
// before ctx is cancelled.
func StartRevocationRefresher(ctx context.Context) (stop func()) {
	loopCtx, cancel := context.WithCancel(ctx)
	saferecover.SafeGo("license_revocation_refresh", func() {
		// Initial best-effort refresh on startup.
		if _, err := RefreshRevocationList(loopCtx); err != nil {
			fmt.Fprintf(os.Stderr,
				"warning: initial revocation refresh failed: %v\n", err)
		}
		ticker := time.NewTicker(RevocationRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				if _, err := RefreshRevocationList(loopCtx); err != nil {
					fmt.Fprintf(os.Stderr,
						"warning: revocation refresh failed: %v\n", err)
				}
			}
		}
	})
	return cancel
}
