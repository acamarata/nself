package license

// tier_test.go — Guards the all-access tier rule.
//
// Purpose: this rule decides whether a licence grants every paid plugin, so it
//   is a revenue boundary in both directions. Too narrow and a paying ɳSelf+
//   customer is refused (which is what happened: the plugin installer never
//   consulted the tier at all, so a valid "plus" licence with an empty plugin
//   list was rejected as "license tier does not include this plugin"). Too
//   broad and every paid plugin is free.
// Inputs:  tier strings as they arrive from the licence server or local cache.
// Outputs: assertions on IsAllAccessTier.
// Constraints: pure; no network, no cache, no filesystem.

import "testing"

// TestIsAllAccessTier_GrantsTheThreeAllAccessTiers pins what must be allowed.
// Removing any of these silently paywalls a customer who has paid for it.
func TestIsAllAccessTier_GrantsTheThreeAllAccessTiers(t *testing.T) {
	t.Parallel()

	for _, tier := range []string{"plus", "owner", "enterprise"} {
		if !IsAllAccessTier(tier) {
			t.Errorf("%q must be all-access — it is a paid all-bundles tier", tier)
		}
	}

	// Case and whitespace vary by source (server JSON vs cache file).
	for _, tier := range []string{"Plus", "PLUS", "  plus  ", "Owner", "ENTERPRISE"} {
		if !IsAllAccessTier(tier) {
			t.Errorf("%q must normalise to all-access", tier)
		}
	}
}

// TestIsAllAccessTier_DeniesEverythingElse is the revenue-protecting half, and
// the more important one. A bundle tier must NOT inherit all-access — its
// entitlements come from the plugin list, per bundle.
func TestIsAllAccessTier_DeniesEverythingElse(t *testing.T) {
	t.Parallel()

	// Every non-all-access product in validProductPrefixes, plus the obvious
	// absent/garbage cases.
	deny := []string{
		"claw", "clawde", "chat", "media", "family", // single-bundle tiers
		"pro", "max", // paid but NOT all-bundles
		"free", "trial", "expired", "revoked",
		"", "   ", "unknown", "null", "true", "all", "*",
	}
	for _, tier := range deny {
		tier := tier
		t.Run("deny/"+tier, func(t *testing.T) {
			t.Parallel()
			if IsAllAccessTier(tier) {
				t.Errorf("%q must NOT be all-access — that would give away every paid plugin", tier)
			}
		})
	}
}

// TestIsAllAccessTier_IsAClosedAllowlist pins the shape of the rule, not just
// its current answers. Written as "not free" or "anything paid", a new tier
// added later would inherit all-access silently. It must be an explicit list so
// adding a tier is a deliberate act.
func TestIsAllAccessTier_IsAClosedAllowlist(t *testing.T) {
	t.Parallel()

	if len(allAccessTiers) != 3 {
		t.Errorf("all-access set has %d entries, expected exactly plus/owner/enterprise — "+
			"adding one grants every paid plugin to that tier", len(allAccessTiers))
	}
	for _, want := range []string{"plus", "owner", "enterprise"} {
		if !allAccessTiers[want] {
			t.Errorf("%q missing from the all-access set", want)
		}
	}
	// A tier nobody has defined must not be all-access by default.
	if IsAllAccessTier("bundle-that-does-not-exist-yet") {
		t.Error("unknown tiers must default to NOT all-access")
	}
}
