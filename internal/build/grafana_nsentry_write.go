package build

// Purpose: writes the full Grafana auto-provisioning tree (datasource,
// dashboard-provisioning YAML, and one dashboard JSON per installed plugin)
// to disk. Split out of grafana_nsentry.go (CLI-R12) as a pure move.
// Inputs: the project workdir and the global plugin directory.
// Outputs: the count of files written, or an error.
// Constraints: idempotent via atomicWrite (loki.go); a no-op when no ɳSentry
// plugin is installed.

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteNSentryGrafanaProvisioning emits the full Grafana auto-provisioning
// tree for the installed ɳSentry plugins under pluginDir. Writes:
//
//	<workdir>/monitoring/grafana/provisioning/datasources/nsentry.yaml
//	<workdir>/monitoring/grafana/provisioning/dashboards/nsentry.yaml
//	<workdir>/monitoring/grafana/dashboards/nsentry/<slug>.json   (one per installed plugin)
//
// Returns the count of files written (0 when no ɳSentry plugin installed).
// Idempotent: a second call with the same on-disk state produces
// byte-identical output (no .tmp leftovers, atomic rename pattern reused
// from loki.go's atomicWrite).
func WriteNSentryGrafanaProvisioning(workdir, pluginDir string) (int, error) {
	installed := installedNSentryForGrafana(pluginDir)
	if len(installed) == 0 {
		return 0, nil
	}

	dsBytes := RenderNSentryDatasourceYAML(pluginDir)
	dashProvBytes := RenderNSentryDashboardProvisioning(pluginDir)
	if dsBytes == nil || dashProvBytes == nil {
		// Defensive: installedNSentryForGrafana returned non-empty so
		// these renderers should too. Bail out if a future refactor
		// breaks the invariant.
		return 0, fmt.Errorf("grafana_nsentry: renderer returned nil despite installed plugins")
	}

	dsPath := filepath.Join(workdir, "monitoring", "grafana", "provisioning", "datasources", "nsentry.yaml")
	dashProvPath := filepath.Join(workdir, "monitoring", "grafana", "provisioning", "dashboards", "nsentry.yaml")
	dashDir := filepath.Join(workdir, "monitoring", "grafana", "dashboards", "nsentry")

	if err := os.MkdirAll(filepath.Dir(dsPath), 0o755); err != nil {
		return 0, fmt.Errorf("grafana_nsentry: mkdir %s: %w", filepath.Dir(dsPath), err)
	}
	if err := os.MkdirAll(filepath.Dir(dashProvPath), 0o755); err != nil {
		return 0, fmt.Errorf("grafana_nsentry: mkdir %s: %w", filepath.Dir(dashProvPath), err)
	}
	if err := os.MkdirAll(dashDir, 0o755); err != nil {
		return 0, fmt.Errorf("grafana_nsentry: mkdir %s: %w", dashDir, err)
	}

	written := 0
	if err := atomicWrite(dsPath, dsBytes, 0o644); err != nil {
		return written, err
	}
	written++
	if err := atomicWrite(dashProvPath, dashProvBytes, 0o644); err != nil {
		return written, err
	}
	written++

	for _, p := range installed {
		body, err := RenderNSentryDashboard(p.slug)
		if err != nil {
			return written, err
		}
		path := filepath.Join(dashDir, p.slug+".json")
		if err := atomicWrite(path, body, 0o644); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}
