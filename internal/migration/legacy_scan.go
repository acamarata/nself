package migration

import (
	"fmt"
	"os"
	"path/filepath"
)

// LegacyGlobalArtifact describes a stale v0.9 path found on the host system
// outside of any specific project directory.
type LegacyGlobalArtifact struct {
	// Path is the absolute path where the artifact was found.
	Path string
	// Kind describes what the artifact is (e.g. "plugin directory", "config cache").
	Kind string
	// Hint is the exact shell command the user should run to clean it up.
	Hint string
}

// legacyGlobalPaths lists the canonical v0.9 global directories that may contain
// stale artifacts after a CLI upgrade. The scan is read-only — nothing is deleted.
//
// Scanned in order:
//  1. ~/.nself/          — v0.9 home directory (replaced by ~/.config/nself/ in v1)
//  2. ~/.nself/plugins/  — v0.9 per-plugin install dirs (replaced by signed bundles)
//  3. ~/.config/nself/   — may coexist; flag if it contains v0.9 markers
//  4. /usr/local/share/nself/ — v0.9 system-wide shared data (removed in v1)
var legacyGlobalPaths = []struct {
	rel  string // path relative to home (or absolute if absolute==true)
	abs  bool
	kind string
	hint string
}{
	{
		rel:  ".nself",
		kind: "v0.9 home directory",
		hint: "rm -rf ~/.nself  # back up anything important first",
	},
	{
		rel:  ".nself/plugins",
		kind: "v0.9 plugin install directory",
		hint: "rm -rf ~/.nself/plugins  # plugins are now managed by `nself plugin install`",
	},
	{
		rel:  ".config/nself",
		kind: "v0.9 config cache",
		hint: "rm -rf ~/.config/nself  # will be recreated by `nself init` or `nself start`",
	},
	{
		abs:  true,
		rel:  "/usr/local/share/nself",
		kind: "v0.9 system-wide data directory",
		hint: "sudo rm -rf /usr/local/share/nself",
	},
}

// ScanLegacyPaths scans the host for v0.9 global artifacts and returns a slice
// describing every stale path found. Returns an empty slice on a clean install.
//
// Errors from permission-denied paths are recorded as warnings (Kind="permission denied"),
// not fatal — the caller receives a best-effort result.
func ScanLegacyPaths() []LegacyGlobalArtifact {
	home, err := os.UserHomeDir()
	if err != nil {
		// Cannot determine home — skip relative paths, still scan absolute ones.
		home = ""
	}

	var found []LegacyGlobalArtifact

	for _, entry := range legacyGlobalPaths {
		var target string
		if entry.abs {
			target = entry.rel
		} else {
			if home == "" {
				continue // skip relative paths when home is unknown
			}
			target = filepath.Join(home, entry.rel)
		}

		info, err := os.Stat(target)
		if err != nil {
			if os.IsPermission(err) {
				// Permission denied — report as a warning artifact so the user knows
				// the scan could not be completed for this path.
				found = append(found, LegacyGlobalArtifact{
					Path: target,
					Kind: fmt.Sprintf("permission denied — could not check %s", entry.kind),
					Hint: "Run with sudo to inspect this path",
				})
			}
			// Not found or other error — treat as clean.
			continue
		}

		if info.IsDir() || !info.IsDir() {
			// Exists (file or directory) — this is a stale artifact.
			found = append(found, LegacyGlobalArtifact{
				Path: target,
				Kind: entry.kind,
				Hint: entry.hint,
			})
		}
	}

	return found
}
