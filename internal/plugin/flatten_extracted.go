package plugin

// Purpose: post-extraction layout normalization for a freshly installed
// plugin — corrects a tarball-authoring bug that nests the plugin's files
// one (or more) directory levels too deep. Split out of download.go for
// file size (engineering-standard <=300 lines, ASI Policy 3).
// Inputs: destDir, the plugin's extraction target (pluginDir/<name>/).
// Outputs: destDir left with plugin.json (and the rest of the plugin's
// files) directly at its root, or unchanged if no recognized wrapper
// layout is found.
// Constraints: called by installLocked (installer_locked.go) immediately
// after extractTarGz; must be idempotent-safe to call on an already-flat
// layout (no-ops via the plugin.json-at-root fast path).

import (
	"fmt"
	"os"
	"path/filepath"
)

// flattenExtractedPlugin corrects a tarball-authoring bug where the archive
// embeds its build-time source path (e.g. "free/<name>/...", from
// `tar -czf dist/<name>.tar.gz free/<name>/` in plugins/scripts/build-tarballs.sh)
// instead of packaging the plugin's own file tree at the archive root. When
// that happens, extractTarGz faithfully reproduces the nesting on disk —
// destDir/free/<name>/plugin.json instead of destDir/plugin.json — which
// breaks every consumer that expects a flat layout directly under destDir:
// internal/build/plugins.go's DiscoverPluginComposeFiles (build-time compose
// discovery) and internal/plugin/runtime_start.go's Start (`nself plugin
// start`). Confirmed as the root cause of 0/7 ntask free-plugin installs
// reaching a running container (P6-E3-W2-S1-T5, 2026-09-03).
//
// This is a defensive fix on the extraction side (belt-and-suspenders with
// any fix to the archive-building pipeline itself, which lives in a
// different repo — plugins/scripts/build-tarballs.sh — and is out of this
// ticket's scope): if plugin.json is not directly under destDir after
// extraction, walk down while there is exactly one child and it is a
// directory, and once a nested plugin.json is found, promote that
// directory's contents up to destDir and remove the now-empty wrapper
// chain. Bounded to a handful of levels so a malformed or unexpected
// archive layout is left alone rather than walked indefinitely.
func flattenExtractedPlugin(destDir string) error {
	if _, err := os.Stat(filepath.Join(destDir, "plugin.json")); err == nil {
		return nil // already flat — the common case for a correctly built archive
	}

	const maxDepth = 4
	cur := destDir
	for depth := 0; depth < maxDepth; depth++ {
		entries, err := os.ReadDir(cur)
		if err != nil {
			return fmt.Errorf("reading %s: %w", cur, err)
		}
		if len(entries) != 1 || !entries[0].IsDir() {
			// Not a single-directory wrapper — either already flat at a
			// different shape or a layout this heuristic doesn't recognize.
			// Leave it as extracted; downstream steps will surface a clear
			// "plugin.json not found" error rather than this function
			// guessing wrong and silently corrupting a valid layout.
			return nil
		}
		cur = filepath.Join(cur, entries[0].Name())
		if _, err := os.Stat(filepath.Join(cur, "plugin.json")); err == nil {
			return promoteExtractedContents(cur, destDir)
		}
	}
	return nil
}

// promoteExtractedContents moves every entry directly under src into dst,
// then removes the now-empty wrapper directory chain between dst
// (exclusive) and src (inclusive). src and dst are both under the same
// destDir tree from flattenExtractedPlugin, so the moves are same-filesystem
// renames.
func promoteExtractedContents(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	for _, e := range entries {
		oldPath := filepath.Join(src, e.Name())
		newPath := filepath.Join(dst, e.Name())
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("moving %s to %s: %w", oldPath, newPath, err)
		}
	}
	// Best-effort cleanup of the now-empty wrapper directories. A failure
	// here (e.g. a directory that turned out non-empty) is not fatal — the
	// plugin's files are already promoted to dst, which is what matters.
	for cur := src; cur != dst; {
		parent := filepath.Dir(cur)
		if err := os.Remove(cur); err != nil {
			break
		}
		cur = parent
	}
	return nil
}
