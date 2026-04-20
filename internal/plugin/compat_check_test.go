package plugin

import (
	"testing"
)

// TestCompatCheck_CleanState verifies that CompareVersions returns 0
// when plugin maxNselfVersion equals the current CLI version. (S58-T06)
func TestCompatCheck_CleanState(t *testing.T) {
	// Plugin max == CLI version: not incompatible.
	result := CompareVersions("1.0.9", "1.0.9")
	if result != 0 {
		t.Errorf("expected 0 (equal), got %d", result)
	}
}

// TestCompatCheck_IncompatiblePlugin verifies that CompareVersions correctly
// identifies when the CLI version exceeds a plugin's maxNselfVersion. (S58-T06)
func TestCompatCheck_IncompatiblePlugin(t *testing.T) {
	// CLI 1.0.9 > plugin maxNselfVersion 1.0.5: plugin is incompatible.
	result := CompareVersions("1.0.9", "1.0.5")
	if result <= 0 {
		t.Errorf("expected CLI > maxNselfVersion (positive), got %d", result)
	}
}

// TestCompatCheck_PostUpdateHookFires verifies that CheckEOLBlock returns nil
// for a non-EOL plugin (the hook should not block normal post-update flow). (S58-T06)
func TestCompatCheck_PostUpdateHookFires(t *testing.T) {
	// A stable plugin should never be blocked by CheckEOLBlock.
	// We test via the manifest-level helper used by the EOL matrix tests.
	m := &PluginManifest{PublishStatus: "stable"}
	err := checkEOLBlockFromManifest(m, false)
	if err != nil {
		t.Errorf("stable plugin should not be blocked by EOL check: %v", err)
	}
}

// TestFindPluginByName verifies the exported wrapper resolves a plugin. (S58-T06)
func TestFindPluginByName(t *testing.T) {
	reg := &Registry{
		Plugins: []PluginManifest{
			{Name: "ai", Version: "1.0.0", PublishStatus: "stable"},
			{Name: "cron", Version: "1.2.0", PublishStatus: "beta"},
		},
	}

	m, found := FindPluginByName(reg, "ai")
	if !found {
		t.Fatal("expected to find plugin 'ai'")
	}
	if m.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.Version)
	}

	_, found = FindPluginByName(reg, "nonexistent")
	if found {
		t.Error("expected not to find 'nonexistent' plugin")
	}
}
