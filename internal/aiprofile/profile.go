// Package aiprofile implements AI_PROFILE enforcement, budget caps,
// feature flags, per-tier timeouts, and rate-limit coordination.
// Sprint P88-S05: T-05-06 through T-05-10.
package aiprofile

import (
	"fmt"
	"os"
	"strings"
)

// Profile represents the AI routing profile.
type Profile string

const (
	ProfileAuto      Profile = "auto"
	ProfileLocalOnly Profile = "local_only"
	ProfilePoolOnly  Profile = "pool_only"
	ProfileOAuthOnly Profile = "oauth_only"
	ProfileCustom    Profile = "custom"
)

// ParseProfile parses an AI_PROFILE string into a typed Profile.
func ParseProfile(s string) (Profile, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "auto", "":
		return ProfileAuto, nil
	case "local_only":
		return ProfileLocalOnly, nil
	case "pool_only":
		return ProfilePoolOnly, nil
	case "oauth_only":
		return ProfileOAuthOnly, nil
	case "custom":
		return ProfileCustom, nil
	default:
		return "", fmt.Errorf("unknown AI_PROFILE: %q (valid: auto, local_only, pool_only, oauth_only, custom)", s)
	}
}

// LoadProfile reads AI_PROFILE from the environment.
func LoadProfile() Profile {
	p, err := ParseProfile(os.Getenv("AI_PROFILE"))
	if err != nil {
		return ProfileAuto
	}
	return p
}

// Tier represents a routing tier.
type Tier string

const (
	TierLocal Tier = "local"
	TierPool  Tier = "pool"
	TierOAuth Tier = "oauth"
	TierPaid  Tier = "paid"
)

// AllowedTiers returns the tiers permitted by the given profile.
func AllowedTiers(p Profile) []Tier {
	switch p {
	case ProfileLocalOnly:
		return []Tier{TierLocal}
	case ProfilePoolOnly:
		return []Tier{TierPool}
	case ProfileOAuthOnly:
		return []Tier{TierOAuth}
	case ProfileCustom:
		// Custom means routing table as-is; all tiers allowed.
		return []Tier{TierLocal, TierPool, TierOAuth, TierPaid}
	default: // auto
		return []Tier{TierLocal, TierPool, TierOAuth, TierPaid}
	}
}

// IsTierAllowed checks whether a tier is permitted by the current profile.
func IsTierAllowed(p Profile, t Tier) bool {
	for _, allowed := range AllowedTiers(p) {
		if allowed == t {
			return true
		}
	}
	return false
}

// FilterTierChain filters a tier chain based on the profile. Returns only
// allowed tiers in the original order. Used by the router at load time (T-05-06).
func FilterTierChain(p Profile, chain []Tier) []Tier {
	var filtered []Tier
	for _, t := range chain {
		if IsTierAllowed(p, t) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
