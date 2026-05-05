// Package license — grace.go implements the license grace period state machine
// and degradation mode enforcement.
//
// States: valid -> grace_soft -> grace_hard -> expired -> revoked
// Grace periods:
//   - <24h offline: proceed silently (valid)
//   - 24h-7d offline: WARNING banner (grace_soft)
//   - >7d offline: read-only degraded mode (grace_hard)
//   - Expired license: refuse to start (expired)
//   - Revoked license: refuse to start (revoked)
package license

import (
	"fmt"
	"time"
)

// GraceState represents the license grace period state.
type GraceState string

const (
	// GraceValid means the license is validated and current.
	GraceValid GraceState = "valid"
	// GraceSoft means the cache is 24h-7d old; show warning banner.
	GraceSoft GraceState = "grace_soft"
	// GraceHard means the cache is >7d old; paid plugin writes are refused.
	GraceHard GraceState = "grace_hard"
	// GraceExpired means the license has expired.
	GraceExpired GraceState = "expired"
	// GraceRevoked means the license was explicitly revoked.
	GraceRevoked GraceState = "revoked"
)

// GraceSoftThreshold is when the soft grace warning starts (24 hours).
const GraceSoftThreshold = 24 * time.Hour

// GraceHardThreshold is when hard degradation begins (7 days).
// Configurable via LICENSE_GRACE_DAYS env var.
const GraceHardThreshold = 7 * 24 * time.Hour

// GraceCheckResult contains the outcome of a grace period check.
type GraceCheckResult struct {
	State        GraceState
	CacheAge     time.Duration
	ExpiresAt    time.Time
	Tier         string
	Message      string
	CanProceed   bool
	WriteAllowed bool
}

// DetermineGraceState evaluates the current grace state from a cache entry.
// If entry is nil, returns GraceExpired (no cache means no validation).
func DetermineGraceState(entry *CacheEntry) GraceCheckResult {
	if entry == nil {
		return GraceCheckResult{
			State:        GraceExpired,
			Message:      "No license cache found. Run 'nself license validate' to validate your license.",
			CanProceed:   false,
			WriteAllowed: false,
		}
	}

	now := time.Now()
	cacheAge := entry.CacheAge()
	expiresAt := time.Unix(entry.ExpiresAt, 0)

	// Check if the license itself has expired (server-reported expiry).
	if entry.ExpiresAt > 0 && now.After(expiresAt) {
		return GraceCheckResult{
			State:        GraceExpired,
			CacheAge:     cacheAge,
			ExpiresAt:    expiresAt,
			Tier:         entry.Tier,
			Message:      fmt.Sprintf("License expired on %s. Renew at https://nself.org/pricing", expiresAt.Format("2006-01-02")),
			CanProceed:   false,
			WriteAllowed: false,
		}
	}

	// Evaluate based on cache freshness.
	switch {
	case cacheAge < GraceSoftThreshold:
		return GraceCheckResult{
			State:        GraceValid,
			CacheAge:     cacheAge,
			ExpiresAt:    expiresAt,
			Tier:         entry.Tier,
			CanProceed:   true,
			WriteAllowed: true,
		}
	case cacheAge < GraceHardThreshold:
		return GraceCheckResult{
			State:     GraceSoft,
			CacheAge:  cacheAge,
			ExpiresAt: expiresAt,
			Tier:      entry.Tier,
			Message: fmt.Sprintf(
				"License validation is %s old. Connect to the internet to refresh.\n"+
					"Offline grace period expires in %s.",
				formatDuration(cacheAge),
				formatDuration(GraceHardThreshold-cacheAge),
			),
			CanProceed:   true,
			WriteAllowed: true,
		}
	default:
		return GraceCheckResult{
			State:     GraceHard,
			CacheAge:  cacheAge,
			ExpiresAt: expiresAt,
			Tier:      entry.Tier,
			Message: fmt.Sprintf(
				"License validation expired (%s offline). "+
					"Paid plugins are in read-only mode. Connect to refresh.",
				formatDuration(cacheAge),
			),
			CanProceed:   true,
			WriteAllowed: false,
		}
	}
}

// IsWriteAllowed checks if write operations on paid plugins are permitted
// given the current grace state.
func (r GraceCheckResult) IsWriteAllowed() bool {
	return r.WriteAllowed
}

// NeedsBanner returns true if a warning banner should be displayed.
func (r GraceCheckResult) NeedsBanner() bool {
	return r.State == GraceSoft || r.State == GraceHard
}

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	hours := int(d.Hours())
	if hours < 48 {
		return fmt.Sprintf("%d hours", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%d days", days)
}
