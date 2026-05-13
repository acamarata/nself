// version_resolver.go — S2.T22: resolve bundle slugs + release channels into
// concrete plugin version pins. Reads plugins-pro/registry.json via the
// existing internal/plugin registry fetcher so the same fallback + cache
// chain (primary URL → GitHub raw → stale cache) applies.
//
// Also provides cross-bundle MAX-resolution: when multiple bundles are
// installed and share plugins, the installed version is always the MAX
// (newest) of what each bundle requires — never a silent downgrade.
package bundle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nself-org/cli/internal/plugin"
)

// Channel is a release-channel identifier (stable/beta/canary). Channels filter
// candidate registry entries by their PublishStatus before version selection:
//
//	stable → status in {"", "stable"}
//	beta   → status in {"", "stable", "beta"}
//	canary → all installable statuses (adds "experimental")
//
// "deprecated" and "eol" entries are never resolved by this function regardless
// of channel — those install paths require explicit --allow-eol acknowledgment
// through plugin.CheckEOLBlock, not silent bundle install.
type Channel string

const (
	ChannelStable Channel = "stable"
	ChannelBeta   Channel = "beta"
	ChannelCanary Channel = "canary"
)

// VersionPins maps plugin slug → concrete semver version string for every
// installable plugin in a bundle, after channel filtering.
type VersionPins map[string]string

// ResolveBundleVersions returns the resolved version pins for every plugin in
// the given bundle, using the highest registry version that satisfies the
// channel filter. Returns an error for unknown bundle slugs or non-installable
// bundles (nself-plus, ntask).
//
// Network behavior: delegates to plugin.FetchRegistry which respects the
// registry cache + fallback chain. A registry fetch failure with no stale
// cache returns an error; callers should respect that.
func ResolveBundleVersions(ctx context.Context, bundleSlug string, channel Channel) (VersionPins, error) {
	b, ok := Get(bundleSlug)
	if !ok {
		return nil, UnknownBundleError(bundleSlug)
	}
	if !b.IsInstallable() {
		return nil, fmt.Errorf("bundle %q is not installable as a unit (meta or free); install constituent plugins individually", b.Slug)
	}

	ch := normalizeChannel(channel)

	reg, err := plugin.FetchRegistry(ctx, "", defaultRegistryCacheDir())
	if err != nil {
		return nil, fmt.Errorf("fetching registry for bundle %q: %w", b.Slug, err)
	}

	pins := make(VersionPins, len(b.Plugins))
	for _, name := range b.Plugins {
		v, vErr := resolveOnePluginVersion(reg, name, ch)
		if vErr != nil {
			return nil, fmt.Errorf("bundle %q: %w", b.Slug, vErr)
		}
		pins[name] = v
	}
	return pins, nil
}

// ResolveVersion returns the version pin for a single plugin under the given
// channel. Exposed for callers that need to pin one plugin (e.g. compat-check,
// install --version<latest-channel>).
func ResolveVersion(ctx context.Context, pluginName string, channel Channel) (string, error) {
	ch := normalizeChannel(channel)
	reg, err := plugin.FetchRegistry(ctx, "", defaultRegistryCacheDir())
	if err != nil {
		return "", fmt.Errorf("fetching registry for plugin %q: %w", pluginName, err)
	}
	return resolveOnePluginVersion(reg, pluginName, ch)
}

// resolveOnePluginVersion finds the registry entry matching pluginName, applies
// the channel filter, and returns the concrete version string. With the
// current registry shape (one entry per plugin) this is a direct lookup;
// the function is structured to accept multi-version registries in the future
// without breaking callers.
func resolveOnePluginVersion(reg *plugin.Registry, pluginName string, channel Channel) (string, error) {
	if reg == nil {
		return "", fmt.Errorf("nil registry")
	}
	for i := range reg.Plugins {
		p := &reg.Plugins[i]
		if !strings.EqualFold(p.Name, pluginName) {
			continue
		}
		if !channelAllows(channel, p.PublishStatus) {
			return "", fmt.Errorf("plugin %q (status=%q) not eligible for channel %q",
				pluginName, p.PublishStatus, channel)
		}
		if strings.TrimSpace(p.Version) == "" {
			return "", fmt.Errorf("plugin %q has no version in registry", pluginName)
		}
		return p.Version, nil
	}
	return "", fmt.Errorf("plugin %q not found in registry", pluginName)
}

// channelAllows reports whether a registry PublishStatus is eligible under the
// given channel. The matrix matches the doc-strings on Channel above.
func channelAllows(ch Channel, status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	switch ch {
	case ChannelStable:
		return s == "" || s == "stable"
	case ChannelBeta:
		return s == "" || s == "stable" || s == "beta"
	case ChannelCanary:
		return s == "" || s == "stable" || s == "beta" || s == "experimental"
	default:
		// Should not happen — normalizeChannel guards. Defensive: stable-only.
		return s == "" || s == "stable"
	}
}

// normalizeChannel maps any case/whitespace variant to a canonical Channel.
// Unknown values fall back to ChannelStable rather than erroring — release
// channels are an operator convenience, not a security gate.
func normalizeChannel(ch Channel) Channel {
	switch Channel(strings.ToLower(strings.TrimSpace(string(ch)))) {
	case ChannelBeta:
		return ChannelBeta
	case ChannelCanary:
		return ChannelCanary
	case "", ChannelStable:
		return ChannelStable
	default:
		return ChannelStable
	}
}

// defaultRegistryCacheDir returns the default cache directory used by the
// registry client. Mirrors the logic in cmd/commands/plugin.go so this
// package does not depend on the cmd layer.
func defaultRegistryCacheDir() string {
	if d := os.Getenv("NSELF_PLUGIN_CACHE"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "cache", "plugins")
	}
	return filepath.Join(home, ".nself", "cache", "plugins")
}

// ---------------------------------------------------------------------------
// Cross-bundle MAX-resolution (S2.T22)
// ---------------------------------------------------------------------------

// ResolveMaxVersion returns the maximum (newest) semver string from versions.
// All versions must be valid semver (with or without a leading "v" prefix).
// Returns an error if versions is empty or any entry is not valid semver.
// The returned string preserves the "v" prefix of the winning candidate.
//
// Example:
//
//	ResolveMaxVersion("ai", []string{"1.0.0", "1.2.0", "1.1.0"}) → "1.2.0", nil
func ResolveMaxVersion(pluginName string, versions []string) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("plugin %q: no version requirements provided", pluginName)
	}
	max := versions[0]
	for _, v := range versions[1:] {
		cmp, err := compareSemver(v, max)
		if err != nil {
			return "", fmt.Errorf("plugin %q: invalid semver %q: %w", pluginName, v, err)
		}
		if cmp > 0 {
			max = v
		}
	}
	// Validate the initial candidate too (catches bad first entry).
	if _, err := parseSemver(versions[0]); err != nil {
		return "", fmt.Errorf("plugin %q: invalid semver %q: %w", pluginName, versions[0], err)
	}
	return max, nil
}

// ResolveConflicts takes a map of plugin slug → list of version requirements
// from different bundles and returns the resolved map of plugin slug → max
// version. If strict is true, any plugin that has more than one distinct
// version requirement returns an error instead of resolving to the max.
//
// A "conflict" in strict mode means two or more bundles require different
// versions of the same plugin. When strict is false (the default), conflicts
// are silently resolved by taking the highest (MAX) version.
func ResolveConflicts(requirements map[string][]string, strict bool) (map[string]string, error) {
	resolved := make(map[string]string, len(requirements))
	for plugin, vers := range requirements {
		if len(vers) == 0 {
			return nil, fmt.Errorf("plugin %q: empty version list", plugin)
		}
		if strict && hasConflict(vers) {
			return nil, fmt.Errorf(
				"plugin %q has conflicting version requirements across bundles: %s",
				plugin, strings.Join(vers, ", "),
			)
		}
		max, err := ResolveMaxVersion(plugin, vers)
		if err != nil {
			return nil, err
		}
		resolved[plugin] = max
	}
	return resolved, nil
}

// hasConflict reports whether a slice contains more than one distinct version.
func hasConflict(versions []string) bool {
	if len(versions) <= 1 {
		return false
	}
	first := strings.TrimPrefix(strings.TrimSpace(versions[0]), "v")
	for _, v := range versions[1:] {
		if strings.TrimPrefix(strings.TrimSpace(v), "v") != first {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Minimal semver helpers (no external dependency)
// ---------------------------------------------------------------------------

// semver holds the parsed components of a version string.
type semver struct {
	major, minor, patch int
	pre                 string // pre-release label (e.g. "alpha.1", "beta.2")
	raw                 string // original string
}

// parseSemver parses a semver string with an optional leading "v".
// Supports MAJOR.MINOR.PATCH, MAJOR.MINOR, and MAJOR forms.
// Pre-release labels after "-" are preserved for string comparison only;
// a version with a pre-release label sorts lower than the same base version
// without one (standard semver rule).
func parseSemver(s string) (semver, error) {
	raw := s
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return semver{}, fmt.Errorf("empty version string")
	}

	// Split off pre-release.
	pre := ""
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		pre = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.SplitN(s, ".", 3)
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, fmt.Errorf("non-numeric component %q in %q", p, raw)
		}
		nums[i] = n
	}
	return semver{
		major: nums[0],
		minor: nums[1],
		patch: nums[2],
		pre:   pre,
		raw:   raw,
	}, nil
}

// compareSemver returns:
//
//	> 0 if a > b
//	  0 if a == b
//	< 0 if a < b
func compareSemver(a, b string) (int, error) {
	pa, err := parseSemver(a)
	if err != nil {
		return 0, err
	}
	pb, err := parseSemver(b)
	if err != nil {
		return 0, err
	}

	if d := pa.major - pb.major; d != 0 {
		return d, nil
	}
	if d := pa.minor - pb.minor; d != 0 {
		return d, nil
	}
	if d := pa.patch - pb.patch; d != 0 {
		return d, nil
	}

	// Pre-release: no pre-release > pre-release (standard semver rule).
	switch {
	case pa.pre == "" && pb.pre == "":
		return 0, nil
	case pa.pre == "" && pb.pre != "":
		return 1, nil
	case pa.pre != "" && pb.pre == "":
		return -1, nil
	default:
		return strings.Compare(pa.pre, pb.pre), nil
	}
}
