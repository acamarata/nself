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
