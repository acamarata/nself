package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// executablePath is a package-level hook so tests can override which binary
// path selfUpdateFromURL (and the upgrade command's binary-url path) targets.
// In production it always delegates to os.Executable.
var executablePath = os.Executable

// channelConfig persists the user's chosen release channel.
type channelConfig struct {
	Channel string `json:"channel"` // "stable" or "canary"
}

// channelConfigPath returns the path to the channel config file.
func channelConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".nself", "channel.json")
	}
	return filepath.Join(home, ".config", "nself", "channel.json")
}

// loadChannel reads the persisted channel preference. Returns "stable" if none set.
func loadChannel() string {
	data, err := os.ReadFile(channelConfigPath())
	if err != nil {
		return "stable"
	}
	var cfg channelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "stable"
	}
	if cfg.Channel == "" {
		return "stable"
	}
	return cfg.Channel
}

// saveChannel persists the channel preference.
func saveChannel(channel string) error {
	path := channelConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.Marshal(channelConfig{Channel: channel})
	if err != nil {
		return fmt.Errorf("marshaling channel config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
