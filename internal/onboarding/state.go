// Package onboarding reads state files written by existing CLI commands to
// determine where a user is in the 6-stage onboarding funnel.
//
// State files live under ~/.config/nself/ and are written by:
//   - install-id       — Q07 telemetry client (install identity)
//   - projects.json    — nself init
//   - last-start.timestamp — nself start
//   - plugin-state.yaml — nself plugin install
//   - query-count      — Hasura telemetry hook (optional)
//   - usage.log        — nself start (one line per invocation)
package onboarding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// nSelfConfigDir returns ~/.config/nself (XDG-style, same as rest of CLI).
// When testConfigDirOverride is set (tests only), it returns that directory
// instead so state reads are redirected to a temporary path.
func nSelfConfigDir() string {
	if testConfigDirOverride != "" {
		return testConfigDirOverride
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "nself")
}

// InstallID reads ~/.config/nself/install-id.
// Returns empty string if absent (Q07 not yet run).
func InstallID() string {
	path := filepath.Join(nSelfConfigDir(), "install-id")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ProjectsCount returns the number of initialized projects from
// ~/.config/nself/projects.json. Returns 0 if absent.
func ProjectsCount() int {
	path := filepath.Join(nSelfConfigDir(), "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var projects []interface{}
	if err := json.Unmarshal(data, &projects); err != nil {
		return 0
	}
	return len(projects)
}

// LastStartTime reads ~/.config/nself/last-start.timestamp.
// Returns zero time if absent.
func LastStartTime() time.Time {
	path := filepath.Join(nSelfConfigDir(), "last-start.timestamp")
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	raw := strings.TrimSpace(string(data))
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		// Try unix timestamp fallback
		return time.Time{}
	}
	return t
}

// FirstInstalledPlugin returns the name of the first installed plugin from
// ~/.config/nself/plugin-state.yaml, plus a tier string. Returns ("", "") if absent.
func FirstInstalledPlugin() (name, tier string) {
	path := filepath.Join(nSelfConfigDir(), "plugin-state.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	// Simple YAML line scan — avoids a YAML dependency.
	// Expected format (per plugin install writer):
	//   plugins:
	//     - name: ai
	//       tier: pro
	//       installed_at: 2026-04-01T00:00:00Z
	lines := strings.Split(string(data), "\n")
	var currentName, currentTier string
	inPlugins := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "plugins:" {
			inPlugins = true
			continue
		}
		if !inPlugins {
			continue
		}
		if strings.HasPrefix(trimmed, "- name:") {
			currentName = strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:"))
			currentTier = ""
		} else if strings.HasPrefix(trimmed, "name:") && currentName == "" {
			currentName = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		} else if strings.HasPrefix(trimmed, "tier:") {
			currentTier = strings.TrimSpace(strings.TrimPrefix(trimmed, "tier:"))
		}
		if currentName != "" && currentTier != "" {
			return currentName, currentTier
		}
	}
	// Return name even without tier
	if currentName != "" {
		return currentName, "free"
	}
	return "", ""
}

// QueryCount reads ~/.config/nself/query-count.
// Returns -1 if absent (UNKNOWN, e.g. self-host without telemetry hook).
// Returns 0+ when the file exists.
func QueryCount() int {
	path := filepath.Join(nSelfConfigDir(), "query-count")
	data, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	raw := strings.TrimSpace(string(data))
	n := 0
	for _, ch := range raw {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		} else {
			break
		}
	}
	return n
}

// RecentStartCount returns the number of nself start invocations in the last
// 7 days by counting lines in ~/.config/nself/usage.log that contain a
// timestamp within the window. Returns 0 if absent.
func RecentStartCount() int {
	path := filepath.Join(nSelfConfigDir(), "usage.log")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each line may be an RFC3339 timestamp or "timestamp event" pair.
		// Try the first field as a timestamp.
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		t, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			// Treat any non-empty line as a start event if timestamp not parseable
			count++
			continue
		}
		if t.After(cutoff) {
			count++
		}
	}
	return count
}
