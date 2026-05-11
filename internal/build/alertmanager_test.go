// S12.T03 — unit tests for the Alertmanager build-pipeline integration.
//
// Cases:
//
//  1. Validate rejects missing ProjectName.
//  2. Zero ɳSentry plugins installed → base alertmanager.yml only; no
//     nsentry receiver, no nsentry routing stanza.
//  3. Full ɳSentry baseline installed → nsentry-<project> receiver added
//     + 7 plugin-specific routing rules appended just before inhibit_rules.
//  4. Oncall override propagates to the existing oncall-stub receiver.
//  5. Idempotent rewrite — same options → byte-identical alertmanager.yml.
package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAlertmanager_Validate: missing ProjectName surfaces an explicit error
// instead of producing an unsigned receiver name like "nsentry-".
func TestAlertmanager_Validate(t *testing.T) {
	if _, err := RenderAlertmanagerBundle(AlertmanagerBuildOptions{}); err == nil ||
		!strings.Contains(err.Error(), "ProjectName is required") {
		t.Fatalf("want ProjectName required error, got %v", err)
	}
}

// TestAlertmanager_NoNSentry: with no plugins installed, the rendered YAML
// equals the base config — no nsentry receiver name, no bundle = nsentry
// matcher appears anywhere.
func TestAlertmanager_NoNSentry(t *testing.T) {
	pluginDir := installNSentryManifests(t) // empty
	body, err := RenderAlertmanagerBundle(AlertmanagerBuildOptions{
		ProjectName: "proj",
		PluginDir:   pluginDir,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if bytes.Contains(body, []byte("nsentry-proj")) {
		t.Errorf("base render should not mention nsentry-proj receiver:\n%s", body)
	}
	if bytes.Contains(body, []byte("bundle = nsentry")) {
		t.Errorf("base render should not emit bundle=nsentry matcher:\n%s", body)
	}
}

// TestAlertmanager_FullNSentry: all 7 baseline plugins installed → receiver
// added, one routing rule per plugin, rules emitted before inhibit_rules.
func TestAlertmanager_FullNSentry(t *testing.T) {
	pluginDir := installNSentryManifests(t, nsentryBaselineNames()...)
	body, err := RenderAlertmanagerBundle(AlertmanagerBuildOptions{
		ProjectName: "proj",
		PluginDir:   pluginDir,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// Receiver block must be present.
	if !bytes.Contains(body, []byte("name: nsentry-proj")) {
		t.Errorf("missing nsentry-proj receiver:\n%s", body)
	}
	// One route per baseline plugin (alphabetical: alert-router, incident-mgmt,
	// rum, slo-tracker, status-page, synthetic-monitor, uptime-monitor).
	for _, slug := range []string{
		"alert-router", "incident-mgmt", "rum", "slo-tracker",
		"status-page", "synthetic-monitor", "uptime-monitor",
	} {
		needle := []byte("- plugin = " + slug)
		if !bytes.Contains(body, needle) {
			t.Errorf("missing routing rule for plugin %s", slug)
		}
	}
	// Routes must appear BEFORE inhibit_rules; assert order on byte indexes.
	routeIdx := bytes.Index(body, []byte("bundle = nsentry"))
	inhibitIdx := bytes.Index(body, []byte("inhibit_rules:"))
	if routeIdx < 0 || inhibitIdx < 0 {
		t.Fatalf("missing routes or inhibit_rules block:\n%s", body)
	}
	if routeIdx >= inhibitIdx {
		t.Errorf("ɳSentry routes (idx=%d) must appear before inhibit_rules (idx=%d)", routeIdx, inhibitIdx)
	}
}

// TestAlertmanager_OncallOverride: passing OncallEmail rewrites the existing
// oncall-stub receiver's destination rather than appending a duplicate.
func TestAlertmanager_OncallOverride(t *testing.T) {
	pluginDir := installNSentryManifests(t) // empty — focus is on oncall
	body, err := RenderAlertmanagerBundle(AlertmanagerBuildOptions{
		ProjectName: "proj",
		PluginDir:   pluginDir,
		OncallEmail: "oncall@example.com",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.Contains(body, []byte("to: oncall@example.com")) {
		t.Errorf("oncall override did not propagate to oncall-stub receiver:\n%s", body)
	}
}

// TestAlertmanager_Idempotent: two consecutive WriteAlertmanagerConfig
// calls with identical inputs produce byte-identical alertmanager.yml.
func TestAlertmanager_Idempotent(t *testing.T) {
	pluginDir := installNSentryManifests(t, nsentryBaselineNames()...)
	workdir := t.TempDir()
	opts := AlertmanagerBuildOptions{
		ProjectName: "proj",
		PluginDir:   pluginDir,
	}

	if _, err := WriteAlertmanagerConfig(workdir, opts); err != nil {
		t.Fatalf("first write: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(workdir, "monitoring", "alertmanager.yml"))
	if err != nil {
		t.Fatalf("read first: %v", err)
	}

	if _, err := WriteAlertmanagerConfig(workdir, opts); err != nil {
		t.Fatalf("second write: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(workdir, "monitoring", "alertmanager.yml"))
	if err != nil {
		t.Fatalf("read second: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Errorf("alertmanager.yml differs across rebuilds — not idempotent\nfirst:\n%s\n---\nsecond:\n%s", first, second)
	}
}
