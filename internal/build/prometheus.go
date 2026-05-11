// Package build — prometheus.go: Auto-generates Prometheus scrape config for
// the ɳSentry plugin bundle.
//
// S12.T01 (P100 v1.1.0): When the ɳSentry bundle is licensed and any of its
// 7 baseline plugins are installed, `nself build` must emit scrape-target
// blocks for each one in monitoring/prometheus.yml so Prometheus can pull
// /metrics from each plugin. This file owns ɳSentry detection + scrape-target
// generation; the actual prometheus.yml render lives in
// internal/compose/monitoring/prometheus.go.
//
// Behavior:
//   - Detects which of the 7 ɳSentry plugins (uptime-monitor, status-page,
//     incident-mgmt, alert-router, slo-tracker, synthetic-monitor, rum) are
//     installed by checking <pluginDir>/<plugin-name>/plugin.json.
//   - Returns one ScrapeTarget per installed plugin with bundle="nsentry"
//     and plugin="<slug>" labels.
//   - Idempotent: re-running yields the same target list (no duplicates).
//   - Zero installed → returns nil (no nsentry stanza emitted upstream).
package build

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/nself-org/cli/internal/compose/monitoring"
)

// nsentryPlugin maps an ɳSentry plugin's installation directory name to the
// port it exposes /metrics on. Ports are reserved per F10-PORT-REGISTRY:
// 3831–3837 are the ɳSentry baseline bundle.
type nsentryPlugin struct {
	name string // installation directory name (matches plugin.json name)
	slug string // short slug used in the job label (without "nself-" prefix)
	port int    // /metrics port
}

// nsentryPlugins is the canonical, alphabetically-stable list of ALL ɳSentry
// plugins (7 baseline + 6 expansion + 2 infrastructure) and their reserved
// ports. Adding a new ɳSentry plugin: append here AND update F10-PORT-REGISTRY
// + .claude/CLAUDE.md PPI port table.
//
// Baseline (7): ports 3831–3837 — ratified at v1.1.0 launch.
// Expansion (6): ports 3838–3843 — nself-audit is free-tier but still nsentry.
// Infrastructure (2): nself-cloud (3845) and nself-csam (3846) expose /metrics
// with bundle="nsentry" and are scraped alongside the bundle plugins.
//
// Ordering (alphabetical by slug) is intentional — RenderPrometheusYAML re-sorts
// by JobName, but keeping this slice stable makes diffs and tests deterministic.
var nsentryPlugins = []nsentryPlugin{
	// Baseline bundle (7 plugins, ports 3831–3837)
	{name: "nself-alert-router", slug: "alert-router", port: 3834},
	{name: "nself-incident-mgmt", slug: "incident-mgmt", port: 3833},
	{name: "nself-rum", slug: "rum", port: 3837},
	{name: "nself-slo-tracker", slug: "slo-tracker", port: 3835},
	{name: "nself-status-page", slug: "status-page", port: 3832},
	{name: "nself-synthetic-monitor", slug: "synthetic-monitor", port: 3836},
	{name: "nself-uptime-monitor", slug: "uptime-monitor", port: 3831},
	// Expansion plugins (6 plugins, ports 3838–3843)
	{name: "nself-errors", slug: "errors", port: 3838},
	{name: "nself-cron-monitor", slug: "cron-monitor", port: 3839},
	{name: "nself-oncall", slug: "oncall", port: 3840},
	{name: "nself-crash", slug: "crash", port: 3841},
	{name: "nself-anomaly", slug: "anomaly", port: 3842},
	{name: "nself-audit", slug: "audit", port: 3843},
	// Infrastructure plugins (2 plugins, ports 3845–3846)
	// nself-cloud provides managed-host observability metrics.
	// nself-csam provides CSAM scanning metrics (compliance-critical).
	{name: "nself-cloud", slug: "cloud", port: 3845},
	{name: "nself-csam", slug: "csam", port: 3846},
}

// NSentryScrapeTargets returns one Prometheus ScrapeTarget per installed
// ɳSentry plugin under pluginDir. A plugin is considered installed when
// <pluginDir>/<plugin-name>/plugin.json exists (matching the convention used
// by ListInstalled / DiscoverPluginComposeFiles).
//
// Returns nil (not empty slice) when no ɳSentry plugin is installed so callers
// can cleanly omit the bundle stanza upstream. Output is stable-sorted by job
// name so repeated builds produce byte-identical prometheus.yml.
func NSentryScrapeTargets(pluginDir string) []monitoring.ScrapeTarget {
	if pluginDir == "" {
		return nil
	}

	installed := make([]nsentryPlugin, 0, len(nsentryPlugins))
	for _, p := range nsentryPlugins {
		manifest := filepath.Join(pluginDir, p.name, "plugin.json")
		if _, err := os.Stat(manifest); err != nil {
			continue
		}
		installed = append(installed, p)
	}
	if len(installed) == 0 {
		return nil
	}

	targets := make([]monitoring.ScrapeTarget, 0, len(installed))
	for _, p := range installed {
		targets = append(targets, monitoring.ScrapeTarget{
			JobName:     "nsentry-" + p.slug,
			ServiceName: p.name,
			Port:        p.port,
			Path:        "/metrics",
			Labels: map[string]string{
				"bundle": "nsentry",
				"plugin": p.slug,
			},
		})
	}

	// Stable order — alphabetical by job name. RenderPrometheusYAML sorts on
	// its own, but emitting in stable order here keeps unit tests simpler and
	// makes intermediate logging deterministic.
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].JobName < targets[j].JobName
	})
	return targets
}

// renderScrapeNSentryYAML renders a standalone prometheus.yml fragment
// containing only the supplied ɳSentry scrape targets. The fragment includes a
// global scrape_interval of 15s and no alerting/rule_files stanzas — it is
// intended to be written as monitoring/scrape-nsentry.yml and loaded by the
// Prometheus file_sd_configs path managed by the monitoring plugin.
//
// Callers must guarantee len(targets) > 0.
func renderScrapeNSentryYAML(targets []monitoring.ScrapeTarget) ([]byte, error) {
	cfg := &monitoring.PrometheusConfig{
		ScrapeInterval:     "15s",
		EvaluationInterval: "15s",
		Targets:            targets,
	}
	return monitoring.RenderPrometheusYAML(cfg)
}

// AppendNSentryTargets merges NSentryScrapeTargets(pluginDir) into cfg.Targets,
// skipping any target whose JobName already exists in cfg.Targets. This is the
// idempotency guard for re-running `nself build` on a project whose
// prometheus.yml was previously generated — duplicate stanzas are never
// emitted.
//
// cfg must be non-nil. Returns the count of targets added (0 when none
// installed or all already present).
func AppendNSentryTargets(cfg *monitoring.PrometheusConfig, pluginDir string) int {
	if cfg == nil {
		return 0
	}
	targets := NSentryScrapeTargets(pluginDir)
	if len(targets) == 0 {
		return 0
	}

	existing := make(map[string]struct{}, len(cfg.Targets))
	for _, t := range cfg.Targets {
		existing[t.JobName] = struct{}{}
	}

	added := 0
	for _, t := range targets {
		if _, dup := existing[t.JobName]; dup {
			continue
		}
		cfg.Targets = append(cfg.Targets, t)
		existing[t.JobName] = struct{}{}
		added++
	}
	return added
}
