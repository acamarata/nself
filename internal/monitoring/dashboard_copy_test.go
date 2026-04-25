package monitoring

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectDashboards_noPlugins(t *testing.T) {
	selected := SelectDashboards(nil)
	if len(selected) != 1 {
		t.Fatalf("expected 1 (overview only), got %d: %v", len(selected), selected)
	}
	if selected[0] != "nself-overview.json" {
		t.Fatalf("expected nself-overview.json, got %s", selected[0])
	}
}

func TestSelectDashboards_clawPlugin(t *testing.T) {
	selected := SelectDashboards([]string{"claw"})
	if !contains(selected, "nclaw-bundle.json") {
		t.Fatalf("expected nclaw-bundle.json in %v", selected)
	}
}

func TestSelectDashboards_mediaProcessingPlugin(t *testing.T) {
	selected := SelectDashboards([]string{"media-processing"})
	if !contains(selected, "ntv-bundle.json") {
		t.Fatalf("expected ntv-bundle.json in %v", selected)
	}
}

func TestSelectDashboards_chatPlugin(t *testing.T) {
	selected := SelectDashboards([]string{"chat"})
	if !contains(selected, "nchat-bundle.json") {
		t.Fatalf("expected nchat-bundle.json in %v", selected)
	}
}

func TestSelectDashboards_socialPlugin(t *testing.T) {
	selected := SelectDashboards([]string{"social"})
	if !contains(selected, "nfamily-bundle.json") {
		t.Fatalf("expected nfamily-bundle.json in %v", selected)
	}
}

func TestSelectDashboards_unrelatedPlugin(t *testing.T) {
	selected := SelectDashboards([]string{"stripe", "webhooks"})
	if len(selected) != 1 {
		t.Fatalf("expected 1 (overview only), got %d: %v", len(selected), selected)
	}
}

func TestSelectDashboards_allBundles(t *testing.T) {
	selected := SelectDashboards([]string{"claw", "ai", "chat", "media-processing", "social"})
	expected := []string{
		"nself-overview.json",
		"nclaw-bundle.json",
		"ntv-bundle.json",
		"nchat-bundle.json",
		"nfamily-bundle.json",
	}
	for _, e := range expected {
		if !contains(selected, e) {
			t.Errorf("expected %s in selected %v", e, selected)
		}
	}
}

func TestCopyDashboards_copiesOverviewAlways(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Write stub dashboard files.
	writeStub(t, sourceDir, "nself-overview.json")
	writeStub(t, sourceDir, "nclaw-bundle.json")

	copied, err := CopyDashboards(sourceDir, destDir, nil)
	if err != nil {
		t.Fatalf("CopyDashboards error: %v", err)
	}
	if !contains(copied, "nself-overview.json") {
		t.Fatalf("overview not copied: %v", copied)
	}
	if _, err := os.Stat(filepath.Join(destDir, "nself-overview.json")); err != nil {
		t.Fatalf("overview not present in destDir: %v", err)
	}
	if contains(copied, "nclaw-bundle.json") {
		t.Fatalf("nclaw-bundle.json should NOT be copied without claw plugin")
	}
}

func TestCopyDashboards_copiesBundleWhenPluginInstalled(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	writeStub(t, sourceDir, "nself-overview.json")
	writeStub(t, sourceDir, "nclaw-bundle.json")

	copied, err := CopyDashboards(sourceDir, destDir, []string{"ai"})
	if err != nil {
		t.Fatalf("CopyDashboards error: %v", err)
	}
	if !contains(copied, "nclaw-bundle.json") {
		t.Fatalf("expected nclaw-bundle.json copied for ai plugin: %v", copied)
	}
}

// helpers

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func writeStub(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(`{"title":"stub"}`), 0644); err != nil {
		t.Fatalf("writeStub %s: %v", name, err)
	}
}
