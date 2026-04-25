// Package deprecation — G4: sunset date computation for per-plugin API entries.
package deprecation

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// PluginEndpoint represents a single deprecated HTTP endpoint within a plugin.
type PluginEndpoint struct {
	Path         string `yaml:"path"`
	DeprecatedIn string `yaml:"deprecated_in"`
	RemovedIn    string `yaml:"removed_in"`
	Replacement  string `yaml:"replacement"`
	Reason       string `yaml:"reason"`
}

// PluginEntry represents a single plugin's API version entry in registry.yaml.
type PluginEntry struct {
	Name                string           `yaml:"name"`
	APIVersion          string           `yaml:"api_version"`
	DeprecatedEndpoints []PluginEndpoint `yaml:"deprecated_endpoints"`
}

// pluginRegistryFile is the YAML envelope for the plugins section.
type pluginRegistryFile struct {
	Plugins []PluginEntry `yaml:"plugins"`
}

// PluginRegistry holds the per-plugin versioning entries loaded from registry.yaml.
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]PluginEntry // keyed by plugin name
}

// LoadPluginRegistry reads the plugins section from the registry YAML file.
// Returns an empty PluginRegistry (not nil) on any error.
func LoadPluginRegistry(path string) (*PluginRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &PluginRegistry{plugins: make(map[string]PluginEntry)},
			fmt.Errorf("deprecation: registry not found at %s: %w", path, err)
	}

	var rf pluginRegistryFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return &PluginRegistry{plugins: make(map[string]PluginEntry)},
			fmt.Errorf("deprecation: failed to parse plugins section: %w", err)
	}

	pr := &PluginRegistry{plugins: make(map[string]PluginEntry, len(rf.Plugins))}
	for _, p := range rf.Plugins {
		pr.plugins[p.Name] = p
	}
	return pr, nil
}

// LookupPlugin returns the PluginEntry for the given plugin name and true if found.
func (pr *PluginRegistry) LookupPlugin(name string) (PluginEntry, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	e, ok := pr.plugins[name]
	return e, ok
}

// AllPlugins returns all plugin entries sorted by name.
func (pr *PluginRegistry) AllPlugins() []PluginEntry {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	out := make([]PluginEntry, 0, len(pr.plugins))
	for _, p := range pr.plugins {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SunsetCalendarEntry is one row in the human-readable sunset calendar for a plugin.
type SunsetCalendarEntry struct {
	Plugin       string
	Path         string
	DeprecatedIn string
	RemovedIn    string
	Replacement  string
	Reason       string
	SunsetDate   time.Time // zero value when not applicable
}

// SunsetDate returns the SunsetCalendarEntries for a plugin, sorted by RemovedIn.
// If the plugin is not found in the registry, it returns nil.
func (pr *PluginRegistry) SunsetDate(pluginName string) []SunsetCalendarEntry {
	entry, ok := pr.LookupPlugin(pluginName)
	if !ok {
		return nil
	}

	out := make([]SunsetCalendarEntry, 0, len(entry.DeprecatedEndpoints))
	for _, ep := range entry.DeprecatedEndpoints {
		out = append(out, SunsetCalendarEntry{
			Plugin:       pluginName,
			Path:         ep.Path,
			DeprecatedIn: ep.DeprecatedIn,
			RemovedIn:    ep.RemovedIn,
			Replacement:  ep.Replacement,
			Reason:       ep.Reason,
		})
	}

	// Sort by RemovedIn string (semver strings sort lexicographically within the same major).
	sort.Slice(out, func(i, j int) bool {
		return out[i].RemovedIn < out[j].RemovedIn
	})
	return out
}

// HTTPSunsetHeader formats a Sunset HTTP header value for the given removal version.
// Returns an empty string if removedIn is empty.
func HTTPSunsetHeader(removedIn string) string {
	if removedIn == "" {
		return ""
	}
	// Map known major removal versions to concrete dates.
	sunsetMap := map[string]string{
		"2.0.0": "Sat, 01 Jan 2027 00:00:00 GMT",
		"1.5.0": "Thu, 01 Jul 2026 00:00:00 GMT",
		"1.4.0": "Wed, 01 Apr 2026 00:00:00 GMT",
		"1.3.0": "Thu, 01 Jan 2026 00:00:00 GMT",
	}
	if v, ok := sunsetMap[removedIn]; ok {
		return v
	}
	// Fallback: 2027-01-01 for any unrecognised removal version.
	return "Sat, 01 Jan 2027 00:00:00 GMT"
}
