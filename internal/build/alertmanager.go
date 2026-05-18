// Package build — alertmanager.go: Build-pipeline integration for the
// Alertmanager routing config, with ɳSentry-aware receiver + route generation.
//
// `nself build` must produce a complete monitoring/alertmanager.yml that
// includes a dedicated `nsentry` receiver plus one routing rule per installed
// ɳSentry plugin. This file owns the build-time stitching — the base
// AlertmanagerConfig template lives in internal/compose/monitoring/alerts.go
// and is reused here so we never fork the YAML schema.
//
// Behavior:
//   - WriteAlertmanagerConfig renders alertmanager.yml + appends ɳSentry
//     routing stanzas atomically into <workdir>/monitoring/alertmanager.yml.
//   - When zero ɳSentry plugins are installed, no nsentry receiver or route
//     blocks are emitted; the base config stands alone.
//   - Idempotent: identical inputs → byte-identical output, so Docker
//     bind-mounts don't churn on rebuild.
package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/compose/monitoring"
)

// AlertmanagerBuildOptions captures the build-time inputs that shape the
// rendered alertmanager.yml. ProjectName is required so the receiver name
// can be scoped per nSelf project (avoids collisions when a single
// Alertmanager fronts multiple deploys).
type AlertmanagerBuildOptions struct {
	// ProjectName is the nSelf project slug. Required.
	ProjectName string
	// PluginDir is the directory under which ɳSentry plugin.json manifests
	// live (typically <workdir>/.nself/plugins). Empty disables ɳSentry
	// detection entirely (no nsentry stanzas emitted).
	PluginDir string
	// OncallEmail overrides the default ALERTMANAGER_ONCALL_EMAIL stub for
	// the critical receiver. Empty falls back to the env-var default in
	// monitoring.DefaultAlertmanagerConfig.
	OncallEmail string
	// NSentryEmail is the destination address for ɳSentry-routed alerts
	// when at least one ɳSentry plugin is installed. Empty defaults to
	// the same env-var fallback as the oncall receiver.
	NSentryEmail string
}

// validate ensures the option set is internally consistent. ProjectName is
// the only hard requirement; ɳSentry handling is conditional.
func (o AlertmanagerBuildOptions) validate() error {
	if strings.TrimSpace(o.ProjectName) == "" {
		return fmt.Errorf("alertmanager build: ProjectName is required")
	}
	return nil
}

// resolveAlertmanagerConfig returns the *monitoring.AlertmanagerConfig the
// underlying renderer expects, applying build-time overrides on top of the
// monitoring package defaults. When ɳSentry plugins are installed, an extra
// receiver named `nsentry-<project>` is appended.
//
// The function is pure (no filesystem side effects) so callers can unit-test
// the resulting Receivers slice without disk I/O.
func resolveAlertmanagerConfig(opts AlertmanagerBuildOptions) *monitoring.AlertmanagerConfig {
	cfg := monitoring.DefaultAlertmanagerConfig()

	// Override oncall email on the existing oncall receiver.
	if opts.OncallEmail != "" {
		for i := range cfg.Receivers {
			if cfg.Receivers[i].Name == "oncall" {
				cfg.Receivers[i].EmailTo = opts.OncallEmail
			}
		}
	}

	// ɳSentry receiver — only added when at least one plugin is installed.
	if len(installedNSentryForGrafana(opts.PluginDir)) > 0 {
		email := opts.NSentryEmail
		if email == "" {
			email = "${ALERTMANAGER_NSENTRY_EMAIL:-${ALERTMANAGER_ONCALL_EMAIL:-support@nself.org}}"
		}
		cfg.Receivers = append(cfg.Receivers, monitoring.AlertReceiver{
			Name:    "nsentry-" + opts.ProjectName,
			EmailTo: email,
		})
	}
	return cfg
}

// renderNSentryRoutes returns the YAML snippet that injects one match rule
// per installed ɳSentry plugin under the main `route.routes` list. The
// rendered text is intended to be appended to the base alertmanager.yml just
// before the `inhibit_rules:` block; routing rules are evaluated in order
// so emitting ɳSentry rules after the severity rules keeps severity routing
// authoritative for non-ɳSentry alerts.
//
// Returns nil when no ɳSentry plugin is installed so callers can branch
// cleanly. Stable order (alphabetical by slug) matches NSentryScrapeTargets.
func renderNSentryRoutes(opts AlertmanagerBuildOptions) []byte {
	installed := installedNSentryForGrafana(opts.PluginDir)
	if len(installed) == 0 {
		return nil
	}
	receiverName := "nsentry-" + opts.ProjectName

	var buf bytes.Buffer
	buf.WriteString("\n# ɳSentry bundle routing — appended by nself build (S12.T03).\n")
	buf.WriteString("# One route per installed ɳSentry plugin so alerts land on the\n")
	buf.WriteString("# bundle-specific receiver instead of the generic oncall stub.\n")
	buf.WriteString("# Appended to the existing route tree's `routes:` block.\n")
	for _, p := range installed {
		fmt.Fprintf(&buf, `    - matchers:
        - bundle = nsentry
        - plugin = %s
      receiver: %s
      continue: false
`, p.slug, receiverName)
	}
	return buf.Bytes()
}

// RenderAlertmanagerBundle returns the rendered alertmanager.yml bytes. The
// output is the base alertmanager template (from internal/compose/monitoring)
// with ɳSentry routing rules spliced in just before the `inhibit_rules:`
// section. When no ɳSentry plugin is installed, the splice is a no-op and
// the output equals the base render.
//
// Pure function — no filesystem side effects. Unit tests assert on the
// returned bytes directly.
func RenderAlertmanagerBundle(opts AlertmanagerBuildOptions) ([]byte, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	cfg := resolveAlertmanagerConfig(opts)
	base, err := monitoring.RenderAlertmanagerYAML(cfg)
	if err != nil {
		return nil, fmt.Errorf("alertmanager build: render base: %w", err)
	}

	routes := renderNSentryRoutes(opts)
	if routes == nil {
		return base, nil
	}

	// Splice the ɳSentry routes just before `inhibit_rules:`. The base
	// template always includes that line; if it ever changes shape this
	// splice fails closed (returns base unchanged + error) rather than
	// producing a broken YAML file.
	marker := []byte("\ninhibit_rules:")
	idx := bytes.Index(base, marker)
	if idx < 0 {
		return nil, fmt.Errorf("alertmanager build: marker 'inhibit_rules:' not found in base render — template drifted")
	}
	var out bytes.Buffer
	out.Write(base[:idx])
	out.Write(routes)
	out.Write(base[idx:])
	return out.Bytes(), nil
}

// WriteAlertmanagerConfig renders the alertmanager.yml and writes it into
// <workdir>/monitoring/alertmanager.yml atomically. Returns 1 on success
// (one file written), 0 on validate failure. Uses the same atomicWrite
// pattern as the Loki + Prometheus pipelines so concurrent Docker readers
// never see partial files.
func WriteAlertmanagerConfig(workdir string, opts AlertmanagerBuildOptions) (int, error) {
	body, err := RenderAlertmanagerBundle(opts)
	if err != nil {
		return 0, err
	}
	monDir := filepath.Join(workdir, "monitoring")
	if err := os.MkdirAll(monDir, 0o755); err != nil {
		return 0, fmt.Errorf("alertmanager build: mkdir %s: %w", monDir, err)
	}
	if err := atomicWrite(filepath.Join(monDir, "alertmanager.yml"), body, 0o644); err != nil {
		return 0, err
	}
	return 1, nil
}
