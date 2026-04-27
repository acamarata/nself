// Package license — validator.go implements the FAIL-OPEN license validation
// policy per memory/decisions.md (D3-T10, P97).
//
// Policy summary:
//   - Cache valid + within TTL              → Valid
//   - Cache valid + remote 200 (verified)   → Valid
//   - Cache valid + remote unreachable + age ≤ 7d   → Valid (FAIL-OPEN, silent)
//   - Cache valid + remote unreachable + age 7-14d  → Valid (FAIL-OPEN, warning)
//   - Cache valid + remote unreachable + age > 14d  → FailClosed
//   - Cache signature invalid OR tampered           → FailClosed (NEVER fail-open)
//   - Cache absent + remote unreachable             → FailClosed
//   - Remote 200 with revoked                       → Revoked (overrides cache)
//   - Remote auth failure (401/403)                 → FailClosed (NOT FAIL-OPEN)
//   - Remote transport / 5xx error                  → FAIL-OPEN if cache permits
//
// Network-failure classification distinguishes transport errors from auth
// failures: only transport-class failures qualify for FAIL-OPEN. Auth failures
// indicate explicit policy decisions from the server and must fail-closed.
package license

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ValidatorStatus represents the validator's terminal status decision.
type ValidatorStatus string

const (
	// StatusValid means the license is current and the operation may proceed.
	StatusValid ValidatorStatus = "valid"
	// StatusExpired means the license itself has passed its expires_at.
	StatusExpired ValidatorStatus = "expired"
	// StatusRevoked means the server explicitly revoked the license.
	StatusRevoked ValidatorStatus = "revoked"
	// StatusFailOpen means cache permits proceeding even though the remote was
	// unreachable. CanProceed is true; a stderr warning may be emitted.
	StatusFailOpen ValidatorStatus = "fail_open"
	// StatusFailClosed means the operation must NOT proceed. Bad signature,
	// stale cache beyond hard TTL, missing cache, or auth failure.
	StatusFailClosed ValidatorStatus = "fail_closed"
)

// FailOpenSoftTTL is the silent-fail-open window. ≤ this value, no warning.
// Configurable for tests via the validator's clock; default 7 days.
const FailOpenSoftTTL = 7 * 24 * time.Hour

// FailOpenHardTTL is the absolute fail-open ceiling. Beyond this, fail-closed.
// Default 14 days.
const FailOpenHardTTL = 14 * 24 * time.Hour

// ValidatorResult is the FAIL-OPEN-aware validation outcome.
type ValidatorResult struct {
	Status      ValidatorStatus
	CanProceed  bool
	FromCache   bool
	Tier        string
	Plugins     []string
	CacheAge    time.Duration
	Reason      string
	WarnMessage string // non-empty when caller should emit a stderr warning
}

// Clock abstracts time.Now for testability.
type Clock interface {
	Now() time.Time
}

// realClock returns the wall clock time.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// HTTPDoer abstracts http.Client.Do for testability.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ValidatorOptions configures Validate.
type ValidatorOptions struct {
	// Clock is used for all time comparisons. nil → wall clock.
	Clock Clock
	// HTTPClient is used for the remote validation call. nil → default 30s client.
	HTTPClient HTTPDoer
	// PingURL overrides the configured ping endpoint. Empty → PingURL().
	PingURL string
	// SkipSignatureVerify allows skipping signature verification for tests.
	// Production callers MUST leave this false.
	SkipSignatureVerify bool
	// WarnOnce is a hook called at most once per process when emitting a
	// FAIL-OPEN warning. nil → default: write to os.Stderr once.
	WarnOnce func(msg string)
}

// failOpenWarnOnce gates the default stderr warning to one emission per process.
//
//nolint:gochecknoglobals // intentional process-lifetime gate
var failOpenWarnOnce sync.Once

func defaultWarnOnce(msg string) {
	failOpenWarnOnce.Do(func() {
		fmt.Fprintln(os.Stderr, msg)
	})
}

// Validate executes the FAIL-OPEN license validation flow.
//
// Steps:
//  1. Try remote validation (when HTTPClient + key supplied). Distinguish:
//     - 200 valid:           update cache, return Valid
//     - 200 invalid/revoked: return Revoked
//     - 401/403:             return FailClosed (auth failure, NOT fail-open)
//     - 5xx / transport err: fall through to cache-only path (fail-open eligible)
//  2. Cache-only path:
//     - cache absent:         FailClosed
//     - signature invalid:    FailClosed (never fail-open on tamper)
//     - cache age ≤ soft TTL: FailOpen (silent — no warning)
//     - cache age ≤ hard TTL: FailOpen (warning)
//     - cache age > hard TTL: FailClosed
//
// `key` may be empty to validate from cache only (no remote call).
func Validate(ctx context.Context, key string, opts *ValidatorOptions) (*ValidatorResult, error) {
	if opts == nil {
		opts = &ValidatorOptions{}
	}
	clk := opts.Clock
	if clk == nil {
		clk = realClock{}
	}
	warnFn := opts.WarnOnce
	if warnFn == nil {
		warnFn = defaultWarnOnce
	}

	// 1) Try remote — but only if a key is supplied.
	if key != "" {
		remote, kind, err := tryRemote(ctx, key, opts)
		switch kind {
		case remoteOK:
			// Persist successful response to cache.
			entry := responseToCache(key, remote)
			if writeErr := WriteCache(entry); writeErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not update license cache: %v\n", writeErr)
			}
			if !remote.Valid {
				// Server says revoked/expired. This OVERRIDES cache.
				status := StatusRevoked
				reason := remote.Reason
				if reason == "" {
					reason = "license rejected by server"
				}
				if strings.Contains(strings.ToLower(reason), "expired") {
					status = StatusExpired
				}
				return &ValidatorResult{
					Status:     status,
					CanProceed: false,
					FromCache:  false,
					Reason:     reason,
				}, nil
			}
			return &ValidatorResult{
				Status:     StatusValid,
				CanProceed: true,
				FromCache:  false,
				Tier:       remote.Tier,
				Plugins:    remote.Plugins,
			}, nil
		case remoteAuthFail:
			// 401/403 — explicit auth rejection. NEVER fail-open here.
			return &ValidatorResult{
				Status:     StatusFailClosed,
				CanProceed: false,
				FromCache:  false,
				Reason:     fmt.Sprintf("license rejected by server (auth): %v", err),
			}, nil
		case remoteTransientFail:
			// transport error / 5xx — fall through to fail-open eligibility check
		}
	}

	// 2) Cache-only path.
	entry, cacheErr := ReadCache()
	if cacheErr != nil || entry == nil {
		return &ValidatorResult{
			Status:     StatusFailClosed,
			CanProceed: false,
			FromCache:  false,
			Reason:     "no license cache available and remote unreachable",
		}, nil
	}

	// Verify signature unless explicitly skipped (tests).
	if !opts.SkipSignatureVerify && !IsZeroPubKey() && !entry.VerifySignature() {
		return &ValidatorResult{
			Status:     StatusFailClosed,
			CanProceed: false,
			FromCache:  true,
			Reason:     "license cache signature invalid (tampered or rotated key) — refusing fail-open",
		}, nil
	}

	// If a key was supplied, ensure the cache matches.
	if key != "" && entry.KeyHash != HashKey(key) {
		return &ValidatorResult{
			Status:     StatusFailClosed,
			CanProceed: false,
			FromCache:  true,
			Reason:     "license cache does not match supplied key",
		}, nil
	}

	// Check server-reported expiry first.
	if entry.ExpiresAt > 0 {
		expiresAt := time.Unix(entry.ExpiresAt, 0)
		if clk.Now().After(expiresAt) {
			return &ValidatorResult{
				Status:     StatusExpired,
				CanProceed: false,
				FromCache:  true,
				Tier:       entry.Tier,
				Reason:     fmt.Sprintf("license expired on %s", expiresAt.Format("2006-01-02")),
			}, nil
		}
	}

	age := clk.Now().Sub(time.Unix(entry.FetchedAt, 0))
	switch {
	case age <= FailOpenSoftTTL:
		return &ValidatorResult{
			Status:     StatusFailOpen,
			CanProceed: true,
			FromCache:  true,
			Tier:       entry.Tier,
			Plugins:    entry.PluginsAllowed,
			CacheAge:   age,
			Reason:     "remote unreachable; using cached license within soft TTL",
		}, nil
	case age <= FailOpenHardTTL:
		warn := fmt.Sprintf(
			"WARNING: nSelf license validation has been offline for %s. "+
				"Connect to the internet within %s or paid plugins will go dormant.",
			formatDuration(age),
			formatDuration(FailOpenHardTTL-age),
		)
		warnFn(warn)
		return &ValidatorResult{
			Status:      StatusFailOpen,
			CanProceed:  true,
			FromCache:   true,
			Tier:        entry.Tier,
			Plugins:     entry.PluginsAllowed,
			CacheAge:    age,
			Reason:      "remote unreachable; using cached license within hard TTL (warning issued)",
			WarnMessage: warn,
		}, nil
	default:
		return &ValidatorResult{
			Status:     StatusFailClosed,
			CanProceed: false,
			FromCache:  true,
			Tier:       entry.Tier,
			CacheAge:   age,
			Reason: fmt.Sprintf(
				"license cache is %s old, exceeding %s fail-open ceiling — refusing to proceed",
				formatDuration(age), formatDuration(FailOpenHardTTL),
			),
		}, nil
	}
}

// remoteOutcome is an internal enum classifying the outcome of tryRemote.
type remoteOutcome int

const (
	remoteOK remoteOutcome = iota
	remoteAuthFail
	remoteTransientFail
)

// tryRemote performs the HTTP POST to /license/validate and classifies the
// outcome. Returns:
//
//	remoteOK            → server returned 200 (resp parsed)
//	remoteAuthFail      → server returned 401 or 403 (explicit reject; NOT fail-open)
//	remoteTransientFail → transport error or 5xx (fail-open eligible)
func tryRemote(ctx context.Context, key string, opts *ValidatorOptions) (*ValidateResponse, remoteOutcome, error) {
	pingURL := opts.PingURL
	if pingURL == "" {
		pingURL = PingURL()
	}
	type request struct {
		LicenseKey string `json:"license_key"`
		Product    string `json:"product"`
	}
	body, err := json.Marshal(request{LicenseKey: key, Product: "plugins-pro"})
	if err != nil {
		return nil, remoteTransientFail, fmt.Errorf("marshalling request: %w", err)
	}

	url := strings.TrimRight(pingURL, "/") + "/license/validate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, remoteTransientFail, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var doer HTTPDoer = opts.HTTPClient
	if doer == nil {
		doer = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := doer.Do(req)
	if err != nil {
		// Transport error — definitely fail-open eligible.
		return nil, remoteTransientFail, fmt.Errorf("network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Drain body for connection reuse and to capture potential error details.
	rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	switch {
	case resp.StatusCode == http.StatusOK:
		var vr ValidateResponse
		if err := json.Unmarshal(rawBody, &vr); err != nil {
			// 200 but unparseable → transient (don't punish user for server bug).
			return nil, remoteTransientFail, fmt.Errorf("decoding response: %w", err)
		}
		return &vr, remoteOK, nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, remoteAuthFail, fmt.Errorf("server returned %d", resp.StatusCode)
	case resp.StatusCode >= 500:
		return nil, remoteTransientFail, fmt.Errorf("server returned %d", resp.StatusCode)
	default:
		// 4xx other than 401/403 — treat as auth-class to avoid silent fail-open
		// on misuse (e.g. 400 bad-request); the caller's cache may not save them.
		// Conservative posture: NOT fail-open.
		return nil, remoteAuthFail, fmt.Errorf("server returned %d", resp.StatusCode)
	}
}

// errCacheMissing is returned by atomic write helpers when the destination is
// unset (placeholder for future use).
var errCacheMissing = errors.New("license: cache path unavailable")

var _ = errCacheMissing // unused-var guard for future callers
