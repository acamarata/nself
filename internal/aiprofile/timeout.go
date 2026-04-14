package aiprofile

import (
	"os"
	"strconv"
	"time"
)

// TierTimeouts holds per-tier timeout overrides from env vars (T-05-09).
// Zero means "use routing defaults from task config".
type TierTimeouts struct {
	Local time.Duration // AI_TIMEOUT_LOCAL_MS
	OAuth time.Duration // AI_TIMEOUT_OAUTH_MS
	Pool  time.Duration // AI_TIMEOUT_POOL_MS
	Paid  time.Duration // AI_TIMEOUT_PAID_MS
}

// LoadTierTimeouts reads AI_TIMEOUT_*_MS env vars.
func LoadTierTimeouts() TierTimeouts {
	return TierTimeouts{
		Local: envDurationMS("AI_TIMEOUT_LOCAL_MS"),
		OAuth: envDurationMS("AI_TIMEOUT_OAUTH_MS"),
		Pool:  envDurationMS("AI_TIMEOUT_POOL_MS"),
		Paid:  envDurationMS("AI_TIMEOUT_PAID_MS"),
	}
}

// TimeoutForTier returns the timeout override for the given tier, or zero
// if no override is set (caller should use its own default).
func (t TierTimeouts) TimeoutForTier(tier Tier) time.Duration {
	switch tier {
	case TierLocal:
		return t.Local
	case TierOAuth:
		return t.OAuth
	case TierPool:
		return t.Pool
	case TierPaid:
		return t.Paid
	default:
		return 0
	}
}

func envDurationMS(key string) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	ms, err := strconv.Atoi(v)
	if err != nil || ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
