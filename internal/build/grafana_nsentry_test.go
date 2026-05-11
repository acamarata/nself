// S12.T02 + S12.T04 — unit tests for ɳSentry Grafana auto-provisioning.
//
// Cases:
//
//  1. Zero ɳSentry plugins installed → all renderers return nil; writer
//     reports 0 files written, no monitoring/grafana/ tree created.
//  2. Full install (7 baseline) → datasource + dashboard-provisioning YAML
//     emitted; one dashboard JSON per installed plugin.
//  3. status-page dashboard contains the T04 text-panel pointing at port
//     3832 with the /status endpoint link.
//  4. Idempotent rewrite — running WriteNSentryGrafanaProvisioning twice on
//     the same plugin set produces byte-identical files.
//  5. Unknown plugin slug → RenderNSentryDashboard returns an error.
package build

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nsentryBaselineNames returns the 7 ɳSentry baseline plugin directory
// names matching nsentryPlugins in prometheus.go. Kept in sync with the
// canonical list so this test fails loudly if the baseline shape drifts.
func nsentryBaselineNames() []string {
	return []string{
		"nself-alert-router",
		"nself-incident-mgmt",
		"nself-rum",
		"nself-slo-tracker",
		"nself-status-page",
		"nself-synthetic-monitor",
		"nself-uptime-monitor",
	}
}

// installNSentryManifests creates a fresh plugin directory with one
// plugin.json stub per name in installed. Reused across tests.
func installNSentryManifests(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		pd := filepath.Join(dir, name)
		if err := os.MkdirAll(pd, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", pd, err)
		}
		manifest := filepath.Join(pd, "plugin.json")
		if err := os.WriteFile(manifest, []byte(`{"name":"`+name+`"}`), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}
	return dir
}

// TestNSentryGrafana_None: no plugins installed → no files emitted and
// renderers return nil. Verifies the "zero stanzas when nothing installed"
// invariant required by the spec.
func TestNSentryGrafana_None(t *testing.T) {
	pluginDir := installNSentryManifests(t)
	workdir := t.TempDir()

	if ds := RenderNSentryDatasourceYAML(pluginDir); ds != nil {
		t.Errorf("datasource renderer with zero plugins: want nil, got %d bytes", len(ds))
	}
	if dp := RenderNSentryDashboardProvisioning(pluginDir); dp != nil {
		t.Errorf("dashboard-provisioning renderer with zero plugins: want nil, got %d bytes", len(dp))
	}

	n, err := WriteNSentryGrafanaProvisioning(workdir, pluginDir)
	if err != nil {
		t.Fatalf("Write with zero installed: %v", err)
	}
	if n != 0 {
		t.Errorf("Write reported %d files written; want 0", n)
	}

	// Confirm no nsentry/ directory was created when nothing is installed.
	dashDir := filepath.Join(workdir, "monitoring", "grafana", "dashboards", "nsentry")
	if _, err := os.Stat(dashDir); !os.IsNotExist(err) {
		t.Errorf("dashboard dir should not exist when nothing installed; stat err = %v", err)
	}
}

// TestNSentryGrafana_FullInstall: 7 baseline plugins installed → 2 YAML
// files + 7 dashboard JSON files emitted, all parseable.
func TestNSentryGrafana_FullInstall(t *testing.T) {
	baseline := nsentryBaselineNames()
	pluginDir := installNSentryManifests(t, baseline...)
	workdir := t.TempDir()

	n, err := WriteNSentryGrafanaProvisioning(workdir, pluginDir)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	// 2 provisioning YAMLs + 7 dashboard JSONs = 9.
	if n != 9 {
		t.Errorf("file count: want 9, got %d", n)
	}

	// Datasource YAML present.
	dsPath := filepath.Join(workdir, "monitoring", "grafana", "provisioning", "datasources", "nsentry.yaml")
	dsBytes, err := os.ReadFile(dsPath)
	if err != nil {
		t.Fatalf("read datasource: %v", err)
	}
	if !strings.Contains(string(dsBytes), "uid: nsentry-prometheus") {
		t.Errorf("datasource YAML missing nsentry-prometheus uid:\n%s", dsBytes)
	}

	// Each baseline plugin has a dashboard with valid JSON + correct UID +
	// bundle="nsentry" label filter in queries.
	for _, name := range baseline {
		// strip nself- prefix to derive slug
		slug := strings.TrimPrefix(name, "nself-")
		path := filepath.Join(workdir, "monitoring", "grafana", "dashboards", "nsentry", slug+".json")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read dashboard %s: %v", slug, err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("dashboard %s invalid JSON: %v", slug, err)
			continue
		}
		if uid, _ := parsed["uid"].(string); uid != "nsentry-"+slug {
			t.Errorf("dashboard %s: uid = %q, want %q", slug, uid, "nsentry-"+slug)
		}
		// JSON encoding escapes `"` so the embedded PromQL contains
		// `bundle=\"nsentry\"` after marshaling. Match the escaped form.
		if !bytes.Contains(body, []byte(`bundle=\"nsentry\"`)) {
			t.Errorf("dashboard %s missing bundle=\"nsentry\" label filter", slug)
		}
	}
}

// TestNSentryGrafana_StatusPageT04: the status-page dashboard MUST include
// the T04 link panel pointing at port 3832 /status. This is the integration
// surface the spec calls out as a hard acceptance criterion for T04.
func TestNSentryGrafana_StatusPageT04(t *testing.T) {
	body, err := RenderNSentryDashboard("status-page")
	if err != nil {
		t.Fatalf("render status-page: %v", err)
	}
	if !bytes.Contains(body, []byte(":3832/status")) {
		t.Errorf("status-page dashboard missing :3832/status link\n%s", body)
	}
	if !bytes.Contains(body, []byte("\"type\": \"text\"")) {
		t.Errorf("status-page dashboard missing text panel\n%s", body)
	}
}

// TestNSentryGrafana_Idempotent: two consecutive writes produce
// byte-identical files so Docker bind-mount hashes don't churn.
func TestNSentryGrafana_Idempotent(t *testing.T) {
	pluginDir := installNSentryManifests(t, nsentryBaselineNames()...)
	workdir := t.TempDir()

	if _, err := WriteNSentryGrafanaProvisioning(workdir, pluginDir); err != nil {
		t.Fatalf("first write: %v", err)
	}
	first := snapshotDir(t, filepath.Join(workdir, "monitoring", "grafana"))
	if _, err := WriteNSentryGrafanaProvisioning(workdir, pluginDir); err != nil {
		t.Fatalf("second write: %v", err)
	}
	second := snapshotDir(t, filepath.Join(workdir, "monitoring", "grafana"))

	if len(first) != len(second) {
		t.Fatalf("file count diff across rebuilds: first=%d second=%d", len(first), len(second))
	}
	for path, body1 := range first {
		body2, ok := second[path]
		if !ok {
			t.Errorf("file %s missing in second write", path)
			continue
		}
		if !bytes.Equal(body1, body2) {
			t.Errorf("file %s differs across rebuilds (not idempotent)", path)
		}
	}
}

// TestNSentryGrafana_UnknownSlug: a slug not in dashboardTitles surfaces an
// error rather than silently emitting a malformed dashboard.
func TestNSentryGrafana_UnknownSlug(t *testing.T) {
	if _, err := RenderNSentryDashboard("not-a-real-plugin"); err == nil {
		t.Error("unknown slug: want error, got nil")
	}
}

// snapshotDir walks root and returns {relpath: bytes} for every regular
// file. Used by the idempotency test to compare two rebuild outputs.
func snapshotDir(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := make(map[string][]byte)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = body
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}
