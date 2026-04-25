package commands

// migrate watch — AI-powered file watcher that generates SQL migration proposals
// when model files change.
//
// Usage:
//
//	nself migration watch
//	nself migration watch --paths ./src,./lib --debounce 500
//
// The watcher monitors TypeScript, Go, and Dart model files for changes. When a
// change is detected, it diffs the old and new AST, routes the delta to the AI
// plugin (via internal/migrationai), stages a candidate migration file, and prints
// a proposal summary. The developer reviews and runs 'nself db migrate up' to apply.
//
// NSELF_MIGRATION_AUTO_APPLY is always false. The watcher never applies migrations
// without explicit developer confirmation.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nself-org/cli/internal/migrationai"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var migrateWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch model files and propose SQL migrations on change",
	Long: `Watch TypeScript, Go, and Dart model files for schema changes.
When a field is added to a model, the AI plugin generates a candidate
ALTER TABLE ADD COLUMN migration and stages it in migrations/.

Additive-only guard: DROP COLUMN, DROP TABLE, and TRUNCATE are never
emitted. They appear as warning comments for manual review.

NSELF_MIGRATION_AUTO_APPLY is always false. Run 'nself db migrate up'
to apply a staged migration.

Examples:
  nself migration watch
  nself migration watch --paths ./src,./internal
  nself migration watch --debounce 1000 --extensions .ts,.go`,
	RunE: runMigrateWatch,
}

var (
	migrateWatchPaths      string
	migrateWatchExtensions string
	migrateWatchDebounceMs int
	migrateWatchAIEndpoint string
	migrateWatchMigDir     string
)

func init() {
	migrateWatchCmd.Flags().StringVar(&migrateWatchPaths, "paths",
		envOr("NSELF_MIGRATION_WATCH_PATHS", "./src,./lib,./internal"),
		"Colon-separated directories to watch (env: NSELF_MIGRATION_WATCH_PATHS)")
	migrateWatchCmd.Flags().StringVar(&migrateWatchExtensions, "extensions",
		envOr("NSELF_MIGRATION_WATCH_EXTENSIONS", ".ts,.go,.dart"),
		"Comma-separated file extensions to watch (env: NSELF_MIGRATION_WATCH_EXTENSIONS)")
	migrateWatchCmd.Flags().IntVar(&migrateWatchDebounceMs, "debounce",
		envOrInt("NSELF_MIGRATION_WATCH_DEBOUNCE_MS", 500),
		"Debounce delay in milliseconds (env: NSELF_MIGRATION_WATCH_DEBOUNCE_MS)")
	migrateWatchCmd.Flags().StringVar(&migrateWatchAIEndpoint, "ai-endpoint", "",
		"nSelf AI plugin endpoint (default: NSELF_AI_ENDPOINT or http://localhost:3001)")
	migrateWatchCmd.Flags().StringVar(&migrateWatchMigDir, "migrations-dir", "migrations",
		"Directory to write staged migration files")

	migrateCmd.AddCommand(migrateWatchCmd)
}

// fileSnapshot captures the parsed state of a model file at a point in time.
type fileSnapshot struct {
	goStructs    []migrationai.GoStruct
	tsInterfaces []migrationai.TSInterface
	dartClasses  []migrationai.DartClass
	modTime      time.Time
}

// watchState maps absolute file paths to their last-seen snapshot.
type watchState map[string]*fileSnapshot

func runMigrateWatch(cmd *cobra.Command, _ []string) error {
	// Build generator config.
	cfg := migrationai.DefaultConfig()
	cfg.MigrationsDir = migrateWatchMigDir
	cfg.AutoApply = false // always — hard rule

	if migrateWatchAIEndpoint != "" {
		cfg.AIEndpoint = migrateWatchAIEndpoint
	} else if ep := os.Getenv("NSELF_AI_ENDPOINT"); ep != "" {
		cfg.AIEndpoint = ep
	}

	if os.Getenv("NSELF_MIGRATION_AUTO_APPLY") == "true" {
		ui.Warn("NSELF_MIGRATION_AUTO_APPLY=true is ignored by 'migration watch'. Auto-apply is never enabled.")
	}

	// Parse watch paths and extensions.
	paths := splitTrimmed(migrateWatchPaths, ",")
	if len(paths) == 0 {
		paths = []string{"."}
	}
	exts := make(map[string]bool)
	for _, e := range splitTrimmed(migrateWatchExtensions, ",") {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		exts[e] = true
	}

	debounce := time.Duration(migrateWatchDebounceMs) * time.Millisecond
	if debounce < 100*time.Millisecond {
		debounce = 100 * time.Millisecond
	}

	gen := migrationai.NewGenerator(cfg)

	// Print startup banner within 200 ms of command invocation.
	var watchDirs []string
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		watchDirs = append(watchDirs, abs)
	}
	ui.Section("nSelf Migration Watcher")
	for _, d := range watchDirs {
		fmt.Printf("  Watching for model changes in %s\n", d)
	}
	fmt.Printf("  Extensions: %s\n", migrateWatchExtensions)
	fmt.Printf("  Debounce:   %d ms\n", migrateWatchDebounceMs)
	fmt.Printf("  AI:         %s\n\n", cfg.AIEndpoint)
	ui.Info("Press Ctrl+C to stop.")
	fmt.Println()

	// Seed initial snapshots so the first poll has a baseline.
	state := make(watchState)
	for _, d := range watchDirs {
		seedSnapshots(state, d, exts)
	}

	// Set up graceful shutdown.
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	// pending debounce: path → time of last detected change.
	pending := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			ui.Info("Migration watcher stopped.")
			return nil

		case now := <-ticker.C:
			// Scan all watched directories for file modifications.
			for _, d := range watchDirs {
				scanDir(d, exts, state, pending, now)
			}

			// Fire proposals for any path whose debounce window has elapsed.
			for path, changedAt := range pending {
				if now.Sub(changedAt) < debounce {
					continue
				}
				delete(pending, path)

				snap, ok := state[path]
				if !ok {
					continue
				}

				// Re-read current file content.
				content, err := os.ReadFile(path)
				if err != nil {
					ui.Warn(fmt.Sprintf("watch: cannot read %s: %v", path, err))
					continue
				}

				// Compute delta against old snapshot, then update snapshot.
				deltas := computeDeltas(snap, content, path)
				updateSnapshot(snap, content, path)

				if len(deltas) == 0 {
					continue
				}

				rel := relPath(path)
				ui.Info(fmt.Sprintf("Schema change detected in %s:", rel))
				fmt.Println(migrationai.DiffSummary(deltas))

				// Generate proposal via AI.
				prompt := buildWatchPrompt(rel, deltas)
				aiCtx, aiCancel := context.WithTimeout(ctx, 60*time.Second)
				spin := ui.NewSpinner("Generating migration proposal...")
				spin.Start()
				proposal, genErr := gen.Generate(aiCtx, prompt, deltas)
				spin.Stop()
				aiCancel()

				if genErr != nil {
					if lcErr, ok := genErr.(*migrationai.ErrLowConfidence); ok {
						ui.Warn(fmt.Sprintf("Proposal skipped: AI confidence %.2f < minimum %.2f",
							lcErr.Got, lcErr.Required))
					} else {
						ui.Warn(fmt.Sprintf("Migration generation failed: %v", genErr))
					}
					continue
				}

				ui.Success(fmt.Sprintf("Migration staged: %s (confidence: %.0f%%, model: %s)",
					proposal.FilePath, proposal.Confidence*100, proposal.AIModel))
				if proposal.WarningSQL != "" {
					ui.Warn("Blocked destructive statements (see WARNING comments in file).")
				}
				ui.Info("Review the file, then run: nself db migrate up")
				fmt.Println()
			}
		}
	}
}

// seedSnapshots walks dir and builds initial file snapshots (baseline for diffing).
func seedSnapshots(state watchState, dir string, exts map[string]bool) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !exts[filepath.Ext(path)] {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		snap := &fileSnapshot{modTime: info.ModTime()}
		parseIntoSnapshot(snap, content, path)
		state[path] = snap
		return nil
	})
}

// scanDir checks all watched files in dir for modification times newer than the snapshot.
func scanDir(dir string, exts map[string]bool, state watchState, pending map[string]time.Time, now time.Time) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !exts[filepath.Ext(path)] {
			return nil
		}
		snap, exists := state[path]
		if !exists {
			// New file: seed its snapshot as baseline (no delta yet).
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			snap = &fileSnapshot{modTime: info.ModTime()}
			parseIntoSnapshot(snap, content, path)
			state[path] = snap
			return nil
		}
		if info.ModTime().After(snap.modTime) {
			// File modified: record pending change time (debounce).
			if _, already := pending[path]; !already {
				pending[path] = now
			}
		}
		return nil
	})
}

// computeDeltas returns schema deltas between the old snapshot and the new file content.
func computeDeltas(snap *fileSnapshot, newContent []byte, path string) []migrationai.SchemaFieldDelta {
	var deltas []migrationai.SchemaFieldDelta
	src := string(newContent)
	ext := filepath.Ext(path)

	switch ext {
	case ".go":
		newStructs, err := migrationai.ParseGoStructs(src)
		if err != nil {
			return nil
		}
		deltas = migrationai.DiffGoStructs(snap.goStructs, newStructs)
	case ".ts":
		newIfaces := migrationai.ParseTSInterfaces(src)
		deltas = migrationai.DiffTSInterfaces(snap.tsInterfaces, newIfaces)
	case ".dart":
		newClasses := migrationai.ParseDartClasses(src)
		deltas = migrationai.DiffDartClasses(snap.dartClasses, newClasses)
	}
	return deltas
}

// updateSnapshot refreshes the snapshot to match the current file content and mod time.
func updateSnapshot(snap *fileSnapshot, newContent []byte, path string) {
	parseIntoSnapshot(snap, newContent, path)
	snap.modTime = time.Now()
}

// parseIntoSnapshot fills snap's parsed fields from content based on file extension.
func parseIntoSnapshot(snap *fileSnapshot, content []byte, path string) {
	src := string(content)
	switch filepath.Ext(path) {
	case ".go":
		structs, err := migrationai.ParseGoStructs(src)
		if err == nil {
			snap.goStructs = structs
		}
	case ".ts":
		snap.tsInterfaces = migrationai.ParseTSInterfaces(src)
	case ".dart":
		snap.dartClasses = migrationai.ParseDartClasses(src)
	}
}

// buildWatchPrompt builds the natural-language prompt sent to the AI generator.
func buildWatchPrompt(relPath string, deltas []migrationai.SchemaFieldDelta) string {
	var sb strings.Builder
	sb.WriteString("schema change in ")
	sb.WriteString(relPath)
	sb.WriteString(": ")
	sb.WriteString(migrationai.DiffSummary(deltas))
	return strings.TrimSpace(sb.String())
}

// relPath returns a relative path from the cwd if possible, otherwise the abs path.
func relPath(abs string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return abs
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil {
		return abs
	}
	return rel
}

// splitTrimmed splits s by sep, trims spaces, and removes empty entries.
func splitTrimmed(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envOrInt returns the integer value of the environment variable key, or fallback.
func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}
