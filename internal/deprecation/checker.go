// Package deprecation — G7: client-side plugin compatibility checker.
// CheckPluginCompat is called during nself start to warn when installed plugin
// versions are below or at a version that has deprecated endpoints.
package deprecation

import (
	"fmt"
	"io"
	"strings"
)

// Warning is a single compatibility warning emitted by CheckPluginCompat.
type Warning struct {
	Plugin      string
	InstalledVersion string
	Path        string
	DeprecatedIn string
	RemovedIn   string
	Replacement string
	Docs        string
}

// String returns the human-readable warning line.
func (w Warning) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[API WARNING] plugin '%s' v%s: endpoint '%s' was deprecated in v%s",
		w.Plugin, w.InstalledVersion, w.Path, w.DeprecatedIn)
	if w.RemovedIn != "" {
		fmt.Fprintf(&sb, " and will be removed in v%s", w.RemovedIn)
	}
	if w.Replacement != "" {
		fmt.Fprintf(&sb, " — use '%s' instead", w.Replacement)
	}
	if w.Docs != "" {
		fmt.Fprintf(&sb, ". Docs: %s", w.Docs)
	}
	return sb.String()
}

// InstalledPlugin is the caller-supplied record for an installed plugin.
type InstalledPlugin struct {
	Name    string
	Version string // SemVer string, e.g. "1.2.0"
}

// CheckPluginCompat compares each installed plugin's version against the registry.
// It returns warnings for every deprecated endpoint where the installed version is
// greater than or equal to the version where the deprecation was introduced.
//
// Version comparison is done lexicographically on the SemVer strings. This is
// accurate for well-formed SemVer within the same major version (which all nSelf
// plugins are expected to be during the v1.x lifecycle).
func CheckPluginCompat(installed []InstalledPlugin, reg *PluginRegistry) []Warning {
	var warnings []Warning

	for _, ip := range installed {
		if ip.Version == "" {
			continue
		}
		calEntries := reg.SunsetDate(ip.Name)
		for _, cal := range calEntries {
			if cal.DeprecatedIn == "" {
				continue
			}
			// Emit a warning if the installed version is >= deprecated_in.
			// A plugin at the exact deprecation version is already affected.
			if versionGTE(ip.Version, cal.DeprecatedIn) {
				warnings = append(warnings, Warning{
					Plugin:           ip.Name,
					InstalledVersion: ip.Version,
					Path:             cal.Path,
					DeprecatedIn:     cal.DeprecatedIn,
					RemovedIn:        cal.RemovedIn,
					Replacement:      cal.Replacement,
					Docs: fmt.Sprintf("https://docs.nself.org/api/migration/%s-%s",
						ip.Name, strings.ReplaceAll(strings.TrimPrefix(cal.Path, "/"), "/", "-")),
				})
			}
		}
	}

	return warnings
}

// PrintWarnings writes all warnings to w, one per line.
func PrintWarnings(w io.Writer, warnings []Warning) {
	for _, warn := range warnings {
		fmt.Fprintln(w, warn.String())
	}
}

// versionGTE returns true if a >= b for simple SemVer strings (lexicographic).
// Works correctly for same-major version comparisons (v1.x).
func versionGTE(a, b string) bool {
	// Strip optional leading 'v'
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.SplitN(a, ".", 3)
	bParts := strings.SplitN(b, ".", 3)

	// Pad to 3 parts
	for len(aParts) < 3 {
		aParts = append(aParts, "0")
	}
	for len(bParts) < 3 {
		bParts = append(bParts, "0")
	}

	for i := 0; i < 3; i++ {
		ai := parseVersionPart(aParts[i])
		bi := parseVersionPart(bParts[i])
		if ai > bi {
			return true
		}
		if ai < bi {
			return false
		}
	}
	return true // equal
}

// parseVersionPart converts a SemVer component string to int; returns 0 on error.
func parseVersionPart(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
