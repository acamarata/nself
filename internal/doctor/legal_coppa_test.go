package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile is a small helper that creates parents and writes contents.
func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// stagePluginsPro builds a plugins-pro/paid skeleton inside projectDir and
// returns the path. When migrationContents is empty, no migration file is
// written (used for FAIL cases).
func stagePluginsPro(t *testing.T, projectDir, migrationContents string) string {
	t.Helper()
	paid := filepath.Join(projectDir, "plugins-pro", "paid")
	if err := os.MkdirAll(paid, 0o755); err != nil {
		t.Fatalf("mkdir plugins-pro/paid: %v", err)
	}
	if migrationContents != "" {
		writeFile(t, filepath.Join(paid, "family", "migrations", "001_coppa_consent.sql"), migrationContents)
	} else {
		// Ensure migrations dir exists so the missing-file branch is
		// exercised cleanly (file absent, dir present).
		if err := os.MkdirAll(filepath.Join(paid, "family", "migrations"), 0o755); err != nil {
			t.Fatalf("mkdir family/migrations: %v", err)
		}
	}
	return paid
}

// TestCheckLegalCOPPA_Skip_NoPlugin: family plugin not loaded → skip.
func TestCheckLegalCOPPA_Skip_NoPlugin(t *testing.T) {
	t.Setenv("NSELF_FAMILY_LOADED", "")
	t.Setenv("PLUGIN_FAMILY_INTERNAL_URL", "")

	r := CheckLegalCOPPA(t.TempDir())
	if r.Status != "skip" {
		t.Fatalf("want skip, got %q (msg=%q)", r.Status, r.Message)
	}
}

// TestCheckLegalCOPPA_Pass: plugin loaded, migration present, secret set,
// TTL within ceiling → pass.
func TestCheckLegalCOPPA_Pass(t *testing.T) {
	dir := t.TempDir()
	stagePluginsPro(t, dir, "-- coppa migration\nCREATE TABLE np_parental_consents();\n")

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("FAMILY_CONSENT_HMAC_SECRET", "deadbeef-secret")
	t.Setenv("FAMILY_CONSENT_TTL_HOURS", "168")

	r := CheckLegalCOPPA(dir)
	if r.Status != "pass" {
		t.Fatalf("want pass, got %q (msg=%q)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "LEGAL-COPPA-01") {
		t.Errorf("expected check ID in message, got %q", r.Message)
	}
}

// TestCheckLegalCOPPA_Warn_NoSecret: plugin + migration but
// FAMILY_CONSENT_HMAC_SECRET unset → warn with fix command.
func TestCheckLegalCOPPA_Warn_NoSecret(t *testing.T) {
	dir := t.TempDir()
	stagePluginsPro(t, dir, "-- coppa migration\n")

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("FAMILY_CONSENT_HMAC_SECRET", "")

	r := CheckLegalCOPPA(dir)
	if r.Status != "warn" {
		t.Fatalf("want warn, got %q (msg=%q)", r.Status, r.Message)
	}
	if r.FixCmd == "" {
		t.Errorf("warn result should include FixCmd")
	}
	if !strings.Contains(r.Message, "FAMILY_CONSENT_HMAC_SECRET") {
		t.Errorf("warn message should name the missing env var, got %q", r.Message)
	}
}

// TestCheckLegalCOPPA_Warn_TTLTooHigh: ttl > 168h ceiling → warn.
func TestCheckLegalCOPPA_Warn_TTLTooHigh(t *testing.T) {
	dir := t.TempDir()
	stagePluginsPro(t, dir, "-- coppa migration\n")

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("FAMILY_CONSENT_HMAC_SECRET", "x")
	t.Setenv("FAMILY_CONSENT_TTL_HOURS", "240")

	r := CheckLegalCOPPA(dir)
	if r.Status != "warn" {
		t.Fatalf("want warn, got %q (msg=%q)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "240") {
		t.Errorf("warn message should mention the offending value, got %q", r.Message)
	}
}

// TestCheckLegalCOPPA_Fail_NoMigration: family plugin loaded but migration
// file missing → fail.
func TestCheckLegalCOPPA_Fail_NoMigration(t *testing.T) {
	dir := t.TempDir()
	stagePluginsPro(t, dir, "") // dir present, file absent

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("FAMILY_CONSENT_HMAC_SECRET", "x")

	r := CheckLegalCOPPA(dir)
	if r.Status != "fail" {
		t.Fatalf("want fail, got %q (msg=%q)", r.Status, r.Message)
	}
	if r.FixCmd == "" {
		t.Errorf("fail result should include FixCmd")
	}
}

// TestCheckLegalCOPPA_BareInstall: family plugin loaded but plugins-pro
// source absent → pass (bare CLI install fallback).
func TestCheckLegalCOPPA_BareInstall(t *testing.T) {
	dir := t.TempDir() // no plugins-pro staged
	t.Setenv("NSELF_FAMILY_LOADED", "1")

	r := CheckLegalCOPPA(dir)
	if r.Status != "pass" {
		t.Fatalf("want pass (bare install), got %q (msg=%q)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "bare CLI install") {
		t.Errorf("expected bare-install note in message, got %q", r.Message)
	}
}

// TestReadCOPPATTLHours_Default: unset or invalid → ceiling default.
func TestReadCOPPATTLHours_Default(t *testing.T) {
	t.Setenv("FAMILY_CONSENT_TTL_HOURS", "")
	if got := readCOPPATTLHours(); got != coppaTTLCeilingHours {
		t.Errorf("unset: want %d, got %d", coppaTTLCeilingHours, got)
	}
	t.Setenv("FAMILY_CONSENT_TTL_HOURS", "not-a-number")
	if got := readCOPPATTLHours(); got != coppaTTLCeilingHours {
		t.Errorf("invalid: want %d, got %d", coppaTTLCeilingHours, got)
	}
	t.Setenv("FAMILY_CONSENT_TTL_HOURS", "0")
	if got := readCOPPATTLHours(); got != coppaTTLCeilingHours {
		t.Errorf("zero: want %d (default), got %d", coppaTTLCeilingHours, got)
	}
	t.Setenv("FAMILY_CONSENT_TTL_HOURS", "72")
	if got := readCOPPATTLHours(); got != 72 {
		t.Errorf("valid: want 72, got %d", got)
	}
}
