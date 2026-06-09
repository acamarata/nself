package plugin

// Purpose: Plugin enable/disable state and table-namespace conflict detection.
//          DisablePlugin/EnablePlugin toggle a .disabled marker file; table
//          helpers prevent two plugins from claiming the same DB namespace.
// Inputs:  plugin name string; pluginDir string; PluginManifest for new plugin.
// Outputs: error on conflict, missing plugin, or filesystem failure; nil on success.
// Constraints: .disabled marker is a zero-byte file in the plugin's directory.
//              Table prefix is the first two underscore-separated segments
//              (e.g. "np_chat_messages" → "np_chat_"). Prefix conflicts block
//              install; exact-name conflicts also block install.
// SPORT: plugin-config; callers: cmd/plugin/disable.go, cmd/plugin/enable.go,
//        installLocked in installer.go

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DisablePlugin creates a .disabled marker file in the plugin's directory,
// causing it to be excluded from compose files on the next build.
func DisablePlugin(name, pluginDir string) error {
	destDir := filepath.Join(pluginDir, name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	markerPath := filepath.Join(destDir, ".disabled")
	if _, err := os.Stat(markerPath); err == nil {
		return fmt.Errorf("plugin %q is already disabled", name)
	}

	f, err := os.Create(markerPath)
	if err != nil {
		return fmt.Errorf("creating disable marker: %w", err)
	}
	f.Close()
	return nil
}

// EnablePlugin removes the .disabled marker file from the plugin's directory,
// allowing it to be included in compose files on the next build.
func EnablePlugin(name, pluginDir string) error {
	destDir := filepath.Join(pluginDir, name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	markerPath := filepath.Join(destDir, ".disabled")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not disabled", name)
	}

	if err := os.Remove(markerPath); err != nil {
		return fmt.Errorf("removing disable marker: %w", err)
	}
	return nil
}

// IsDisabled returns true if the named plugin has a .disabled marker file.
func IsDisabled(name, pluginDir string) bool {
	markerPath := filepath.Join(pluginDir, name, ".disabled")
	_, err := os.Stat(markerPath)
	return err == nil
}

// checkTablePrefixConflict scans all installed plugins in pluginDir and
// returns an error if any of them share a table prefix with the tables listed
// in newTables. Table prefixes are derived from table names by taking the
// first two underscore-separated segments followed by a trailing underscore
// (e.g. "np_chat_messages" → prefix "np_chat_"). The newPluginName parameter
// is used to skip the plugin being installed (allowing reinstalls/updates).
func checkTablePrefixConflict(pluginDir, newPluginName string, newTables []string) error {
	if len(newTables) == 0 {
		return nil
	}

	newPrefixes := make(map[string]bool)
	for _, table := range newTables {
		if p := tablePrefix(table); p != "" {
			newPrefixes[p] = true
		}
	}
	if len(newPrefixes) == 0 {
		return nil
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading plugin directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.EqualFold(entry.Name(), newPluginName) {
			continue
		}
		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.json")
		m, err := parseManifest(manifestPath)
		if err != nil {
			continue
		}
		for _, table := range m.Tables {
			p := tablePrefix(table)
			if p != "" && newPrefixes[p] {
				return fmt.Errorf("plugin %s conflicts with installed plugin %s: table prefix %q already claimed",
					newPluginName, m.Name, p)
			}
		}
	}
	return nil
}

// checkTableConflicts scans all installed plugins in pluginDir and returns an
// error if any of them declare a table name that also appears in newPlugin's
// Tables list. This catches exact name collisions (the prefix check above
// catches broader namespace conflicts).
func checkTableConflicts(pluginDir string, newPlugin *PluginManifest) error {
	if len(newPlugin.Tables) == 0 {
		return nil
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == newPlugin.Name {
			continue
		}
		existing, err := parseManifest(filepath.Join(pluginDir, entry.Name(), "plugin.json"))
		if err != nil {
			continue
		}
		for _, newTable := range newPlugin.Tables {
			for _, existingTable := range existing.Tables {
				if newTable == existingTable {
					return fmt.Errorf("table %q already used by plugin %q", newTable, existing.Name)
				}
			}
		}
	}
	return nil
}

// tablePrefix extracts the two-segment prefix from a table name. Given
// "np_chat_messages" it returns "np_chat_". Returns empty string for table
// names that do not have at least two underscore-separated segments.
func tablePrefix(table string) string {
	parts := strings.Split(table, "_")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "_" + parts[1] + "_"
}
