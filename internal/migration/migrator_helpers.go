package migration

// migrator_helpers.go — file-copy helpers, v0.9 plugin parsing and run summary.
//
// Purpose: copy files/directories for backup and rollback, parse legacy v0.9 plugin config for the migration warning, and print the final run summary, used throughout migrator.go, split out for file size.
// Inputs: source/destination paths for the copy helpers, or the legacy config for plugin parsing.
// Outputs: copied files/directories on disk, a plugin compatibility warning, or printed summary output.
// Constraints: pure move from migrator.go (CLI-R12 Batch E); no behaviour change.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/ui"
)

// dirSize returns the total size in bytes of all files under dir.
func dirSize(dir string) int64 {
	var total int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// copyFile copies the file at src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s → %s: %w", src, dst, err)
	}
	return out.Sync()
}

// copyDir recursively copies the directory at src to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

// copyPath copies src to dst, handling both files and directories.
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// pluginWarning parses the v0.9 .env backup for PLUGIN_* entries and prints
// a loud, ANSI-highlighted warning telling the user to re-install each plugin.
// It is display-only — it never executes any commands.
// Gracefully handles a missing or malformed .env (warns with empty plugin list).
func pluginWarning(projectDir, backupDir string) {
	// Try the backed-up .env first; fall back to the project dir.
	envPath := filepath.Join(backupDir, ".env")
	if _, err := os.Stat(envPath); err != nil {
		envPath = filepath.Join(projectDir, ".env")
	}

	plugins := parseV09Plugins(envPath)

	// Box-drawing characters for visibility.
	const (
		bold  = "\033[1m"
		reset = "\033[0m"
		red   = "\033[31m"
	)
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, bold+"┌─────────────────────────────────────────────────────────┐"+reset)
	_, _ = fmt.Fprintln(os.Stdout, bold+"│  PLUGINS MUST BE RE-INSTALLED                           │"+reset)
	_, _ = fmt.Fprintln(os.Stdout, bold+"│                                                          │"+reset)
	_, _ = fmt.Fprintln(os.Stdout, bold+"│  v0.9 plugin code is not compatible with v1.0.9 signed  │"+reset)
	_, _ = fmt.Fprintln(os.Stdout, bold+"│  bundles. Re-install each plugin after migration.        │"+reset)
	_, _ = fmt.Fprintln(os.Stdout, bold+"└─────────────────────────────────────────────────────────┘"+reset)
	_, _ = fmt.Fprintln(os.Stdout)

	_, _ = fmt.Fprintln(os.Stdout, "  Step 1: Re-enter your license key")
	_, _ = fmt.Fprintln(os.Stdout, bold+"    nself license set <your-key>"+reset)
	_, _ = fmt.Fprintln(os.Stdout)

	if len(plugins) > 0 {
		installCmd := "nself plugin install"
		for _, p := range plugins {
			installCmd += " " + p
		}
		_, _ = fmt.Fprintf(os.Stdout, "  Step 2: Re-install your %d plugin(s) (detected from v0.9 .env)\n", len(plugins))
		_, _ = fmt.Fprintln(os.Stdout, bold+"    "+installCmd+reset)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "  Step 2: Re-install your plugins")
		_, _ = fmt.Fprintln(os.Stdout, bold+"    nself plugin install <plugin1> <plugin2> ..."+reset)
		_, _ = fmt.Fprintln(os.Stdout, red+"    (Could not detect plugin list from .env — check manually)"+reset)
	}

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "  See: https://nself.org/docs/migrate/from-v0.9#step-3-re-install-plugins")
	_, _ = fmt.Fprintln(os.Stdout)
}

// parseV09Plugins reads a v0.9 .env file and returns the list of enabled plugins.
// v0.9 format: PLUGIN_<NAME>=true (case-insensitive value check).
// Returns nil (not an error) on any file read or parse failure.
func parseV09Plugins(envPath string) []string {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return nil
	}

	// v0.9 → v1.0.9 plugin name mapping (strips "nself-" prefix if present).
	// Keys are the v0.9 env var suffix (uppercase); values are v1 plugin names.
	nameMap := map[string]string{
		"AI":               "ai",
		"MUX":              "mux",
		"CLAW":             "claw",
		"VOICE":            "voice",
		"BROWSER":          "browser",
		"GOOGLE":           "google",
		"NOTIFY":           "notify",
		"CRON":             "cron",
		"CHAT":             "chat",
		"LIVEKIT":          "livekit",
		"RECORDING":        "recording",
		"MODERATION":       "moderation",
		"BOTS":             "bots",
		"REALTIME":         "realtime",
		"MEDIA_PROCESSING": "media-processing",
		"STREAMING":        "streaming",
		"EPG":              "epg",
		"TMDB":             "tmdb",
		"PODCAST":          "podcast",
		"SOCIAL":           "social",
	}

	var plugins []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(key, "PLUGIN_") {
			continue
		}
		if !strings.EqualFold(val, "true") && val != "1" {
			continue
		}
		suffix := strings.TrimPrefix(key, "PLUGIN_")
		if v1Name, ok := nameMap[suffix]; ok {
			plugins = append(plugins, v1Name)
		} else {
			// Unknown plugin — use lowercase suffix as best-guess v1 name.
			plugins = append(plugins, strings.ToLower(strings.ReplaceAll(suffix, "_", "-")))
		}
	}
	return plugins
}

// printRunSummary prints a human-readable migration summary.
func printRunSummary(manifest *BackupManifest, timestamp string) {
	items := []string{
		fmt.Sprintf("Backup:  .nself/backup/%s/ (%d file(s))", timestamp, len(manifest.Files)),
		"Config:  v2 docker-compose.yml generated",
		"Nginx:   nginx/sites/ layout applied",
		"SSL:     certificates regenerated",
	}
	ui.SummaryBox("Migration Complete", items)
	ui.Info("Next step: nself start")
	ui.Info("To undo:   nself migrate rollback")
}
