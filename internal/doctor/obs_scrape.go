// S12.T01 — doctor check OBS-SCRAPE-01.
//
// When any ɳSentry plugin (nself-uptime-monitor, nself-status-page,
// nself-incident-mgmt, nself-alert-router, nself-slo-tracker,
// nself-synthetic-monitor, nself-rum) is installed, the generated
// monitoring/prometheus.yml MUST contain a scrape-target stanza for each
// installed plugin (job name "nsentry-<slug>").
//
// Severity: WARN. Missing stanzas mean Prometheus is not scraping the plugin's
// /metrics endpoint, but the plugin itself still runs — degraded observability,
// not broken functionality.
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// nsentryPluginNames is the canonical list of ɳSentry baseline plugin
// directory names (matches build.nsentryPlugins). Kept in this package to
// avoid importing internal/build (which already imports nothing from doctor;
// keeping it that way is cleaner than introducing a shared types package).
var nsentryPluginNames = []string{
	"nself-alert-router",
	"nself-incident-mgmt",
	"nself-rum",
	"nself-slo-tracker",
	"nself-status-page",
	"nself-synthetic-monitor",
	"nself-uptime-monitor",
}

// CheckOBSScrape implements OBS-SCRAPE-01 — verifies Prometheus scrape config
// covers every installed ɳSentry plugin.
//
// projectDir is the user's project root (where monitoring/ lives). pluginDir
// is the plugin install directory (typically ~/.nself/plugins). When either
// is empty, the check is skipped (returns pass with explanation).
func CheckOBSScrape(_ context.Context, projectDir, pluginDir string) CheckResult {
	const (
		name    = "ɳSentry Prometheus scrape config (OBS-SCRAPE-01)"
		section = "observability"
	)

	if projectDir == "" || pluginDir == "" {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "pass",
			Message: "project or plugin directory not set — check skipped",
		}
	}

	installed := installedNSentryPlugins(pluginDir)
	if len(installed) == 0 {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "pass",
			Message: "no ɳSentry plugins installed — scrape targets not required",
		}
	}

	prometheusYAML := filepath.Join(projectDir, "monitoring", "prometheus.yml")
	body, err := os.ReadFile(prometheusYAML)
	if err != nil {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "warn",
			Message: fmt.Sprintf("ɳSentry installed (%d plugins) but %s missing or unreadable: %v",
				len(installed), prometheusYAML, err),
			FixCmd: "nself build",
		}
	}

	missing := missingScrapeJobs(string(body), installed)
	if len(missing) == 0 {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "pass",
			Message: fmt.Sprintf("all %d installed ɳSentry plugins have scrape targets", len(installed)),
		}
	}

	return CheckResult{
		Section: section,
		Name:    name,
		Status:  "warn",
		Message: fmt.Sprintf("ɳSentry plugins installed but scrape stanzas missing for: %s — re-run nself build",
			strings.Join(missing, ", ")),
		FixCmd: "nself build",
	}
}

// installedNSentryPlugins returns the slugs (without "nself-" prefix) of every
// ɳSentry plugin that has a plugin.json under pluginDir.
func installedNSentryPlugins(pluginDir string) []string {
	var out []string
	for _, name := range nsentryPluginNames {
		manifest := filepath.Join(pluginDir, name, "plugin.json")
		if _, err := os.Stat(manifest); err != nil {
			continue
		}
		// Strip "nself-" prefix for the job-name slug match.
		slug := strings.TrimPrefix(name, "nself-")
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

// missingScrapeJobs returns the slugs whose "job_name: nsentry-<slug>" line is
// absent from the rendered prometheus.yml body. The match is exact-substring;
// the YAML emitter (RenderPrometheusYAML) writes job_name on its own line.
func missingScrapeJobs(yamlBody string, installedSlugs []string) []string {
	var missing []string
	for _, slug := range installedSlugs {
		needle := "job_name: nsentry-" + slug
		if !strings.Contains(yamlBody, needle) {
			missing = append(missing, slug)
		}
	}
	return missing
}
