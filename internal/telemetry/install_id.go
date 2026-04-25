package telemetry

// install_id.go — anonymous install-ID management and opt-out gate.
//
// LoadOrCreateInstallID returns a persistent UUID v4 stored at
// ~/.config/nself/install-id. The file is created with 0600 permissions on
// first run. The ID is purely random; it contains no personal data.
//
// IsEnabled returns false when:
//   - NSELF_TELEMETRY=0 is set (env override, highest priority), or
//   - NSELF_TELEMETRY_OPT_OUT=1 is set (legacy override), or
//   - ~/.nself/config.toml [telemetry] enabled = false has been written.
//
// Default when neither override is present: true (opt-in by default as of v1.1.0).

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	// installIDFile is the path under the user home directory where the
	// anonymous install-ID UUID is stored.
	installIDFile = ".config/nself/install-id"

	// telemetryStateFile is the path under the user home directory where
	// the simple "true"/"false" telemetry opt-in state is stored.
	// This is distinct from the TOML config.toml path used by the
	// `nself telemetry on/off` commands.
	telemetryStateFile = ".config/nself/telemetry"
)

// LoadOrCreateInstallID returns the persistent anonymous install-ID UUID.
// On first call it creates ~/.config/nself/install-id with a random UUID v4.
// Subsequent calls return the same value. The file is never deleted by the CLI.
// If the home directory cannot be determined, returns an empty string.
func LoadOrCreateInstallID() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, installIDFile)

	// Read existing ID.
	if data, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id
		}
	}

	// Create a new UUID v4.
	id := uuid.New().String()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return id // best-effort: return ID even if persistence fails
	}
	if err := os.WriteFile(path, []byte(id), 0600); err != nil {
		return id // best-effort
	}
	return id
}

// IsEnabled reports whether CLI telemetry is currently active.
// Priority order (highest to lowest):
//  1. NSELF_TELEMETRY=0       → disabled
//  2. NSELF_TELEMETRY_OPT_OUT=1 → disabled (legacy env var)
//  3. ~/.config/nself/telemetry == "false" → disabled
//  4. ~/.nself/config.toml [telemetry] enabled = false → disabled
//  5. Default → enabled
//
// IsEnabled does NOT check NSELF_TELEMETRY_OPT_IN (the old strict-opt-in gate
// used in v1.0.9 by IsOptedIn). That gate is preserved for backward compatibility
// but IsEnabled is the canonical check for the v1.1.0+ telemetry client.
func IsEnabled() bool {
	// 1. New env var override.
	if v := os.Getenv("NSELF_TELEMETRY"); v != "" {
		return v != "0" && v != "false"
	}

	// 2. Legacy opt-out env var.
	if os.Getenv("NSELF_TELEMETRY_OPT_OUT") == "1" {
		return false
	}

	// 3. ~/.config/nself/telemetry state file (written by future first-run prompt).
	if enabled, ok := readStatefile(); ok {
		return enabled
	}

	// 4. ~/.nself/config.toml [telemetry] (written by `nself telemetry on/off`).
	if enabled, ok := readConfigToml(); ok {
		return enabled
	}

	// 5. Default: enabled.
	return true
}

// readStatefile reads the simple true/false state file at ~/.config/nself/telemetry.
// Returns (value, true) when the file exists and is parseable; (false, false) otherwise.
func readStatefile() (bool, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, false
	}
	data, err := os.ReadFile(filepath.Join(home, telemetryStateFile))
	if err != nil {
		return false, false
	}
	v := strings.TrimSpace(string(data))
	switch v {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

// readConfigToml reads the [telemetry] enabled value from ~/.nself/config.toml.
// Returns (value, true) when the file and key exist; (false, false) otherwise.
func readConfigToml() (bool, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, false
	}
	data, err := os.ReadFile(filepath.Join(home, ".nself", "config.toml"))
	if err != nil {
		return false, false
	}
	inSection := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "[telemetry]" {
			inSection = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = false
			continue
		}
		if inSection {
			kv := strings.SplitN(line, "=", 2)
			if len(kv) == 2 && strings.TrimSpace(kv[0]) == "enabled" {
				val := strings.TrimSpace(kv[1])
				switch val {
				case "true":
					return true, true
				case "false":
					return false, true
				}
			}
		}
	}
	return false, false
}

// WriteStatefile writes the telemetry enabled state to ~/.config/nself/telemetry.
// This is used by the first-run prompt and by `nself telemetry on/off`.
func WriteStatefile(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determining home directory: %w", err)
	}
	path := filepath.Join(home, telemetryStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating telemetry config dir: %w", err)
	}
	val := "false"
	if enabled {
		val = "true"
	}
	if err := os.WriteFile(path, []byte(val+"\n"), 0600); err != nil {
		return fmt.Errorf("writing telemetry state: %w", err)
	}
	return nil
}
