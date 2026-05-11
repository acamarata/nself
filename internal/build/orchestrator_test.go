// Package build — orchestrator_test.go: verifies the Step 9.7 wiring of
// AppendNSentryTargets (S12.T01) + WriteLokiConfigs (S9.T08) that
// orchestrator.go runs when MONITORING_ENABLED=true.
//
// These tests exercise the exact sequence of calls the orchestrator makes
// without booting the full Build() pipeline (config.Load + SSL + nginx +
// docker-compose generation all require a real project scaffold and are
// covered by their own packages). The goal is to prove:
//
//  1. The wiring writes both prometheus.yml AND loki.yml + promtail.yml.
//  2. ɳSentry plugin targets are merged into prometheus.yml when installed.
//  3. Running the same sequence twice yields byte-identical files
//     (idempotency — Docker bind-mount volume hashes don't churn).
package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/compose/monitoring"
)

// runMonitoringWiring executes the same sequence Step 9.7 runs:
//   - AppendNSentryTargets onto a fresh monitoring.Defaults()
//   - RenderPrometheusYAML → atomicWrite to monitoring/prometheus.yml
//   - WriteLokiConfigs → monitoring/loki.yml + promtail.yml
//
// Returns the rendered prometheus.yml bytes for assertion.
func runMonitoringWiring(t *testing.T, workdir, pluginDir, projectName string) []byte {
	t.Helper()

	promCfg := monitoring.Defaults()
	AppendNSentryTargets(promCfg, pluginDir)
	promYAML, err := monitoring.RenderPrometheusYAML(promCfg)
	if err != nil {
		t.Fatalf("render prometheus yaml: %v", err)
	}
	monDir := filepath.Join(workdir, "monitoring")
	if err := os.MkdirAll(monDir, 0o755); err != nil {
		t.Fatalf("mkdir monitoring: %v", err)
	}
	if err := atomicWrite(filepath.Join(monDir, "prometheus.yml"), promYAML, 0o644); err != nil {
		t.Fatalf("atomic write prometheus: %v", err)
	}

	if _, err := WriteLokiConfigs(workdir, LokiBuildOptions{ProjectName: projectName}); err != nil {
		t.Fatalf("write loki configs: %v", err)
	}
	return promYAML
}

// TestMonitoringWiring_WithNSentryPlugins verifies that when ɳSentry plugins
// are installed, the wiring writes prometheus.yml + loki.yml + promtail.yml,
// and prometheus.yml contains stanzas for each installed plugin.
func TestMonitoringWiring_WithNSentryPlugins(t *testing.T) {
	workdir := t.TempDir()
	pluginDir := makePluginDir(t,
		"nself-uptime-monitor",
		"nself-status-page",
		"nself-rum",
	)

	promYAML := runMonitoringWiring(t, workdir, pluginDir, "wire-test")

	// All three files must exist.
	for _, name := range []string{"prometheus.yml", "loki.yml", "promtail.yml"} {
		p := filepath.Join(workdir, "monitoring", name)
		st, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", p, err)
			continue
		}
		if st.Size() == 0 {
			t.Errorf("file %s is empty", p)
		}
	}

	// Prometheus YAML must reference the three installed ɳSentry plugins.
	want := []string{
		"nsentry-uptime-monitor",
		"nsentry-status-page",
		"nsentry-rum",
		"bundle: nsentry",
	}
	for _, w := range want {
		if !strings.Contains(string(promYAML), w) {
			t.Errorf("prometheus.yml missing %q\n%s", w, promYAML)
		}
	}

	// A NOT-installed ɳSentry plugin must not appear.
	if strings.Contains(string(promYAML), "nsentry-alert-router") {
		t.Errorf("prometheus.yml should not contain nsentry-alert-router (not installed)")
	}
}

// TestMonitoringWiring_NoNSentryPlugins verifies that with zero ɳSentry plugins
// installed, the wiring still writes prometheus.yml (with builtin targets only)
// + loki.yml + promtail.yml.
func TestMonitoringWiring_NoNSentryPlugins(t *testing.T) {
	workdir := t.TempDir()
	pluginDir := makePluginDir(t /* none */)

	promYAML := runMonitoringWiring(t, workdir, pluginDir, "no-nsentry")

	// Prometheus must contain builtin targets (e.g. prometheus, node-exporter).
	if !strings.Contains(string(promYAML), "job_name: prometheus") {
		t.Errorf("prometheus.yml missing builtin prometheus target\n%s", promYAML)
	}
	if !strings.Contains(string(promYAML), "job_name: node") {
		t.Errorf("prometheus.yml missing builtin node target")
	}
	// And must NOT contain any nsentry- prefix.
	if strings.Contains(string(promYAML), "nsentry-") {
		t.Errorf("prometheus.yml should not contain nsentry- targets when no plugins installed")
	}

	// Loki + promtail still written.
	for _, name := range []string{"loki.yml", "promtail.yml"} {
		if _, err := os.Stat(filepath.Join(workdir, "monitoring", name)); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}
}

// TestMonitoringWiring_Idempotent verifies that running the wiring twice on
// the same workdir yields byte-identical files for all three outputs.
func TestMonitoringWiring_Idempotent(t *testing.T) {
	workdir := t.TempDir()
	pluginDir := makePluginDir(t,
		"nself-uptime-monitor",
		"nself-incident-mgmt",
	)

	// First pass.
	runMonitoringWiring(t, workdir, pluginDir, "idempotent")
	prom1 := mustRead(t, filepath.Join(workdir, "monitoring", "prometheus.yml"))
	loki1 := mustRead(t, filepath.Join(workdir, "monitoring", "loki.yml"))
	promt1 := mustRead(t, filepath.Join(workdir, "monitoring", "promtail.yml"))

	// Second pass — same inputs, no changes expected.
	runMonitoringWiring(t, workdir, pluginDir, "idempotent")
	prom2 := mustRead(t, filepath.Join(workdir, "monitoring", "prometheus.yml"))
	loki2 := mustRead(t, filepath.Join(workdir, "monitoring", "loki.yml"))
	promt2 := mustRead(t, filepath.Join(workdir, "monitoring", "promtail.yml"))

	if !bytes.Equal(prom1, prom2) {
		t.Errorf("prometheus.yml differs across rebuilds — not idempotent")
	}
	if !bytes.Equal(loki1, loki2) {
		t.Errorf("loki.yml differs across rebuilds — not idempotent")
	}
	if !bytes.Equal(promt1, promt2) {
		t.Errorf("promtail.yml differs across rebuilds — not idempotent")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}
