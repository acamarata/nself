package plugin

// tier_list.go — `nself plugin list --available` support.
//
// Purpose: surface every registry entry for a slug served more than once
// (OWNER-ACTIONS.md item 15), marking which tier a plain
// `nself plugin install <name>` (no --tier override) currently resolves to,
// so an operator can see the free/pro split and the license-driven default
// before installing rather than being surprised by it.
// Inputs: none beyond the fetched Registry (via FetchRegistry) and an
// EntitlementFunc (nil selects defaultEntitlement, same as ResolvePlugin).
// Outputs: one TieredPluginInfo per registry entry, grouped by slug and
// ordered by first appearance in the registry.
// Constraints: never collapses an unresolved (non-tier_pair) duplicate down
// to one row — both entries are listed with IsDefault=false, matching
// ResolvePlugin's refusal to guess at install time.

import (
	"context"
	"fmt"
	"strings"
)

// TieredPluginInfo describes one registry entry as shown by
// `nself plugin list --available`.
type TieredPluginInfo struct {
	Name      string
	Tier      string
	Version   string
	Category  string
	TierPair  bool
	IsDefault bool // true if ResolvePlugin(ctx, reg, Name, "", entitled) picks this exact entry
}

// ListAvailableWithTiers fetches the registry and returns every entry,
// grouped by case-insensitive slug in first-seen order, with IsDefault set on
// whichever entry a tier-override-free install would currently resolve to.
func ListAvailableWithTiers(ctx context.Context, entitled EntitlementFunc) ([]TieredPluginInfo, error) {
	cacheDir := defaultCacheDir()
	reg, err := FetchRegistry(ctx, "", cacheDir)
	if err != nil {
		return nil, fmt.Errorf("fetching registry: %w", err)
	}

	order := make([]string, 0, len(reg.Plugins))
	groups := make(map[string][]*PluginManifest, len(reg.Plugins))
	for i := range reg.Plugins {
		key := strings.ToLower(reg.Plugins[i].Name)
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], &reg.Plugins[i])
	}

	out := make([]TieredPluginInfo, 0, len(reg.Plugins))
	for _, key := range order {
		entries := groups[key]

		var defaultEntry *PluginManifest
		if resolved, rerr := ResolvePlugin(ctx, reg, entries[0].Name, "", entitled); rerr == nil {
			defaultEntry = resolved
		}

		for _, e := range entries {
			out = append(out, TieredPluginInfo{
				Name:      e.Name,
				Tier:      e.Tier,
				Version:   e.Version,
				Category:  e.Category,
				TierPair:  e.TierPair,
				IsDefault: defaultEntry == e,
			})
		}
	}
	return out, nil
}
