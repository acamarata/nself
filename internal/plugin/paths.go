package plugin

// Purpose: Path helpers and exported utility wrappers for the plugin package.
//          Provides deterministic directory paths for plugin cache, license
//          store, and license key file; plus exported wrappers around internal
//          helpers for use by cmd/ callers.
// Inputs:  OS home directory (via os.UserHomeDir); env override vars.
// Outputs: string paths; bool presence flags; int comparison result.
// Constraints: Falls back to /tmp/.nself/... when home directory is unavailable.
//              pingAPIURL honours NSELF_PING_API_URL env override (test seam).
//              findPlugin is case-insensitive on plugin name.
// SPORT: path-helpers; callers: cmd/plugin/*.go, security.go, installer.go,
//        loader.go

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultCacheDir is the exported alias of defaultCacheDir for use by
// commands that need to call FetchRegistry directly. (S58-T06)
func DefaultCacheDir() string { return defaultCacheDir() }

// FindPluginByName is the exported wrapper around findPlugin for use by
// commands outside the plugin package. (S58-T06)
func FindPluginByName(reg *Registry, name string) (*PluginManifest, bool) {
	return findPlugin(reg, name)
}

// CompareVersions compares two semver strings a and b.
// Returns -1 if a < b, 0 if a == b, +1 if a > b. (S58-T06)
func CompareVersions(a, b string) int { return compareSemver(a, b) }

// defaultCacheDir returns the default plugin cache directory.
func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "cache", "plugins")
	}
	return filepath.Join(home, ".nself", "cache", "plugins")
}

// LicenseCacheDir returns the directory used for license validation caching.
// This is ~/.nself/license/ (or /tmp/.nself/license/ if the home directory
// cannot be determined).
func LicenseCacheDir() string { return licenseCacheDir() }

// licenseCacheDir returns the directory used for license validation caching.
func licenseCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "license")
	}
	return filepath.Join(home, ".nself", "license")
}

// licenseKeyPath returns the file path where the license key is stored.
func licenseKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "license", "key")
	}
	return filepath.Join(home, ".nself", "license", "key")
}

// pingAPIURL returns the license validation API base URL.
func pingAPIURL() string {
	if u := os.Getenv("NSELF_PING_API_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "https://ping.nself.org"
}

// findPlugin locates a plugin in the registry by name (case-insensitive).
func findPlugin(reg *Registry, name string) (*PluginManifest, bool) {
	for i := range reg.Plugins {
		if strings.EqualFold(reg.Plugins[i].Name, name) {
			return &reg.Plugins[i], true
		}
	}
	return nil, false
}
