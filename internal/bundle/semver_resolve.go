package bundle

// Purpose: cross-bundle MAX-resolution (S2.T22) — resolving the newest required version of a shared plugin across installed bundles, plus the semver parsing/comparison it depends on.
// Inputs: plugin names and their candidate/required version strings.
// Outputs: the maximum resolved version, detected version conflicts, and semver comparison results.
// Constraints: split out of version_resolver.go as a pure move (CLI-R12); no behavior change.

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Cross-bundle MAX-resolution (S2.T22)
// ---------------------------------------------------------------------------

// ResolveMaxVersion returns the maximum (newest) semver string from versions.
// All versions must be valid semver (with or without a leading "v" prefix).
// Returns an error if versions is empty or any entry is not valid semver.
// The returned string preserves the "v" prefix of the winning candidate.
//
// Example:
//
//	ResolveMaxVersion("ai", []string{"1.0.0", "1.2.0", "1.1.0"}) → "1.2.0", nil
func ResolveMaxVersion(pluginName string, versions []string) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("plugin %q: no version requirements provided", pluginName)
	}
	max := versions[0]
	for _, v := range versions[1:] {
		cmp, err := compareSemver(v, max)
		if err != nil {
			return "", fmt.Errorf("plugin %q: invalid semver %q: %w", pluginName, v, err)
		}
		if cmp > 0 {
			max = v
		}
	}
	// Validate the initial candidate too (catches bad first entry).
	if _, err := parseSemver(versions[0]); err != nil {
		return "", fmt.Errorf("plugin %q: invalid semver %q: %w", pluginName, versions[0], err)
	}
	return max, nil
}

// ResolveConflicts takes a map of plugin slug → list of version requirements
// from different bundles and returns the resolved map of plugin slug → max
// version. If strict is true, any plugin that has more than one distinct
// version requirement returns an error instead of resolving to the max.
//
// A "conflict" in strict mode means two or more bundles require different
// versions of the same plugin. When strict is false (the default), conflicts
// are silently resolved by taking the highest (MAX) version.
func ResolveConflicts(requirements map[string][]string, strict bool) (map[string]string, error) {
	resolved := make(map[string]string, len(requirements))
	for plugin, vers := range requirements {
		if len(vers) == 0 {
			return nil, fmt.Errorf("plugin %q: empty version list", plugin)
		}
		if strict && hasConflict(vers) {
			return nil, fmt.Errorf(
				"plugin %q has conflicting version requirements across bundles: %s",
				plugin, strings.Join(vers, ", "),
			)
		}
		max, err := ResolveMaxVersion(plugin, vers)
		if err != nil {
			return nil, err
		}
		resolved[plugin] = max
	}
	return resolved, nil
}

// hasConflict reports whether a slice contains more than one distinct version.
func hasConflict(versions []string) bool {
	if len(versions) <= 1 {
		return false
	}
	first := strings.TrimPrefix(strings.TrimSpace(versions[0]), "v")
	for _, v := range versions[1:] {
		if strings.TrimPrefix(strings.TrimSpace(v), "v") != first {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Minimal semver helpers (no external dependency)
// ---------------------------------------------------------------------------

// semver holds the parsed components of a version string.
type semver struct {
	major, minor, patch int
	pre                 string // pre-release label (e.g. "alpha.1", "beta.2")
	raw                 string // original string
}

// parseSemver parses a semver string with an optional leading "v".
// Supports MAJOR.MINOR.PATCH, MAJOR.MINOR, and MAJOR forms.
// Pre-release labels after "-" are preserved for string comparison only;
// a version with a pre-release label sorts lower than the same base version
// without one (standard semver rule).
func parseSemver(s string) (semver, error) {
	raw := s
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return semver{}, fmt.Errorf("empty version string")
	}

	// Split off pre-release.
	pre := ""
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		pre = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.SplitN(s, ".", 3)
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, fmt.Errorf("non-numeric component %q in %q", p, raw)
		}
		nums[i] = n
	}
	return semver{
		major: nums[0],
		minor: nums[1],
		patch: nums[2],
		pre:   pre,
		raw:   raw,
	}, nil
}

// compareSemver returns:
//
//	> 0 if a > b
//	  0 if a == b
//	< 0 if a < b
func compareSemver(a, b string) (int, error) {
	pa, err := parseSemver(a)
	if err != nil {
		return 0, err
	}
	pb, err := parseSemver(b)
	if err != nil {
		return 0, err
	}

	if d := pa.major - pb.major; d != 0 {
		return d, nil
	}
	if d := pa.minor - pb.minor; d != 0 {
		return d, nil
	}
	if d := pa.patch - pb.patch; d != 0 {
		return d, nil
	}

	// Pre-release: no pre-release > pre-release (standard semver rule).
	switch {
	case pa.pre == "" && pb.pre == "":
		return 0, nil
	case pa.pre == "" && pb.pre != "":
		return 1, nil
	case pa.pre != "" && pb.pre == "":
		return -1, nil
	default:
		return strings.Compare(pa.pre, pb.pre), nil
	}
}
