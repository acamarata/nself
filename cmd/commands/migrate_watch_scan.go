package commands

// Purpose: File-scanning, diffing, and small string/env helpers for
// `nself migration watch`, split out of migrate_watch.go (CLI-R12 Batch B
// mechanical file-size split). Everything here is pure/stateless given a
// watchState — the polling loop that drives them stays in
// migrate_watch.go's runMigrateWatch.
// Inputs: a watchState map, directory paths, extension filters, and raw
// file content/bytes.
// Outputs: updated snapshots, computed schema deltas, a generated AI
// prompt string, and small string/env parsing utilities.
// Constraints: pure move, no behavior change.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/migrationai"
)

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
