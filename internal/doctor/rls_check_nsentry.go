// Package doctor — rls_check_nsentry.go
//
// ɳSentry-specific RLS checks (NSENTRY-RLS-01..07).
//
// When any of the 7 ɳSentry baseline plugins is installed, nself doctor --deep
// verifies that its np_* tables have:
//   - RLS enabled (relrowsecurity = true)
//   - At least one policy
//   - FORCE RLS when a tenant_id column is present
//   - A Hasura row filter for tenant_id ({"tenant_id":{"_eq":"X-Hasura-Tenant-Id"}})
//
// Each of the 7 plugins gets its own check ID (NSENTRY-RLS-01..07) and runs
// only when that plugin's plugin.json is present in pluginDir.
//
// Severity: CRITICAL — RLS misconfiguration on ɳSentry tables exposes uptime,
// incident, and SLO data across tenants.
//
// Security-Always-Free Doctrine: these checks run without a license.
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// nsentryRLSPlugin maps one ɳSentry baseline plugin to its canonical check ID
// and the np_* table-name prefix expected for that plugin.
type nsentryRLSPlugin struct {
	name        string // installation directory name (e.g. "nself-uptime-monitor")
	checkID     string // canonical doctor check ID (e.g. "NSENTRY-RLS-01")
	tablePrefix string // np_* table prefix for this plugin (e.g. "np_uptime_")
}

// nsentryRLSPlugins is the canonical list of the 7 ɳSentry baseline plugins
// and their check IDs. Order is alphabetical by name for stable output.
var nsentryRLSPlugins = []nsentryRLSPlugin{
	{"nself-alert-router", "NSENTRY-RLS-01", "np_alert_"},
	{"nself-incident-mgmt", "NSENTRY-RLS-02", "np_incident_"},
	{"nself-rum", "NSENTRY-RLS-03", "np_rum_"},
	{"nself-slo-tracker", "NSENTRY-RLS-04", "np_slo_"},
	{"nself-status-page", "NSENTRY-RLS-05", "np_status_"},
	{"nself-synthetic-monitor", "NSENTRY-RLS-06", "np_synthetic_"},
	{"nself-uptime-monitor", "NSENTRY-RLS-07", "np_uptime_"},
}

// CheckNSentryRLS runs NSENTRY-RLS-01..07 for each installed ɳSentry baseline
// plugin. A plugin is considered installed when pluginDir/<plugin-name>/plugin.json
// exists. When none of the 7 baseline plugins are installed, the function
// returns a single pass result ("nsentry not installed — skipping").
//
// Each per-plugin check delegates to CheckRLSEnforcement (the generic PERM-RLS-01
// implementation) and filters to only the np_* tables whose name begins with
// the plugin's tablePrefix. This avoids re-querying the DB once per plugin;
// instead a single query fetches all np_* tables and per-plugin results are
// sliced from that shared list.
//
// Severity escalation: all violations are reported with status "fail" (CRITICAL).
// This is stricter than the generic PERM-RLS-01 (which uses "warn" by default)
// because ɳSentry tables hold cross-tenant observability data.
//
// The function accepts pluginDir as a parameter so tests can provide a temp
// directory without touching the real filesystem.
func CheckNSentryRLS(ctx context.Context, pluginDir string) []CheckResult {
	if pluginDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			pluginDir = filepath.Join(home, ".nself", "plugins")
		}
	}

	// Determine which of the 7 baseline plugins are installed.
	var installed []nsentryRLSPlugin
	for _, p := range nsentryRLSPlugins {
		manifest := filepath.Join(pluginDir, p.name, "plugin.json")
		if _, err := os.Stat(manifest); err == nil {
			installed = append(installed, p)
		}
	}

	if len(installed) == 0 {
		return []CheckResult{{
			Section: "security",
			Name:    "NSENTRY-RLS: ɳSentry RLS enforcement",
			Status:  "pass",
			Message: "no ɳSentry baseline plugins installed — skipping NSENTRY-RLS checks",
		}}
	}

	// Run the generic RLS enforcement query once (strict=true → fail on
	// violation). strict=true is intentional: ɳSentry tables are CRITICAL.
	genericResults := CheckRLSEnforcement(ctx, true)

	// Distribute generic results to per-plugin check IDs and filter to the
	// np_* tables owned by each plugin. Any violation that matches the plugin's
	// table prefix is re-emitted under the plugin's NSENTRY-RLS-NN check ID.
	var results []CheckResult
	for _, p := range installed {
		pluginViolations := filterByTablePrefix(genericResults, p.tablePrefix)
		if len(pluginViolations) == 0 {
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: %s RLS enforcement", p.checkID, p.name),
				Status:  "pass",
				Message: fmt.Sprintf("all %s* tables have RLS enabled, policies present, and tenant_id Hasura filters where applicable", p.tablePrefix),
			})
			continue
		}
		// Re-emit each violation under the plugin's own check ID.
		for _, v := range pluginViolations {
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: %s", p.checkID, v.Name),
				Status:  "fail", // always CRITICAL for ɳSentry tables
				Message: v.Message,
				FixCmd:  v.FixCmd,
			})
		}
	}

	return results
}

// filterByTablePrefix returns the subset of results whose Name contains the
// given table prefix. This isolates per-plugin RLS violations from the
// combined PERM-RLS-01 output without running a second DB query.
func filterByTablePrefix(results []CheckResult, prefix string) []CheckResult {
	var out []CheckResult
	for _, r := range results {
		if containsRune(r.Name, prefix) || containsRune(r.Message, prefix) {
			out = append(out, r)
		}
	}
	return out
}

// containsRune is a simple substring check used internally to avoid importing
// strings solely for Contains. The doctor package already uses strings elsewhere
// but this keeps filterByTablePrefix self-contained.
func containsRune(s, sub string) bool {
	if len(sub) == 0 || len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
