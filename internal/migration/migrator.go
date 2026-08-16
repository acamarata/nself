package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/ui"
)

// BackupManifest records what was backed up during a migration.
type BackupManifest struct {
	Timestamp  string            `json:"timestamp"`
	ProjectDir string            `json:"project_dir"`
	Files      []BackupFileEntry `json:"files"`
}

// BackupFileEntry describes a single backed-up file or directory.
type BackupFileEntry struct {
	// Source is the path relative to the project directory.
	Source string `json:"source"`
	// Dest is the path relative to the backup directory.
	Dest string `json:"dest"`
}

// BackupInfo describes an available backup for listing.
type BackupInfo struct {
	Timestamp string
	Dir       string
	// Size is the total size of all files in the backup directory, in bytes.
	Size int64
}

// Run performs a full v1 → v2 migration.
// It is idempotent: running on an already-migrated project prints a message and exits cleanly.
func Run(ctx context.Context, projectDir string) error {
	// 1. Detect v1 artifacts — idempotency check
	artifacts := Detect(projectDir)
	if len(artifacts) == 0 {
		ui.Success("Already on v2 — nothing to migrate")
		return nil
	}

	fmt.Fprintln(os.Stdout, "\n"+ui.C(ui.Yellow, ui.IconWarning)+" "+ui.C(ui.Bold, fmt.Sprintf("Migrating %d v1 artifact(s) to v2...", len(artifacts)))+"\n")

	// 2. Stop all running containers gracefully
	ui.Section("Stopping containers")
	if err := composeDown(ctx, projectDir); err != nil {
		ui.Warn(fmt.Sprintf("docker compose down: %v (containers may not be running)", err))
	} else {
		ui.Success("Containers stopped")
	}

	// 3. Backup current state
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(projectDir, ".nself", "backup", timestamp)
	ui.Section(fmt.Sprintf("Creating backup at .nself/backup/%s", timestamp))
	manifest, err := createBackup(projectDir, backupDir)
	if err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}
	ui.Success(fmt.Sprintf("Backed up %d file(s)", len(manifest.Files)))

	// 4. Move v1 nginx configs to v2 layout (nginx/ → nginx/sites/)
	ui.Section("Migrating nginx layout")
	moved, err := migrateNginxLayout(projectDir)
	if err != nil {
		return fmt.Errorf("migrating nginx layout (run `nself migrate rollback` to restore): %w", err)
	}
	if moved > 0 {
		ui.Success(fmt.Sprintf("Moved %d nginx config(s) to nginx/sites/", moved))
	} else {
		ui.Info("Nginx layout already compatible")
	}

	// 5. Regenerate compose + nginx + SSL via build
	ui.Section("Regenerating v2 configuration")
	if _, err = build.Build(projectDir, build.BuildOptions{Force: true}); err != nil {
		return fmt.Errorf("regenerating v2 config (run `nself migrate rollback` to restore): %w", err)
	}
	ui.Success("v2 configuration generated")

	// 6. Print plugin re-install warning (S60-T05)
	// v0.9 plugin code is incompatible with v1 signed bundles — hard break.
	// Parse the backed-up .env for PLUGIN_* entries and surface the re-install chain.
	pluginWarning(projectDir, backupDir)

	// 7. Print summary
	printRunSummary(manifest, timestamp)
	return nil
}

// Rollback restores a project from a backup created by Run.
// If backupTimestamp is empty, the most recent backup is used.
func Rollback(ctx context.Context, projectDir, backupTimestamp string) error {
	backupsBase := filepath.Join(projectDir, ".nself", "backup")

	var backupDir string
	if backupTimestamp != "" {
		backupDir = filepath.Join(backupsBase, backupTimestamp)
		if _, err := os.Stat(backupDir); err != nil {
			return fmt.Errorf("backup %q not found", backupTimestamp)
		}
	} else {
		latest, err := latestBackupDir(backupsBase)
		if err != nil {
			return err
		}
		backupDir = latest
	}

	// Load and validate manifest
	manifestPath := filepath.Join(backupDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading backup manifest: %w", err)
	}
	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parsing backup manifest (file may be corrupt): %w", err)
	}

	ui.Section(fmt.Sprintf("Restoring from backup %s", filepath.Base(backupDir)))

	// Stop containers before restoring
	ui.Info("Stopping containers...")
	if err := composeDown(ctx, projectDir); err != nil {
		ui.Warn(fmt.Sprintf("docker compose down: %v (continuing)", err))
	}

	// Restore each file recorded in the manifest
	for _, entry := range manifest.Files {
		src := filepath.Join(backupDir, entry.Dest)
		dst := filepath.Join(projectDir, entry.Source)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("preparing restore path for %s: %w", entry.Source, err)
		}
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("restoring %s: %w", entry.Source, err)
		}
	}

	ui.Success(fmt.Sprintf("Restored %d file(s) from backup %s", len(manifest.Files), filepath.Base(backupDir)))
	fmt.Fprintln(os.Stdout)
	ui.Info("Rollback complete. Run `docker compose up -d` to start your v1 stack.")
	return nil
}

// ListBackups returns all available backups sorted newest-first.
// Returns an empty slice (not an error) when no backups exist.
func ListBackups(projectDir string) ([]BackupInfo, error) {
	backupsBase := filepath.Join(projectDir, ".nself", "backup")
	entries, err := os.ReadDir(backupsBase)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(backupsBase, e.Name())
		backups = append(backups, BackupInfo{
			Timestamp: e.Name(),
			Dir:       dir,
			Size:      dirSize(dir),
		})
	}

	// Timestamp format 20060102-150405 sorts lexicographically; newest-first = descending.
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp > backups[j].Timestamp
	})
	return backups, nil
}

// composeDown runs `docker compose down` in the given directory.
func composeDown(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "down")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// createBackup copies key project files to backupDir and writes a manifest.json.
func createBackup(projectDir, backupDir string) (*BackupManifest, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating backup directory: %w", err)
	}

	manifest := &BackupManifest{
		Timestamp:  filepath.Base(backupDir),
		ProjectDir: projectDir,
	}

	// Candidates to back up, relative to projectDir.
	candidates := []string{
		"docker-compose.yml",
		".env",
		"nginx",
		".nself/config",
	}
	// Include SSL cert directories if present.
	for _, sslRel := range []string{"ssl", ".nself/ssl"} {
		if _, err := os.Stat(filepath.Join(projectDir, sslRel)); err == nil {
			candidates = append(candidates, sslRel)
		}
	}

	for _, rel := range candidates {
		src := filepath.Join(projectDir, rel)
		info, err := os.Stat(src)
		if err != nil {
			continue // skip absent files/dirs
		}

		destPath := filepath.Join(backupDir, rel)
		if info.IsDir() {
			if err := copyDir(src, destPath); err != nil {
				return nil, fmt.Errorf("backing up %s: %w", rel, err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return nil, fmt.Errorf("creating backup subdir for %s: %w", rel, err)
			}
			if err := copyFile(src, destPath); err != nil {
				return nil, fmt.Errorf("backing up %s: %w", rel, err)
			}
		}
		manifest.Files = append(manifest.Files, BackupFileEntry{Source: rel, Dest: rel})
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "manifest.json"), data, 0o644); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	return manifest, nil
}

// migrateNginxLayout moves .conf files from nginx/ to nginx/sites/ (v1 → v2 layout).
// Returns the number of files moved. Safe to call on a v2 project (no-op).
func migrateNginxLayout(projectDir string) (int, error) {
	nginxDir := filepath.Join(projectDir, "nginx")
	info, err := os.Stat(nginxDir)
	if err != nil || !info.IsDir() {
		return 0, nil // no nginx directory — nothing to migrate
	}

	sitesDir := filepath.Join(nginxDir, "sites")
	if _, err := os.Stat(sitesDir); err == nil {
		return 0, nil // already has sites/ — v2 layout, nothing to do
	}

	if err := os.MkdirAll(sitesDir, 0o755); err != nil {
		return 0, fmt.Errorf("creating nginx/sites: %w", err)
	}

	entries, err := os.ReadDir(nginxDir)
	if err != nil {
		return 0, fmt.Errorf("reading nginx directory: %w", err)
	}

	moved := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".conf") || strings.HasSuffix(name, ".conf.template") {
			src := filepath.Join(nginxDir, name)
			dst := filepath.Join(sitesDir, name)
			if err := os.Rename(src, dst); err != nil {
				return moved, fmt.Errorf("moving %s to nginx/sites/: %w", name, err)
			}
			moved++
		}
	}
	return moved, nil
}

// latestBackupDir returns the path to the most recent backup directory.
func latestBackupDir(backupsBase string) (string, error) {
	entries, err := os.ReadDir(backupsBase)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no backups found: .nself/backup/ does not exist")
		}
		return "", fmt.Errorf("reading backup directory: %w", err)
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no backups found in .nself/backup/")
	}

	sort.Strings(dirs)
	return filepath.Join(backupsBase, dirs[len(dirs)-1]), nil
}

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
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer out.Close()

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
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, bold+"┌─────────────────────────────────────────────────────────┐"+reset)
	fmt.Fprintln(os.Stdout, bold+"│  PLUGINS MUST BE RE-INSTALLED                           │"+reset)
	fmt.Fprintln(os.Stdout, bold+"│                                                          │"+reset)
	fmt.Fprintln(os.Stdout, bold+"│  v0.9 plugin code is not compatible with v1.0.9 signed  │"+reset)
	fmt.Fprintln(os.Stdout, bold+"│  bundles. Re-install each plugin after migration.        │"+reset)
	fmt.Fprintln(os.Stdout, bold+"└─────────────────────────────────────────────────────────┘"+reset)
	fmt.Fprintln(os.Stdout)

	fmt.Fprintln(os.Stdout, "  Step 1: Re-enter your license key")
	fmt.Fprintln(os.Stdout, bold+"    nself license set <your-key>"+reset)
	fmt.Fprintln(os.Stdout)

	if len(plugins) > 0 {
		installCmd := "nself plugin install"
		for _, p := range plugins {
			installCmd += " " + p
		}
		fmt.Fprintln(os.Stdout, fmt.Sprintf("  Step 2: Re-install your %d plugin(s) (detected from v0.9 .env)", len(plugins)))
		fmt.Fprintln(os.Stdout, bold+"    "+installCmd+reset)
	} else {
		fmt.Fprintln(os.Stdout, "  Step 2: Re-install your plugins")
		fmt.Fprintln(os.Stdout, bold+"    nself plugin install <plugin1> <plugin2> ..."+reset)
		fmt.Fprintln(os.Stdout, red+"    (Could not detect plugin list from .env — check manually)"+reset)
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  See: https://nself.org/docs/migrate/from-v0.9#step-3-re-install-plugins")
	fmt.Fprintln(os.Stdout)
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
