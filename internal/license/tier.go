package license

// Purpose: the single definition of which license tiers grant every plugin.
// Inputs:  a tier string as reported by the license server or the local cache.
// Outputs: whether that tier entitles all bundles/plugins.
// Constraints: pure function, no I/O. Must stay the ONLY place this rule is
//   written — it existed twice before, and the copy the plugin installer used
//   did not have it at all.
// SPORT: CLI-LICENSE-TIER-001

import "strings"

// allAccessTiers are the tiers that include every bundle and every plugin.
//
// ɳSelf+ ("plus") is the all-bundles subscription. "owner" is the internal
// all-access key. "enterprise" is the top commercial tier. Everything else is
// a single-bundle or lower tier and must be entitled per plugin.
var allAccessTiers = map[string]bool{
	"plus":       true,
	"owner":      true,
	"enterprise": true,
}

// IsAllAccessTier reports whether tier includes every plugin.
//
// This rule lived inline in exactly one place (bundleEntitledFromCache) while
// the plugin installer enforced entitlements from an explicit plugin list only.
// A valid ɳSelf+ licence whose server response carried an empty plugin list was
// therefore refused with "license tier does not include this plugin", because
// nothing on that path ever looked at the tier. Keeping the rule in one
// exported function is what stops the two paths disagreeing again.
//
// Deliberately a closed allowlist, not a "not free" test: an unknown tier
// returns false, so a new tier has to be added here on purpose rather than
// silently inheriting all-access.
func IsAllAccessTier(tier string) bool {
	return allAccessTiers[strings.ToLower(strings.TrimSpace(tier))]
}
