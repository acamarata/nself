// Package plugin — manager_error_paths_test.go
// S88b.T01 coverage extension: error paths for plugin manager operations.
// These tests cover install/remove/update/list error branches not covered by
// the existing happy-path tests in plugin_test.go.
// Note: TestListInstalled_EmptyDir, TestListInstalled_NonExistentDir, and
// TestListInstalled_MultiplePlugins are already in plugin_test.go — omitted here.
package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

// TestListInstalled_SkipsFilesNotDirs verifies that ListInstalled ignores
// regular files in the plugin directory (only directories are plugins).
func TestListInstalled_SkipsFilesNotDirs(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a regular file — should be ignored.
	if err := os.WriteFile(filepath.Join(pluginDir, "not-a-plugin.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	plugins, err := ListInstalled(pluginDir)
	if err != nil {
		t.Fatalf("ListInstalled with file: unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("ListInstalled should skip regular files, got %d plugins", len(plugins))
	}
}

// TestListInstalled_SkipsDirsWithoutManifest verifies that directories without
// a valid plugin.json are silently skipped.
func TestListInstalled_SkipsDirsWithoutManifest(t *testing.T) {
	pluginDir := t.TempDir()

	// Directory without plugin.json.
	noManifestDir := filepath.Join(pluginDir, "not-a-plugin")
	if err := os.MkdirAll(noManifestDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	plugins, err := ListInstalled(pluginDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("directory without manifest should be skipped, got %d plugins", len(plugins))
	}
}

// TestListInstalled_WithValidManifest verifies that a directory with a valid
// plugin.json is included in the result.
func TestListInstalled_WithValidManifest(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a valid plugin directory.
	manifest := `{"name":"test-plugin","version":"1.0.0","description":"Test plugin","category":"utility","license":"MIT"}`
	pluginSubDir := filepath.Join(pluginDir, "test-plugin")
	if err := os.MkdirAll(pluginSubDir, 0755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	plugins, err := ListInstalled(pluginDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "test-plugin" {
		t.Errorf("plugin name: got %q, want test-plugin", plugins[0].Name)
	}
	if plugins[0].Version != "1.0.0" {
		t.Errorf("plugin version: got %q, want 1.0.0", plugins[0].Version)
	}
}

// TestListInstalled_TierInferredFromManifest verifies that when a manifest has no
// explicit Tier field but has RequiresLicense=true, the tier is inferred as "pro".
func TestListInstalled_TierInferredFromManifest(t *testing.T) {
	pluginDir := t.TempDir()

	// Manifest with requires_license=true but no explicit tier field.
	manifest := `{"name":"pro-plugin","version":"1.0.0","description":"Pro plugin","category":"ai","license":"source-available","requires_license":true}`
	subDir := filepath.Join(pluginDir, "pro-plugin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "plugin.json"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	plugins, err := ListInstalled(pluginDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	// Tier should be inferred as "pro" for license-required plugins.
	if plugins[0].Tier != "pro" {
		t.Errorf("tier inference: got %q, want pro for requires_license=true plugin", plugins[0].Tier)
	}
}

// TestListInstalled_CorruptManifest verifies that a directory with invalid JSON
// in plugin.json is silently skipped (not a fatal error).
func TestListInstalled_CorruptManifest(t *testing.T) {
	pluginDir := t.TempDir()

	subDir := filepath.Join(pluginDir, "corrupt-plugin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "plugin.json"), []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("write corrupt manifest: %v", err)
	}

	// Should not return an error — corrupt manifests are skipped.
	plugins, err := ListInstalled(pluginDir)
	if err != nil {
		t.Fatalf("ListInstalled with corrupt manifest: unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("corrupt manifest should be skipped, got %d plugins", len(plugins))
	}
}

// TestListInstalled_FreePluginDefaultTier verifies that a plugin with no
// requires_license field defaults to "free" tier.
func TestListInstalled_FreePluginDefaultTier(t *testing.T) {
	pluginDir := t.TempDir()

	// Manifest with explicit requires_license: false.
	manifest := `{"name":"free-plugin","version":"1.0.0","description":"Free plugin","category":"utility","license":"MIT","requires_license":false}`
	subDir := filepath.Join(pluginDir, "free-plugin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "plugin.json"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	plugins, err := ListInstalled(pluginDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Tier != "free" {
		t.Errorf("free plugin tier: got %q, want free", plugins[0].Tier)
	}
}

// TestListInstalled_MixedFreeAndPro verifies that a mix of free and pro
// plugins are correctly classified in a single listing call.
func TestListInstalled_MixedFreeAndPro(t *testing.T) {
	pluginDir := t.TempDir()

	// Free plugin.
	freeDir := filepath.Join(pluginDir, "webhooks")
	os.MkdirAll(freeDir, 0755)                          //nolint:errcheck
	os.WriteFile(filepath.Join(freeDir, "plugin.json"), //nolint:errcheck
		[]byte(`{"name":"webhooks","version":"1.0.0","description":"Webhooks","category":"communication","license":"MIT"}`), 0644)

	// Pro plugin.
	proDir := filepath.Join(pluginDir, "ai")
	os.MkdirAll(proDir, 0755)                          //nolint:errcheck
	os.WriteFile(filepath.Join(proDir, "plugin.json"), //nolint:errcheck
		[]byte(`{"name":"ai","version":"1.0.0","description":"AI","category":"ai","license":"source-available","requires_license":true}`), 0644)

	plugins, err := ListInstalled(pluginDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}

	byName := make(map[string]InstalledPluginInfo)
	for _, p := range plugins {
		byName[p.Name] = p
	}
	if byName["webhooks"].Tier != "free" {
		t.Errorf("webhooks tier: got %q, want free", byName["webhooks"].Tier)
	}
	if byName["ai"].Tier != "pro" {
		t.Errorf("ai tier: got %q, want pro", byName["ai"].Tier)
	}
}
