// Package upgrade implements release channel selection and version resolution
// for the `nself upgrade` command.
//
// Three channels are supported:
//
//   - stable: public releases (tags matching `vX.Y.Z` with no suffix).
//   - rc:     release candidates (tags matching `vX.Y.Z-rc<N>`).
//   - edge:   nightly / pre-rc (tags matching `vX.Y.Z-edge.<date>.<sha>`).
//
// Channel preference is persisted to ~/.config/nself/channel.json so repeated
// `nself upgrade` invocations default to the user's chosen channel.
//
// For Homebrew-installed users, each channel maps to a distinct formula name
// (`nself`, `nself-rc`, `nself-edge`) so `brew upgrade nself-rc` pulls the rc
// lineage without crossing channels.
//
// Cross-ref:
//   - .claude/docs/operations/rc-tagging.md (tag format + graduation flow)
//   - cmd/commands/upgrade.go (command wiring)
//   - S34-T07 (channels) + S34-T08 (--channel flag)
package upgrade

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Channel is one of the three supported release channels.
type Channel string

const (
	ChannelStable Channel = "stable"
	ChannelRC     Channel = "rc"
	ChannelEdge   Channel = "edge"
)

// DefaultChannel is the channel used when no preference has been saved.
const DefaultChannel = ChannelStable

// AllChannels lists every accepted channel name, in display order.
var AllChannels = []Channel{ChannelStable, ChannelRC, ChannelEdge}

// channelConfig is the on-disk representation of the user's chosen channel.
type channelConfig struct {
	Channel string `json:"channel"`
}

// IsValid returns true if s is one of the accepted channel names.
func IsValid(s string) bool {
	for _, c := range AllChannels {
		if string(c) == s {
			return true
		}
	}
	return false
}

// Parse returns the Channel value for s, or an error if s is not a valid
// channel name.
func Parse(s string) (Channel, error) {
	if !IsValid(s) {
		return "", fmt.Errorf("invalid channel %q: must be one of: stable, rc, edge", s)
	}
	return Channel(s), nil
}

// ConfigPath returns the absolute path to the channel preference file.
// If the user's home directory cannot be determined, a temp-dir fallback
// is returned so the CLI never panics on a broken environment.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".nself", "channel.json")
	}
	return filepath.Join(home, ".config", "nself", "channel.json")
}

// LoadChannel reads the persisted channel preference, returning DefaultChannel
// if none is set or the file is unreadable or malformed. Failure to read is
// never fatal — a missing or corrupt file simply falls back to stable.
func LoadChannel() Channel {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return DefaultChannel
	}
	var cfg channelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultChannel
	}
	if cfg.Channel == "" || !IsValid(cfg.Channel) {
		return DefaultChannel
	}
	return Channel(cfg.Channel)
}

// SaveChannel persists the user's channel preference to ConfigPath. The parent
// directory is created at 0755 if missing. The config file itself is written
// at 0644 since it contains no secrets (just "stable" / "rc" / "edge").
func SaveChannel(c Channel) error {
	if !IsValid(string(c)) {
		return fmt.Errorf("invalid channel %q", c)
	}
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.Marshal(channelConfig{Channel: string(c)})
	if err != nil {
		return fmt.Errorf("marshaling channel config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing channel config: %w", err)
	}
	return nil
}

// FormulaName returns the Homebrew formula name for the given channel.
//
//	stable -> "nself"
//	rc     -> "nself-rc"
//	edge   -> "nself-edge"
//
// Separate formulae per channel let Homebrew users pin to a channel via
// `brew upgrade nself-rc` without crossing into stable.
func FormulaName(c Channel) string {
	switch c {
	case ChannelRC:
		return "nself-rc"
	case ChannelEdge:
		return "nself-edge"
	default:
		return "nself"
	}
}

// TagSuffix returns the version-tag suffix convention for the channel.
//
//	stable -> ""         (e.g. v1.0.9)
//	rc     -> "-rc"      (e.g. v1.0.9-rc3)
//	edge   -> "-edge"    (e.g. v1.0.9-edge.20260418.a3f7c21)
//
// Callers use this to filter the GitHub releases list when resolving the
// latest tag on a channel.
func TagSuffix(c Channel) string {
	switch c {
	case ChannelRC:
		return "-rc"
	case ChannelEdge:
		return "-edge"
	default:
		return ""
	}
}

// ClassifyTag returns the channel a given tag belongs to based on its suffix.
//
//	"v1.0.9"                          -> stable
//	"v1.0.9-rc1"                      -> rc
//	"v1.0.9-edge.20260418.a3f7c21"    -> edge
//
// Tags that do not match any known pattern are classified as stable (the
// safest default for unexpected tag shapes).
func ClassifyTag(tag string) Channel {
	// Strip leading "v" for consistent matching.
	t := strings.TrimPrefix(tag, "v")
	// Split on the first hyphen after the semver digits.
	// Semver core (major.minor.patch) has no hyphens; the pre-release segment
	// starts with the FIRST hyphen that follows digits.
	idx := strings.Index(t, "-")
	if idx == -1 {
		return ChannelStable
	}
	suffix := t[idx:]
	switch {
	case strings.HasPrefix(suffix, "-rc"):
		return ChannelRC
	case strings.HasPrefix(suffix, "-edge"):
		return ChannelEdge
	default:
		return ChannelStable
	}
}

// PingAPIEndpoint returns the ping.nself.org endpoint that serves the latest
// release tag for the given channel. ping_api proxies the GitHub releases API
// and caches results to avoid GitHub rate limits for offline/enterprise users.
//
//	stable -> https://ping.nself.org/release/stable/latest
//	rc     -> https://ping.nself.org/release/rc/latest
//	edge   -> https://ping.nself.org/release/edge/latest
//
// Response body: `{"tag": "v1.0.9", "url": "https://github.com/...", "sha256": "..."}`
func PingAPIEndpoint(c Channel) string {
	base := "https://ping.nself.org/release"
	switch c {
	case ChannelRC:
		return base + "/rc/latest"
	case ChannelEdge:
		return base + "/edge/latest"
	default:
		return base + "/stable/latest"
	}
}
