package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}
}

// TestLinkCLIBinary_PublishesWhereProxyLooks is the regression guard for the
// gap CLI-R11's first extraction exposed: Install extracted a plugin to
// ~/.nself/plugins/<name>/ while ProxyCommand only ever searched
// ~/.nself/plugins/bin/. A command-providing plugin installed cleanly and then
// could not be run — the user saw "unknown command" immediately after a
// successful install.
func TestLinkCLIBinary_PublishesWhereProxyLooks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	destDir := filepath.Join(home, ".nself", "plugins", "dogfood")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "nself-dogfood"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	m := &PluginManifest{PluginType: "cli", BinaryName: "nself-dogfood"}
	if err := linkCLIBinary(destDir, "dogfood", m); err != nil {
		t.Fatalf("linkCLIBinary: %v", err)
	}

	published := PublishedBinaryPath("nself-dogfood")
	info, err := os.Stat(published)
	if err != nil {
		t.Fatalf("binary was not published where ProxyCommand looks (%s): %v", published, err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("published binary is not executable: mode %v", info.Mode().Perm())
	}
}

// TestLinkCLIBinary_IgnoresServicePlugins keeps the change narrow: the vast
// majority of plugins are services and must not have anything published into
// the command lookup path.
func TestLinkCLIBinary_IgnoresServicePlugins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	destDir := filepath.Join(home, ".nself", "plugins", "backup")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "nself-backup"), []byte("x"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}

	m := &PluginManifest{PluginType: "service"}
	if err := linkCLIBinary(destDir, "backup", m); err != nil {
		t.Fatalf("linkCLIBinary: %v", err)
	}

	if _, err := os.Stat(PublishedBinaryPath("nself-backup")); err == nil {
		t.Error("a service plugin had a binary published into the command lookup path")
	}
}

// TestLinkCLIBinary_SourceOnlyIsNotAnError covers a plugin that declares a
// command but ships no built binary yet: nothing to publish is not a failure.
func TestLinkCLIBinary_SourceOnlyIsNotAnError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	destDir := filepath.Join(home, ".nself", "plugins", "dogfood")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeManifest(t, destDir, `{"name":"dogfood","pluginType":"cli"}`)

	if err := linkCLIBinary(destDir, "dogfood", &PluginManifest{PluginType: "cli"}); err != nil {
		t.Fatalf("a source-only plugin should not fail install: %v", err)
	}
}

// TestUnlinkCLIBinary_RemovesPublishedCommand proves removal is symmetric: the
// proxy must stop resolving a plugin the user just removed.
func TestUnlinkCLIBinary_RemovesPublishedCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	binDir := PluginBinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	published := PublishedBinaryPath("nself-dogfood")
	if err := os.WriteFile(published, []byte("x"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := unlinkCLIBinary("dogfood", &PluginManifest{PluginType: "cli"}); err != nil {
		t.Fatalf("unlinkCLIBinary: %v", err)
	}
	if _, err := os.Stat(published); err == nil {
		t.Error("published command survived plugin removal — the proxy would still run it")
	}
}

// TestUnlinkCLIBinary_MissingIsNotAnError keeps uninstall working for the
// plugin that never published anything.
func TestUnlinkCLIBinary_MissingIsNotAnError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if err := unlinkCLIBinary("never-installed", nil); err != nil {
		t.Fatalf("removing a plugin that published nothing should succeed: %v", err)
	}
}
