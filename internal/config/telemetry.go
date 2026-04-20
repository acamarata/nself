// Package config — user-level telemetry preference management.
//
// Telemetry preference is stored in ~/.nself/config.toml under [telemetry].
// This file does NOT ship a telemetry client at v1.0.9. The preference is
// persisted now so the v1.1.0 client can respect choices made today.
//
// Env var NSELF_TELEMETRY_OPT_OUT=1 always takes precedence over the file.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// telemetryConfigFile is the filename under ~/.nself/
const telemetryConfigFile = "config.toml"

// telemetryConfigDir is the subdirectory under the user home where the CLI
// user-level config lives.
const telemetryConfigDir = ".nself"

// TelemetryPreference describes the user's telemetry opt-in/out state.
type TelemetryPreference struct {
	// Enabled is the stored preference from config.toml.
	// This value is overridden to false when NSELF_TELEMETRY_OPT_OUT=1 is set.
	Enabled bool

	// Source describes where the value came from: "env", "config", or "default".
	Source string
}

// GetTelemetryPreference returns the effective telemetry preference.
// NSELF_TELEMETRY_OPT_OUT=1 beats the config file, which beats the default.
func GetTelemetryPreference() TelemetryPreference {
	// Env var takes highest precedence.
	if os.Getenv("NSELF_TELEMETRY_OPT_OUT") == "1" {
		return TelemetryPreference{Enabled: false, Source: "env"}
	}

	// Read from config file.
	val, err := readTelemetryEnabled()
	if err == nil {
		return TelemetryPreference{Enabled: val, Source: "config"}
	}

	// Default: telemetry is enabled unless user opts out.
	// At v1.0.9 this has no effect because no telemetry client ships.
	return TelemetryPreference{Enabled: true, Source: "default"}
}

// SetTelemetryEnabled persists the telemetry preference to ~/.nself/config.toml.
// Creates the file and directory if they do not exist.
func SetTelemetryEnabled(enabled bool) error {
	path, err := telemetryConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return writeTelemetryEnabled(path, enabled)
}

// telemetryConfigPath returns the absolute path to ~/.nself/config.toml.
func telemetryConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, telemetryConfigDir, telemetryConfigFile), nil
}

// readTelemetryEnabled reads the [telemetry] enabled value from the config file.
// Returns (false, err) when the file is missing or the key is absent.
func readTelemetryEnabled() (bool, error) {
	path, err := telemetryConfigPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return parseTelemetryEnabled(string(data))
}

// parseTelemetryEnabled extracts the `enabled` value from a [telemetry] section
// of a minimal TOML config. Uses line-by-line parsing to avoid a dependency on
// a full TOML library.
func parseTelemetryEnabled(content string) (bool, error) {
	inTelemetry := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "[telemetry]" {
			inTelemetry = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inTelemetry = false
			continue
		}
		if inTelemetry {
			kv := strings.SplitN(line, "=", 2)
			if len(kv) == 2 && strings.TrimSpace(kv[0]) == "enabled" {
				val := strings.TrimSpace(kv[1])
				if val == "true" {
					return true, nil
				}
				if val == "false" {
					return false, nil
				}
				return false, fmt.Errorf("invalid value for telemetry.enabled: %q", val)
			}
		}
	}
	return false, fmt.Errorf("telemetry.enabled not found in config")
}

// writeTelemetryEnabled writes (or updates) the [telemetry] enabled line.
// Reads the existing file if present and rewrites with the new value in-place,
// preserving all other content. Appends a [telemetry] section if absent.
func writeTelemetryEnabled(path string, enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}

	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	}

	var out strings.Builder
	inTelemetry := false
	wrote := false

	scanner := bufio.NewScanner(strings.NewReader(existing))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "[telemetry]" {
			inTelemetry = true
			out.WriteString(line + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "[") && trimmed != "[telemetry]" {
			if inTelemetry && !wrote {
				out.WriteString("enabled = " + val + "\n")
				wrote = true
			}
			inTelemetry = false
		}
		if inTelemetry {
			kv := strings.SplitN(trimmed, "=", 2)
			if len(kv) == 2 && strings.TrimSpace(kv[0]) == "enabled" {
				out.WriteString("enabled = " + val + "\n")
				wrote = true
				continue
			}
		}
		out.WriteString(line + "\n")
	}

	// If we were still in the [telemetry] section at EOF without writing
	if inTelemetry && !wrote {
		out.WriteString("enabled = " + val + "\n")
		wrote = true
	}

	// If no [telemetry] section existed, append one.
	if !wrote {
		if len(existing) > 0 && !strings.HasSuffix(existing, "\n") {
			out.WriteString("\n")
		}
		out.WriteString("[telemetry]\n")
		out.WriteString("enabled = " + val + "\n")
	}

	if err := os.WriteFile(path, []byte(out.String()), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}
