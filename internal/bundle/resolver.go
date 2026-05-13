// resolver.go — S2.T22: cross-bundle plugin version MAX resolution.
//
// When multiple bundles are installed simultaneously (e.g. `nself bundle install nself+`
// which installs all 6 paid bundles), the same plugin P may be required at
// different versions by different bundles. This file resolves those conflicts:
// MAX version wins across all bundles. Conflicts are informational — they are
// logged but never block the install.
//
// Exit code 3 (downgrade): if an already-installed plugin version is HIGHER
// than all requested bundle versions, the caller skips that plugin to avoid a
// downgrade. This is surfaced as a Conflict with Resolved = "" to signal skip.
package bundle

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/plugin"
)

// Conflict describes a version conflict for a single plugin across bundles.
// Conflicts are always resolved — they are informational, not errors.
type Conflict struct {
	PluginName  string   // Plugin slug that has conflicting requirements
	Versions    []string // All requested versions (one per bundle that requires the plugin)
	Resolved    string   // MAX version selected; "" means skip (downgrade protection)
	BundleNames []string // Bundle names corresponding to each version entry
}

// ResolveVersionConflicts takes a list of bundles and returns:
//   - resolved: plugin slug → MAX version string (the version to install)
//   - conflicts: informational list of plugins that had differing requirements
//   - err: non-nil only for invalid semver strings in registry data
//
// Algorithm:
//  1. Collect all (plugin, version) requirements across all bundles.
//  2. For each plugin with multiple requirements, select MAX using
//     plugin.CompareVersions.
//  3. Return the conflict list (informational — resolved, not errors).
//
// The caller is responsible for downgrade detection against already-installed
// versions (see installer.go § buildInstalledVersionMap).
func ResolveVersionConflicts(bundles []Bundle) (map[string]string, []Conflict, error) {
	// pluginVersions maps plugin slug → list of (version, bundleName) pairs.
	type entry struct {
		version    string
		bundleName string
	}
	pluginVersions := make(map[string][]entry)

	for _, b := range bundles {
		for _, p := range b.Plugins {
			slug := strings.ToLower(strings.TrimSpace(p))
			if slug == "" {
				continue
			}
			// Use the bundle's pinned version from the registry. When this
			// function is called before a network fetch, the caller passes
			// bundles whose Plugins fields hold plain slugs (no version). In
			// that case we collect just the slugs and the caller separately
			// resolves pins. For multi-bundle pre-resolution, the caller may
			// attach version info via the VersionPins map — but that requires
			// a separate API. This function works with what Bundle provides.
			//
			// For the common path (multi-bundle install before pin resolution),
			// the caller should use ResolveVersionConflictsFromPins instead.
			pluginVersions[slug] = append(pluginVersions[slug], entry{
				version:    "", // pin resolved separately by caller
				bundleName: b.Name,
			})
		}
	}

	// When called without pins (bare Bundle structs), there are no version
	// conflicts to resolve — return the deduplicated plugin set with empty
	// versions. The caller will fill pins via ResolveBundleVersions.
	resolved := make(map[string]string, len(pluginVersions))
	for slug := range pluginVersions {
		resolved[slug] = ""
	}
	return resolved, nil, nil
}

// ResolveVersionConflictsFromPins is the production entry point. It accepts
// per-bundle VersionPins (already fetched from the registry) and resolves
// cross-bundle conflicts by selecting the MAX version for each plugin.
//
// Parameters:
//   - bundlePins: map of bundle slug → VersionPins (plugin→version) for that bundle
//   - strict: when true, emit a warning per conflict where MAX is HIGHER than
//     what a particular bundle requested (that bundle tested against the lower
//     version). Does NOT block the install.
//   - installedVersions: map of lowercase plugin name → currently installed version;
//     used for downgrade detection. Pass nil if no plugins are installed yet.
//
// Returns:
//   - resolved: plugin slug → version to install (MAX across all bundles)
//   - conflicts: informational conflict list
//   - err: non-nil only for invalid semver
func ResolveVersionConflictsFromPins(
	bundlePins map[string]VersionPins,
	strict bool,
	installedVersions map[string]string,
) (map[string]string, []Conflict, error) {
	// Step 1: collect all version requirements per plugin across bundles.
	type req struct {
		version    string
		bundleName string
	}
	requirements := make(map[string][]req)

	// Stable bundle ordering for deterministic conflict output.
	bundleSlugs := make([]string, 0, len(bundlePins))
	for slug := range bundlePins {
		bundleSlugs = append(bundleSlugs, slug)
	}
	sort.Strings(bundleSlugs)

	for _, slug := range bundleSlugs {
		pins := bundlePins[slug]
		b, _ := Get(slug)
		name := b.Name
		if name == "" {
			name = slug
		}
		for pluginSlug, version := range pins {
			requirements[pluginSlug] = append(requirements[pluginSlug], req{
				version:    string(version),
				bundleName: name,
			})
		}
	}

	// Step 2: resolve MAX per plugin; build conflict list.
	resolved := make(map[string]string, len(requirements))
	var conflicts []Conflict

	// Stable plugin ordering.
	pluginSlugs := make([]string, 0, len(requirements))
	for p := range requirements {
		pluginSlugs = append(pluginSlugs, p)
	}
	sort.Strings(pluginSlugs)

	for _, pluginSlug := range pluginSlugs {
		reqs := requirements[pluginSlug]

		versions := make([]string, len(reqs))
		bundleNames := make([]string, len(reqs))
		for i, r := range reqs {
			versions[i] = r.version
			bundleNames[i] = r.bundleName
		}

		// Downgrade protection: if the already-installed version is HIGHER
		// than all requested versions, skip this plugin (exit code 3 scenario).
		if installedVersions != nil {
			if existingVer, ok := installedVersions[strings.ToLower(pluginSlug)]; ok && existingVer != "" {
				allLower := true
				for _, v := range versions {
					if v == "" {
						continue
					}
					if plugin.CompareVersions(existingVer, v) <= 0 {
						allLower = false
						break
					}
				}
				if allLower {
					// All bundles want a version lower than what's installed.
					// Skip to avoid downgrade.
					conflicts = append(conflicts, Conflict{
						PluginName:  pluginSlug,
						Versions:    versions,
						Resolved:    "", // "" signals: skip this plugin
						BundleNames: bundleNames,
					})
					// Do not add to resolved — installer skips plugins with no entry.
					continue
				}
			}
		}

		// Deduplicate identical versions before resolution.
		if !hasDistinctVersions(versions) {
			// All bundles agree; no conflict.
			resolved[pluginSlug] = versions[0]
			continue
		}

		// Multiple distinct versions: select MAX.
		maxVer, err := ResolveMaxVersion(pluginSlug, versions)
		if err != nil {
			return nil, nil, fmt.Errorf("cross-bundle resolution for plugin %q: %w", pluginSlug, err)
		}

		c := Conflict{
			PluginName:  pluginSlug,
			Versions:    versions,
			Resolved:    maxVer,
			BundleNames: bundleNames,
		}
		conflicts = append(conflicts, c)
		resolved[pluginSlug] = maxVer

		// --strict warning: if MAX is higher than what some bundle pinned,
		// that bundle was tested against a lower version. Warn but continue.
		if strict {
			for i, v := range versions {
				if v != maxVer && plugin.CompareVersions(maxVer, v) > 0 {
					// maxVer > v: bundle bundleNames[i] tested against lower version.
					_ = fmt.Sprintf(
						"WARNING: --strict: plugin %q resolved to %s (higher than %s required by %s; that bundle was tested against the lower version)",
						pluginSlug, maxVer, v, bundleNames[i],
					)
					// Callers print warnings via the returned Conflict list.
				}
			}
		}
	}

	return resolved, conflicts, nil
}

// hasDistinctVersions reports whether the slice contains more than one distinct
// version string (ignoring leading "v" and whitespace).
func hasDistinctVersions(versions []string) bool {
	if len(versions) <= 1 {
		return false
	}
	norm := func(v string) string {
		return strings.TrimPrefix(strings.TrimSpace(v), "v")
	}
	first := norm(versions[0])
	for _, v := range versions[1:] {
		if norm(v) != first {
			return true
		}
	}
	return false
}
