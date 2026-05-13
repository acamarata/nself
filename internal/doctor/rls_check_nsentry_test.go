package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// makeNSentryPluginDir creates a temp directory with plugin.json stubs for the
// given plugin names. The returned path is suitable to pass to CheckNSentryRLS.
func makeNSentryPluginDir(t *testing.T, plugins ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range plugins {
		pDir := filepath.Join(dir, name)
		if err := os.MkdirAll(pDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", pDir, err)
		}
		manifest := filepath.Join(pDir, "plugin.json")
		if err := os.WriteFile(manifest, []byte(`{"name":"`+name+`","version":"1.0.0"}`), 0o644); err != nil {
			t.Fatalf("write manifest %s: %v", manifest, err)
		}
	}
	return dir
}

// TestCheckNSentryRLS_NoneInstalled: no ɳSentry plugins → single pass result.
func TestCheckNSentryRLS_NoneInstalled(t *testing.T) {
	dir := makeNSentryPluginDir(t) // empty — no plugins

	results := CheckNSentryRLS(context.Background(), dir)
	if len(results) != 1 {
		t.Fatalf("no plugins installed: want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "pass" {
		t.Errorf("no plugins installed: want pass, got %q", r.Status)
	}
	if !containsStr(r.Message, "no ɳSentry baseline plugins installed") {
		t.Errorf("want skip message, got %q", r.Message)
	}
}

// TestCheckNSentryRLS_NonNSentryPlugin: only a non-ɳSentry plugin present
// should still be treated as "none installed" for NSENTRY-RLS purposes.
func TestCheckNSentryRLS_NonNSentryPlugin(t *testing.T) {
	dir := makeNSentryPluginDir(t, "nself-stripe", "nself-mail")

	results := CheckNSentryRLS(context.Background(), dir)
	if len(results) != 1 {
		t.Fatalf("non-nsentry plugins: want 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("non-nsentry plugins: want pass, got %q", results[0].Status)
	}
}

// TestCheckNSentryRLS_InstalledNoDB: one ɳSentry plugin installed but DB
// unreachable → results must not be empty and must not panic. The generic
// PERM-RLS-01 returns a warn when DB URL is absent; CheckNSentryRLS surfaces
// at least one result.
func TestCheckNSentryRLS_InstalledNoDB(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HASURA_GRAPHQL_ADMIN_SECRET", "")

	dir := makeNSentryPluginDir(t, "nself-uptime-monitor")

	results := CheckNSentryRLS(context.Background(), dir)
	if len(results) == 0 {
		t.Fatal("plugin installed, no DB: want at least 1 result")
	}
	// When the DB is unreachable, the generic check returns a single warn.
	// CheckNSentryRLS should propagate it (no panic, no empty slice).
	found := false
	for _, r := range results {
		if r.Status == "warn" || r.Status == "pass" || r.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Errorf("no valid status found in results: %#v", results)
	}
}

// TestCheckNSentryRLS_CheckIDs: verify that each installed plugin maps to the
// correct NSENTRY-RLS-NN check ID in the result Name field.
func TestCheckNSentryRLS_CheckIDs(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "")
	t.Setenv("DATABASE_URL", "")

	// Install all 7 baseline plugins in a temp dir.
	dir := makeNSentryPluginDir(t,
		"nself-alert-router",
		"nself-incident-mgmt",
		"nself-rum",
		"nself-slo-tracker",
		"nself-status-page",
		"nself-synthetic-monitor",
		"nself-uptime-monitor",
	)

	results := CheckNSentryRLS(context.Background(), dir)
	if len(results) == 0 {
		t.Fatal("want at least 1 result for 7 installed plugins")
	}

	// Each NSENTRY-RLS-01..07 ID must appear in at least one result Name.
	wantIDs := []string{
		"NSENTRY-RLS-01",
		"NSENTRY-RLS-02",
		"NSENTRY-RLS-03",
		"NSENTRY-RLS-04",
		"NSENTRY-RLS-05",
		"NSENTRY-RLS-06",
		"NSENTRY-RLS-07",
	}
	for _, id := range wantIDs {
		found := false
		for _, r := range results {
			if containsStr(r.Name, id) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("check ID %s not found in any result; results: %#v", id, results)
		}
	}
}

// TestFilterByTablePrefix verifies the internal helper used to distribute
// PERM-RLS-01 violations to per-plugin NSENTRY-RLS-NN checks.
func TestFilterByTablePrefix(t *testing.T) {
	input := []CheckResult{
		{Name: "PERM-RLS-01: RLS-DISABLED table=np_uptime_checks", Status: "fail"},
		{Name: "PERM-RLS-01: RLS-DISABLED table=np_incident_events", Status: "fail"},
		{Name: "PERM-RLS-01: RLS enforcement", Status: "pass"},
	}

	uptimeResults := filterByTablePrefix(input, "np_uptime_")
	if len(uptimeResults) != 1 {
		t.Fatalf("np_uptime_ filter: want 1 result, got %d", len(uptimeResults))
	}
	if !containsStr(uptimeResults[0].Name, "np_uptime_") {
		t.Errorf("wrong result filtered: %#v", uptimeResults[0])
	}

	incidentResults := filterByTablePrefix(input, "np_incident_")
	if len(incidentResults) != 1 {
		t.Fatalf("np_incident_ filter: want 1 result, got %d", len(incidentResults))
	}

	noMatch := filterByTablePrefix(input, "np_nonexistent_")
	if len(noMatch) != 0 {
		t.Errorf("np_nonexistent_ filter: want 0, got %d", len(noMatch))
	}
}
