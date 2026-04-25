// Package monitoring handles Grafana dashboard provisioning for nself monitoring plugin.
// On nself start, this copies the relevant dashboard JSON files based on installed plugins.
package monitoring

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// dashboardEntry maps a bundle trigger plugin to a dashboard filename.
// When any plugin in RequiredPlugins is installed, the dashboard is provisioned.
type dashboardEntry struct {
	Filename        string
	RequiredPlugins []string
	Bundle          string
}

// dashboardRegistry is the list of bundle dashboards. The nself-overview is always copied.
var dashboardRegistry = []dashboardEntry{
	{
		Filename:        "nclaw-bundle.json",
		RequiredPlugins: []string{"claw", "ai", "mux"},
		Bundle:          "nclaw",
	},
	{
		Filename:        "ntv-bundle.json",
		RequiredPlugins: []string{"media-processing", "streaming"},
		Bundle:          "ntv",
	},
	{
		Filename:        "nchat-bundle.json",
		RequiredPlugins: []string{"chat", "livekit"},
		Bundle:          "nchat",
	},
	{
		Filename:        "nfamily-bundle.json",
		RequiredPlugins: []string{"social", "photos"},
		Bundle:          "nfamily",
	},
}

// nSelfProvisioningMeta is the __nself_provisioning block embedded in dashboard JSON files.
// Used to verify that the dashboard file matches expectations.
type nSelfProvisioningMeta struct {
	RequiredPlugins []string `json:"requiredPlugins"`
	Bundle          string   `json:"bundle"`
}

type dashboardFile struct {
	NselfProvisioning *nSelfProvisioningMeta `json:"__nself_provisioning,omitempty"`
}

// CopyDashboards copies the nself-overview and any bundle dashboards whose
// trigger plugins are present in installedPlugins into destDir.
// destDir is the Grafana provisioning dashboards directory (e.g. /etc/grafana/provisioning/dashboards).
// sourceDir is the monitoring plugin's configs/grafana/dashboards/ directory.
// Returns the list of dashboard filenames copied and any error encountered.
func CopyDashboards(sourceDir, destDir string, installedPlugins []string) ([]string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create dest dir %s: %w", destDir, err)
	}

	pluginSet := make(map[string]bool, len(installedPlugins))
	for _, p := range installedPlugins {
		pluginSet[p] = true
	}

	var copied []string

	// Always copy the overview dashboard.
	overviewSrc := filepath.Join(sourceDir, "nself-overview.json")
	overviewDst := filepath.Join(destDir, "nself-overview.json")
	if err := copyFile(overviewSrc, overviewDst); err != nil {
		return nil, fmt.Errorf("copy nself-overview.json: %w", err)
	}
	copied = append(copied, "nself-overview.json")

	// Copy bundle dashboards only when at least one trigger plugin is installed.
	for _, entry := range dashboardRegistry {
		if !anyInstalled(pluginSet, entry.RequiredPlugins) {
			continue
		}
		src := filepath.Join(sourceDir, entry.Filename)
		dst := filepath.Join(destDir, entry.Filename)
		if err := copyFile(src, dst); err != nil {
			return nil, fmt.Errorf("copy %s: %w", entry.Filename, err)
		}
		copied = append(copied, entry.Filename)
	}

	return copied, nil
}

// SelectDashboards returns the list of dashboard filenames that would be copied
// for the given installed plugins, without performing any file I/O.
// Useful for unit testing and dry-run output.
func SelectDashboards(installedPlugins []string) []string {
	pluginSet := make(map[string]bool, len(installedPlugins))
	for _, p := range installedPlugins {
		pluginSet[p] = true
	}

	selected := []string{"nself-overview.json"}
	for _, entry := range dashboardRegistry {
		if anyInstalled(pluginSet, entry.RequiredPlugins) {
			selected = append(selected, entry.Filename)
		}
	}
	return selected
}

// ReadProvisioningMeta reads the __nself_provisioning block from a dashboard JSON file.
// Returns nil if the key is absent (overview dashboard and non-nself dashboards).
func ReadProvisioningMeta(dashboardPath string) (*nSelfProvisioningMeta, error) {
	f, err := os.Open(dashboardPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dashboardPath, err)
	}
	defer f.Close()

	var df dashboardFile
	if err := json.NewDecoder(f).Decode(&df); err != nil {
		return nil, fmt.Errorf("decode %s: %w", dashboardPath, err)
	}
	return df.NselfProvisioning, nil
}

// anyInstalled returns true when at least one plugin in candidates is in the installed set.
func anyInstalled(installed map[string]bool, candidates []string) bool {
	for _, c := range candidates {
		if installed[c] {
			return true
		}
	}
	return false
}

// copyFile copies src to dst, creating or overwriting dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
