package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFlattenExtractedPlugin_NestedFreePrefix is a regression test for
// P6-E3-W2-S1-T5 (2026-09-03): a tarball built by
// plugins/scripts/build-tarballs.sh's `tar -czf dist/<name>.tar.gz free/<name>/`
// embeds a "free/<name>/" prefix. extractTarGz faithfully reproduces that
// nesting on disk; flattenExtractedPlugin must promote the nested
// plugin.json (and everything alongside it) up to destDir.
func TestFlattenExtractedPlugin_NestedFreePrefix(t *testing.T) {
	destDir := t.TempDir()
	entries := []struct{ name, content string }{
		{"free/", ""},
		{"free/notifications/", ""},
		{"free/notifications/plugin.json", `{"name":"notifications","version":"1.0.0"}`},
		{"free/notifications/docker-compose.plugin.yml", "services:\n  notifications:\n    image: x\n"},
		{"free/notifications/lib/", ""},
		{"free/notifications/lib/helper.sh", "#!/bin/sh\necho ok"},
	}
	archivePath := writeTempArchive(t, buildTarGz(t, entries))
	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	// Sanity: confirm the bug is reproduced before the fix runs.
	if _, err := os.Stat(filepath.Join(destDir, "plugin.json")); err == nil {
		t.Fatalf("test fixture bug: plugin.json unexpectedly flat before flatten")
	}

	if err := flattenExtractedPlugin(destDir); err != nil {
		t.Fatalf("flattenExtractedPlugin: %v", err)
	}

	for _, rel := range []string{"plugin.json", "docker-compose.plugin.yml", "lib/helper.sh"} {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Errorf("expected flattened file %q: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(destDir, "free")); err == nil {
		t.Errorf("expected wrapper dir %q/free to be removed after promotion", destDir)
	}
}

// TestFlattenExtractedPlugin_AlreadyFlat verifies a correctly built archive
// (plugin.json directly at the root) is left untouched.
func TestFlattenExtractedPlugin_AlreadyFlat(t *testing.T) {
	destDir := t.TempDir()
	entries := []struct{ name, content string }{
		{"plugin.json", `{"name":"cron","version":"1.0.0"}`},
		{"docker-compose.plugin.yml", "services:\n  cron:\n    image: x\n"},
	}
	archivePath := writeTempArchive(t, buildTarGz(t, entries))
	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	if err := flattenExtractedPlugin(destDir); err != nil {
		t.Fatalf("flattenExtractedPlugin: %v", err)
	}
	for _, rel := range []string{"plugin.json", "docker-compose.plugin.yml"} {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Errorf("expected file %q to remain: %v", rel, err)
		}
	}
}

// TestFlattenExtractedPlugin_MultiLevelNesting verifies deeper wrapper
// chains (two levels) are also promoted, bounded by maxDepth.
func TestFlattenExtractedPlugin_MultiLevelNesting(t *testing.T) {
	destDir := t.TempDir()
	entries := []struct{ name, content string }{
		{"a/", ""},
		{"a/b/", ""},
		{"a/b/plugin.json", `{"name":"jobs","version":"1.0.0"}`},
		{"a/b/plugin.go", "package main"},
	}
	archivePath := writeTempArchive(t, buildTarGz(t, entries))
	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	if err := flattenExtractedPlugin(destDir); err != nil {
		t.Fatalf("flattenExtractedPlugin: %v", err)
	}
	for _, rel := range []string{"plugin.json", "plugin.go"} {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Errorf("expected flattened file %q: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(destDir, "a")); err == nil {
		t.Errorf("expected wrapper dir %q/a to be removed after promotion", destDir)
	}
}

// TestFlattenExtractedPlugin_MultipleTopEntries verifies an archive that
// legitimately has multiple top-level entries (not a single wrapper dir) is
// left untouched rather than mis-flattened.
func TestFlattenExtractedPlugin_MultipleTopEntries(t *testing.T) {
	destDir := t.TempDir()
	entries := []struct{ name, content string }{
		{"plugin.json", `{"name":"multi","version":"1.0.0"}`},
		{"README.md", "# multi"},
		{"lib/", ""},
		{"lib/helper.sh", "#!/bin/sh"},
	}
	archivePath := writeTempArchive(t, buildTarGz(t, entries))
	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	if err := flattenExtractedPlugin(destDir); err != nil {
		t.Fatalf("flattenExtractedPlugin: %v", err)
	}
	// Already flat (plugin.json at root) — flattenExtractedPlugin must no-op
	// immediately and never touch README.md/lib/.
	for _, rel := range []string{"plugin.json", "README.md", "lib/helper.sh"} {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Errorf("expected file %q to remain: %v", rel, err)
		}
	}
}
