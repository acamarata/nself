// Package bundle implements the canonical nSelf paid plugin bundle catalog
// plus install/remove orchestration with license validation, version pinning,
// and atomic rollback on failure (S2.T03/T04/T22).
//
// Bundle membership is authoritative here — the source of truth that mirrors
// .claude/docs/sport/F06-BUNDLE-INVENTORY.md. The cmd layer must consume this
// package; do NOT duplicate the bundle map elsewhere.
package bundle

import (
	"fmt"
	"sort"
	"strings"
)

// Bundle describes a canonical nSelf plugin bundle.
type Bundle struct {
	Name        string   // Display name (with ɳ glyph)
	Slug        string   // CLI-arg lowercase identifier
	Price       string   // Human-readable price string
	Description string   // Optional one-line description
	Plugins     []string // Plugin slugs in the bundle (nil for meta-bundle)
}

// canonicalBundles is the authoritative bundle membership map.
// Mirrors F06-BUNDLE-INVENTORY.md verbatim. Any update here MUST update F06.
var canonicalBundles = map[string]Bundle{
	"nclaw": {
		Slug:  "nclaw",
		Name:  "ɳClaw",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			// "ai" = legacy paid/ai (3709); canonical AI suite post-E6 = "ai-gateway" (3761)
			"ai", "ai-gateway", "ai-cc", "ai-mcp",
			"claw", "claw-web", "mux", "voice", "browser",
			"google", "notify", "cron", "claw-budget", "claw-news",
			"mcp", "knowledge-base",
		},
	},
	"nchat": {
		Slug:  "nchat",
		Name:  "ɳChat",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"chat", "livekit", "recording", "moderation", "bots", "realtime", "auth",
			"support",
		},
	},
	"nfamily": {
		Slug:  "nfamily",
		Name:  "ɳFamily",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"social", "photos", "activity-feed", "moderation", "realtime", "cms", "chat",
			"geolocation", "calendar",
		},
	},
	"ntv": {
		Slug:  "ntv",
		Name:  "ɳTV",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"media-processing", "streaming", "epg", "tmdb", "podcast", "recording",
			"game-metadata", "file-processing", "subtitle-manager", "vpn", "stream-gateway",
		},
	},
	"clawde": {
		Slug:  "clawde",
		Name:  "ClawDE",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"claw",
			// "ai" = legacy paid/ai (3709); canonical AI suite post-E6 = "ai-gateway" (3761)
			"ai", "ai-gateway", "ai-cc", "ai-mcp",
			"realtime", "auth", "notify", "cms",
			"mcp", "knowledge-base",
		},
	},
	"nsentry": {
		Slug:  "nsentry",
		Name:  "ɳSentry",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"nself-uptime-monitor", "nself-status-page", "nself-incident-mgmt",
			"nself-alert-router", "nself-slo-tracker", "nself-synthetic-monitor",
			"nself-rum", "nself-errors", "nself-cron-monitor", "nself-oncall",
			"nself-crash", "nself-anomaly", "nself-audit",
		},
	},
	"nself-plus": {
		Slug:        "nself-plus",
		Name:        "ɳSelf+",
		Price:       "$3.99/mo or $39.99/yr",
		Description: "All 6 paid bundles + all apps + support",
		Plugins:     nil, // meta-bundle — no individual plugin list
	},
	"ntask": {
		Slug:        "ntask",
		Name:        "ɳTask",
		Price:       "FREE",
		Description: "Free bundle — uses free plugins only",
		Plugins:     []string{"(free plugins only — see: nself plugin list)"},
	},
}

// DisplayOrder is the canonical print order for bundle listings.
var DisplayOrder = []string{
	"nclaw", "nchat", "nfamily", "ntv", "clawde", "nsentry", "ntask", "nself-plus",
}

// Get returns the Bundle for the given slug. The slug is case-insensitive and
// trimmed. Returns ok=false when the slug is unknown; callers can call Names()
// to render a useful error hint.
func Get(slug string) (Bundle, bool) {
	key := strings.ToLower(strings.TrimSpace(slug))
	b, ok := canonicalBundles[key]
	return b, ok
}

// Names returns every known bundle slug sorted alphabetically. Useful for
// error messages and shell completion.
func Names() []string {
	out := make([]string, 0, len(canonicalBundles))
	for k := range canonicalBundles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// All returns every Bundle in canonical display order.
func All() []Bundle {
	out := make([]Bundle, 0, len(canonicalBundles))
	for _, slug := range DisplayOrder {
		if b, ok := canonicalBundles[slug]; ok {
			out = append(out, b)
		}
	}
	return out
}

// IsInstallable reports whether the bundle can be installed via
// `nself bundle install`. Meta-bundles (nself-plus) and free bundles (ntask)
// are not installable as a unit — operators install their constituent plugins
// directly via `nself plugin install`.
func (b Bundle) IsInstallable() bool {
	if b.Slug == "nself-plus" {
		return false
	}
	if b.Slug == "ntask" {
		// ntask is free and its plugin list is a placeholder — users install
		// free plugins individually.
		return false
	}
	if len(b.Plugins) == 0 {
		return false
	}
	return true
}

// UnknownBundleError formats a friendly error when a slug does not match a
// known bundle. Includes the sorted list of valid names as a hint.
func UnknownBundleError(slug string) error {
	return fmt.Errorf(
		"unknown bundle %q\n\nAvailable bundles: %s",
		slug, strings.Join(Names(), ", "),
	)
}
