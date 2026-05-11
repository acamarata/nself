// S12.T01 — unit tests for ɳSentry Prometheus scrape-target generation.
//
// Cases (per spec):
//
//  1. zero ɳSentry plugins installed   → no targets, no nsentry stanza
//  2. partial install (3 of 7)          → exactly 3 targets, correct ports/labels
//  3. full install (all 7)              → all 7 targets present, alphabetical
//  4. idempotent rebuild                → re-running AppendNSentryTargets does
//     not duplicate stanzas
package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/compose/monitoring"
)

// makePluginDir creates a temp directory with one plugin.json file per name in
// installed. The returned path is suitable to pass to NSentryScrapeTargets.
func makePluginDir(t *testing.T, installed ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range installed {
		pluginDir := filepath.Join(dir, name)
		if err := os.MkdirAll(pluginDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", pluginDir, err)
		}
		manifest := filepath.Join(pluginDir, "plugin.json")
		// Minimal valid JSON — NSentryScrapeTargets only checks file existence,
		// not contents. Tests for ListInstalled handle parse semantics.
		if err := os.WriteFile(manifest, []byte(`{"name":"`+name+`","version":"1.0.0"}`), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}
	return dir
}

// TestNSentryScrapeTargets_None: zero ɳSentry plugins installed → nil result.
func TestNSentryScrapeTargets_None(t *testing.T) {
	dir := makePluginDir(t) // empty
	got := NSentryScrapeTargets(dir)
	if got != nil {
		t.Errorf("zero installed: want nil, got %d targets: %#v", len(got), got)
	}

	// Also verify AppendNSentryTargets is a no-op when nothing is installed.
	cfg := &monitoring.PrometheusConfig{}
	if added := AppendNSentryTargets(cfg, dir); added != 0 {
		t.Errorf("AppendNSentryTargets on empty plugindir: want 0 added, got %d", added)
	}
	if len(cfg.Targets) != 0 {
		t.Errorf("cfg.Targets should remain empty: got %#v", cfg.Targets)
	}
}

// TestNSentryScrapeTargets_Partial: 3 of 7 installed → exactly 3 targets,
// correct ports + bundle/plugin labels.
func TestNSentryScrapeTargets_Partial(t *testing.T) {
	// Pick three non-adjacent plugins to exercise the full lookup path.
	dir := makePluginDir(t,
		"nself-uptime-monitor",    // port 3831
		"nself-incident-mgmt",     // port 3833
		"nself-synthetic-monitor", // port 3836
		// also add a non-ɳSentry plugin to confirm it's ignored
		"nself-stripe",
	)

	got := NSentryScrapeTargets(dir)
	if len(got) != 3 {
		t.Fatalf("partial install: want 3 targets, got %d: %#v", len(got), got)
	}

	// Build expectations keyed on slug.
	want := map[string]int{
		"uptime-monitor":    3831,
		"incident-mgmt":     3833,
		"synthetic-monitor": 3836,
	}
	for _, target := range got {
		if target.Path != "/metrics" {
			t.Errorf("target %s: want path /metrics, got %q", target.JobName, target.Path)
		}
		if target.Labels["bundle"] != "nsentry" {
			t.Errorf("target %s: want bundle=nsentry, got %q", target.JobName, target.Labels["bundle"])
		}
		slug := target.Labels["plugin"]
		if slug == "" {
			t.Errorf("target %s: missing plugin label", target.JobName)
			continue
		}
		expectedPort, ok := want[slug]
		if !ok {
			t.Errorf("unexpected target slug %q in partial result", slug)
			continue
		}
		if target.Port != expectedPort {
			t.Errorf("plugin %s: want port %d, got %d", slug, expectedPort, target.Port)
		}
		if target.JobName != "nsentry-"+slug {
			t.Errorf("plugin %s: want jobname nsentry-%s, got %q", slug, slug, target.JobName)
		}
	}
}

// TestNSentryScrapeTargets_Full: all 7 installed → 7 targets in alphabetical
// order with the canonical port mapping.
func TestNSentryScrapeTargets_Full(t *testing.T) {
	dir := makePluginDir(t,
		"nself-uptime-monitor",
		"nself-status-page",
		"nself-incident-mgmt",
		"nself-alert-router",
		"nself-slo-tracker",
		"nself-synthetic-monitor",
		"nself-rum",
	)

	got := NSentryScrapeTargets(dir)
	if len(got) != 7 {
		t.Fatalf("full install: want 7 targets, got %d", len(got))
	}

	// Expect alphabetical-by-jobname order.
	wantOrder := []struct {
		job  string
		port int
	}{
		{"nsentry-alert-router", 3834},
		{"nsentry-incident-mgmt", 3833},
		{"nsentry-rum", 3837},
		{"nsentry-slo-tracker", 3835},
		{"nsentry-status-page", 3832},
		{"nsentry-synthetic-monitor", 3836},
		{"nsentry-uptime-monitor", 3831},
	}
	for i, w := range wantOrder {
		if got[i].JobName != w.job {
			t.Errorf("targets[%d].JobName: want %q, got %q", i, w.job, got[i].JobName)
		}
		if got[i].Port != w.port {
			t.Errorf("targets[%d].Port (%s): want %d, got %d", i, w.job, w.port, got[i].Port)
		}
		if got[i].Labels["bundle"] != "nsentry" {
			t.Errorf("targets[%d] (%s): missing bundle=nsentry label", i, w.job)
		}
	}
}

// TestAppendNSentryTargets_Idempotent: calling AppendNSentryTargets twice on
// the same config produces identical output — no duplicate stanzas.
func TestAppendNSentryTargets_Idempotent(t *testing.T) {
	dir := makePluginDir(t,
		"nself-uptime-monitor",
		"nself-status-page",
	)

	cfg := monitoring.Defaults()
	baseLen := len(cfg.Targets)

	added1 := AppendNSentryTargets(cfg, dir)
	if added1 != 2 {
		t.Fatalf("first append: want 2 added, got %d", added1)
	}
	if len(cfg.Targets) != baseLen+2 {
		t.Fatalf("first append: want %d targets total, got %d", baseLen+2, len(cfg.Targets))
	}

	added2 := AppendNSentryTargets(cfg, dir)
	if added2 != 0 {
		t.Errorf("second append (idempotent): want 0 added, got %d", added2)
	}
	if len(cfg.Targets) != baseLen+2 {
		t.Errorf("second append must not duplicate: want %d total, got %d", baseLen+2, len(cfg.Targets))
	}

	// Render YAML twice and confirm output is byte-identical (full idempotency).
	yaml1, err := monitoring.RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render 1: %v", err)
	}
	// No mutation between renders.
	yaml2, err := monitoring.RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render 2: %v", err)
	}
	if string(yaml1) != string(yaml2) {
		t.Errorf("renders differ across calls — not idempotent")
	}
}
