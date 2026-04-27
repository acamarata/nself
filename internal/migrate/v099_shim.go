// v099_shim.go
//
// Home-level v0.9.9 → v1.x migration shim. The Bash-era CLI persisted state
// under ~/.nself/v099-state/ (cached license keys, plugin install logs,
// channel preference, generated SSH keys). The Go CLI moved to ~/.config/nself
// + ~/.nself/bin + per-project state directories. This shim detects the
// legacy home-level state, backs it up, and migrates the parts that still
// have a meaningful destination in v1.x.
//
// This is opt-in: it is NOT auto-run on first install. Operators run it
// explicitly via `nself migrate-from-v099`. The reason is two-fold:
//
//  1. The legacy state directory may contain operator-customised values that
//     should not be silently overwritten.
//  2. Some legacy values (license keys especially) need to be re-validated
//     against the current ping_api before being trusted, and we do not want
//     to do network calls on the upgrade path.
//
// Companion package `from_bash` handles per-project (project-dir) artifacts
// (.nself/config.sh, hand-written docker-compose.yml, etc.). This file only
// handles the home-level state. The two are independent and can be run in
// either order.

package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// V099HomeState describes one entity inside ~/.nself/v099-state/ that the
// shim recognises and knows how to migrate.
type V099HomeState struct {
	// LegacyPath is the path inside ~/.nself/v099-state/.
	LegacyPath string
	// Description is a human-readable label used in the report.
	Description string
	// NewPath is where the v1.x CLI expects the equivalent data, or empty
	// when there is no automatic destination (operator must re-enter
	// manually).
	NewPath string
	// Action is a short instruction printed alongside the entry.
	Action string
}

// V099DetectResult holds the outcome of DetectV099Home.
type V099DetectResult struct {
	// LegacyDir is the absolute path to ~/.nself/v099-state/ (whether or not
	// it exists).
	LegacyDir string
	// Found is the list of recognised legacy entries that exist on disk.
	Found []V099HomeState
	// Present is true if the legacy directory exists on disk at all.
	Present bool
}

// DetectV099Home returns a read-only summary of legacy v0.9.9 home-level
// state. It does not modify anything.
//
// homeDir is normally os.UserHomeDir(); tests inject a temp directory.
func DetectV099Home(homeDir string) V099DetectResult {
	legacyDir := filepath.Join(homeDir, ".nself", "v099-state")
	res := V099DetectResult{LegacyDir: legacyDir}

	if _, err := os.Stat(legacyDir); err != nil {
		// Directory missing — nothing to migrate.
		return res
	}
	res.Present = true

	// Recognised legacy entries. Each entry is checked for existence; only
	// present ones are reported.
	candidates := []V099HomeState{
		{
			LegacyPath:  "license.key",
			Description: "v0.9.9 cached license key",
			NewPath:     filepath.Join(homeDir, ".config", "nself", "license.json"),
			Action:      "Migrated. Re-validate with: nself license verify",
		},
		{
			LegacyPath:  "channel",
			Description: "v0.9.9 release channel preference",
			NewPath:     filepath.Join(homeDir, ".config", "nself", "channel.json"),
			Action:      "Migrated to JSON form. Override with: nself upgrade --channel <name>",
		},
		{
			LegacyPath:  "plugin-install.log",
			Description: "v0.9.9 plugin install log (informational only)",
			NewPath:     "",
			Action:      "Backed up only. v1.x manages plugin state per-project.",
		},
		{
			LegacyPath:  "ssh",
			Description: "v0.9.9 generated SSH keypair (used for remote deploys)",
			NewPath:     filepath.Join(homeDir, ".nself", "ssh"),
			Action:      "Moved to ~/.nself/ssh. Permissions preserved (0600).",
		},
	}

	for _, c := range candidates {
		full := filepath.Join(legacyDir, c.LegacyPath)
		if _, err := os.Stat(full); err == nil {
			res.Found = append(res.Found, c)
		}
	}
	return res
}

// V099MigrationResult summarises what RunV099Migration accomplished.
type V099MigrationResult struct {
	// BackupDir is the timestamped backup directory the shim wrote.
	BackupDir string
	// Migrated lists entries that were successfully copied/moved to v1.x
	// destinations.
	Migrated []string
	// Backed lists entries that were backed up but had no v1.x destination
	// (user must re-enter manually).
	Backed []string
	// Warnings are non-fatal issues encountered during migration.
	Warnings []string
}

// RunV099Migration performs the home-level shim migration. It is opt-in and
// idempotent: running it twice does nothing the second time (the legacy
// directory is moved aside on first success).
//
// homeDir is normally os.UserHomeDir(); tests inject a temp directory.
//
// On success the legacy ~/.nself/v099-state/ is renamed to
// ~/.nself/v099-state.migrated.<ts>/ so the operator still has the original
// data if they need it, but DetectV099Home no longer reports it.
func RunV099Migration(homeDir string) (*V099MigrationResult, error) {
	if homeDir == "" {
		return nil, errors.New("homeDir is required")
	}

	det := DetectV099Home(homeDir)
	if !det.Present {
		return &V099MigrationResult{
			Warnings: []string{"no v0.9.9 home state found at " + det.LegacyDir + " — nothing to do"},
		}, nil
	}

	out := &V099MigrationResult{}

	ts := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(homeDir, ".nself", "v099-state-backup", ts)
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("creating backup dir %s: %w", backupDir, err)
	}
	out.BackupDir = backupDir

	// Make sure the v1.x config dir exists.
	v1ConfigDir := filepath.Join(homeDir, ".config", "nself")
	if err := os.MkdirAll(v1ConfigDir, 0700); err != nil {
		return nil, fmt.Errorf("creating v1.x config dir: %w", err)
	}

	for _, entry := range det.Found {
		src := filepath.Join(det.LegacyDir, entry.LegacyPath)
		// 1. Always back up the raw file/dir.
		if err := copyTreeMode(src, filepath.Join(backupDir, entry.LegacyPath), 0600); err != nil {
			out.Warnings = append(out.Warnings,
				fmt.Sprintf("could not back up %s: %v", entry.LegacyPath, err))
			continue
		}

		// 2. Migrate to v1.x destination if one is declared.
		if entry.NewPath == "" {
			out.Backed = append(out.Backed, entry.LegacyPath)
			continue
		}

		// Specific per-entry conversion.
		switch entry.LegacyPath {
		case "license.key":
			data, err := os.ReadFile(src)
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("reading license.key: %v", err))
				continue
			}
			// v0.9.9 stored the bare license key. v1.x stores JSON. We
			// write a minimal JSON envelope marked unverified — operator
			// re-validates with `nself license verify`.
			payload := fmt.Sprintf(
				`{"key":%q,"verified":false,"migrated_from":"v099"}`,
				trimWhitespace(string(data)),
			)
			if err := os.WriteFile(entry.NewPath, []byte(payload), 0600); err != nil {
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("writing v1.x license.json: %v", err))
				continue
			}
			out.Migrated = append(out.Migrated, entry.LegacyPath)

		case "channel":
			data, err := os.ReadFile(src)
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("reading channel: %v", err))
				continue
			}
			ch := trimWhitespace(string(data))
			if ch != "stable" && ch != "canary" {
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("unknown legacy channel value %q, defaulting to stable", ch))
				ch = "stable"
			}
			payload := fmt.Sprintf(`{"channel":%q}`, ch)
			if err := os.WriteFile(entry.NewPath, []byte(payload), 0600); err != nil {
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("writing v1.x channel.json: %v", err))
				continue
			}
			out.Migrated = append(out.Migrated, entry.LegacyPath)

		case "ssh":
			if err := copyTreeMode(src, entry.NewPath, 0600); err != nil {
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("copying ssh dir: %v", err))
				continue
			}
			out.Migrated = append(out.Migrated, entry.LegacyPath)

		default:
			// No converter — back up only.
			out.Backed = append(out.Backed, entry.LegacyPath)
		}
	}

	// 3. Move the legacy directory aside so re-runs no longer detect it.
	migratedDir := filepath.Join(homeDir, ".nself", fmt.Sprintf("v099-state.migrated.%s", ts))
	if err := os.Rename(det.LegacyDir, migratedDir); err != nil {
		out.Warnings = append(out.Warnings,
			fmt.Sprintf("could not rename legacy dir aside: %v (you may delete %s manually)", err, det.LegacyDir))
	}

	return out, nil
}

// copyTreeMode copies a file or directory tree from src to dst. Files are
// written with the supplied mode (typically 0600 to match env-file
// conventions). Directories are written with 0700.
func copyTreeMode(src, dst string, fileMode os.FileMode) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0700); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyTreeMode(
				filepath.Join(src, e.Name()),
				filepath.Join(dst, e.Name()),
				fileMode,
			); err != nil {
				return err
			}
		}
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	return os.WriteFile(dst, data, fileMode)
}

// trimWhitespace trims spaces, tabs, CR, LF from both ends of s.
func trimWhitespace(s string) string {
	start, end := 0, len(s)
	for start < end {
		c := s[start]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		start++
	}
	for end > start {
		c := s[end-1]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		end--
	}
	return s[start:end]
}
