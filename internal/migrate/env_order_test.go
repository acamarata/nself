package migrate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeTestEnvFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// TestDetectEnvOrder_NoChangeNeeded verifies that a project whose files never
// define the same key in more than one place — the common case — reports
// NoChangeNeeded rather than churning anything.
func TestDetectEnvOrder_NoChangeNeeded(t *testing.T) {
	dir := t.TempDir()
	writeTestEnvFile(t, dir, ".env", "PROJECT_NAME=demo\n")
	writeTestEnvFile(t, dir, ".env.dev", "HASURA_GRAPHQL_DEV_MODE=true\n")
	writeTestEnvFile(t, dir, ".env.secrets", "POSTGRES_PASSWORD=abc123\n")

	report, err := DetectEnvOrder(dir)
	if err != nil {
		t.Fatalf("DetectEnvOrder() error: %v", err)
	}
	if !report.NoChangeNeeded {
		t.Errorf("NoChangeNeeded = false, want true; changes: %+v", report.Changes)
	}
	if len(report.Changes) != 0 {
		t.Errorf("Changes = %d, want 0", len(report.Changes))
	}
}

// TestDetectEnvOrder_IdenticalValuesAcrossFiles verifies that a key set to
// the SAME value in both a low- and high-precedence file is not reported as
// drift — only an actual effective-value change counts.
func TestDetectEnvOrder_IdenticalValuesAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestEnvFile(t, dir, ".env", "SHARED_VAR=same-value\n")
	writeTestEnvFile(t, dir, ".env.dev", "ENV=dev\n")
	writeTestEnvFile(t, dir, ".env.secrets", "SHARED_VAR=same-value\n")

	report, err := DetectEnvOrder(dir)
	if err != nil {
		t.Fatalf("DetectEnvOrder() error: %v", err)
	}
	if !report.NoChangeNeeded {
		t.Errorf("NoChangeNeeded = false, want true (identical values); changes: %+v", report.Changes)
	}
}

// TestMigrate_FixesBareEnvOverridingSecrets is the primary regression this
// ticket exists to fix: under the legacy order, bare .env won over
// .env.secrets. Migrate must preserve that old effective value by writing it
// into .env.secrets so the resolved config doesn't silently change.
func TestMigrate_FixesBareEnvOverridingSecrets(t *testing.T) {
	dir := t.TempDir()
	writeTestEnvFile(t, dir, ".env", "POSTGRES_PASSWORD=old-effective-value\n")
	writeTestEnvFile(t, dir, ".env.dev", "ENV=dev\n")
	writeTestEnvFile(t, dir, ".env.secrets", "POSTGRES_PASSWORD=different-secrets-value\n")

	report, err := Migrate(dir)
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if report.NoChangeNeeded {
		t.Fatal("NoChangeNeeded = true, want a drift to be detected and fixed")
	}
	if report.FixedCount() != 1 {
		t.Fatalf("FixedCount() = %d, want 1; changes: %+v", report.FixedCount(), report.Changes)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("reading .env.secrets: %v", err)
	}
	if got := string(data); !strings.Contains(got, "POSTGRES_PASSWORD=old-effective-value") {
		t.Errorf(".env.secrets = %q, want it to contain the preserved old effective value", got)
	}

	info, err := os.Stat(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("stat .env.secrets: %v", err)
	}
	// Windows has no Unix permission bits — Go's os package models only the
	// read-only flag, so a file written 0600 reports 0666 there. The 0600
	// requirement is a real invariant on Unix (P15 shipped .env.local at 0644
	// and leaked secrets), so the production chmod stays; only the assertion
	// is platform-gated.
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf(".env.secrets perm = %o, want 0600", perm)
		}
	}
}

// TestMigrate_Idempotent verifies that running Migrate twice on the same
// project only rewrites anything the first time; the second run reports
// NoChangeNeeded.
func TestMigrate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeTestEnvFile(t, dir, ".env", "POSTGRES_PASSWORD=old-effective-value\n")
	writeTestEnvFile(t, dir, ".env.dev", "ENV=dev\n")
	writeTestEnvFile(t, dir, ".env.secrets", "POSTGRES_PASSWORD=different-secrets-value\n")

	first, err := Migrate(dir)
	if err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}
	if first.NoChangeNeeded {
		t.Fatal("first run: NoChangeNeeded = true, want a fix to happen")
	}

	firstSecrets, err := os.ReadFile(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("reading .env.secrets after first run: %v", err)
	}

	second, err := Migrate(dir)
	if err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}
	if !second.NoChangeNeeded {
		t.Errorf("second run: NoChangeNeeded = false, want true (idempotent); changes: %+v", second.Changes)
	}

	secondSecrets, err := os.ReadFile(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("reading .env.secrets after second run: %v", err)
	}
	if string(firstSecrets) != string(secondSecrets) {
		t.Error(".env.secrets content changed on the second (idempotent) run")
	}
}

// TestDetectEnvOrder_LocalOverrideFlaggedNotFixed verifies the honest limit:
// when .env.local sets a variable that a committed file also sets with a
// different value, the shim must NOT silently pick a side — it flags the
// conflict for manual review and leaves every file untouched.
func TestDetectEnvOrder_LocalOverrideFlaggedNotFixed(t *testing.T) {
	dir := t.TempDir()
	writeTestEnvFile(t, dir, ".env", "API_KEY=shared-value\n")
	writeTestEnvFile(t, dir, ".env.dev", "ENV=dev\n")
	writeTestEnvFile(t, dir, ".env.local", "API_KEY=personal-value\n")

	report, err := DetectEnvOrder(dir)
	if err != nil {
		t.Fatalf("DetectEnvOrder() error: %v", err)
	}
	if report.NoChangeNeeded {
		t.Fatal("NoChangeNeeded = true, want the .env.local conflict to be detected")
	}
	if report.ManualReviewCount() != 1 {
		t.Fatalf("ManualReviewCount() = %d, want 1; changes: %+v", report.ManualReviewCount(), report.Changes)
	}
	if report.FixedCount() != 0 {
		t.Fatalf("FixedCount() = %d, want 0 (must not auto-fix a .env.local shadow)", report.FixedCount())
	}

	if err := Apply(dir, report); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	localData, err := os.ReadFile(filepath.Join(dir, ".env.local"))
	if err != nil {
		t.Fatalf("reading .env.local: %v", err)
	}
	if got := string(localData); got != "API_KEY=personal-value\n" {
		t.Errorf(".env.local was modified by Apply(): %q", got)
	}
}

// TestDetectEnvOrder_DevLeakIntoStagingFlaggedNotFixed verifies the other
// honest limit: the legacy order always loaded .env.dev even when running
// under staging/prod. That leak is itself a bug the reorder removes, so
// Migrate must not "preserve" it by baking the leaked value into
// .env.secrets — it flags it for manual review instead.
func TestDetectEnvOrder_DevLeakIntoStagingFlaggedNotFixed(t *testing.T) {
	dir := t.TempDir()
	writeTestEnvFile(t, dir, ".env.dev", "SOME_DEV_ONLY_VAR=dev-value\n")
	writeTestEnvFile(t, dir, ".env.staging", "ENV=staging\n")

	report, err := DetectEnvOrder(dir)
	if err != nil {
		t.Fatalf("DetectEnvOrder() error: %v", err)
	}
	if report.NoChangeNeeded {
		t.Fatal("NoChangeNeeded = true, want the dev-leak drift to be detected")
	}

	var found bool
	for _, c := range report.Changes {
		if c.Var == "SOME_DEV_ONLY_VAR" && c.EnvName == "staging" {
			found = true
			if c.Action != ActionManualReview {
				t.Errorf("Action = %q, want %q for a dev-leak into staging", c.Action, ActionManualReview)
			}
		}
	}
	if !found {
		t.Fatalf("expected a change entry for SOME_DEV_ONLY_VAR/staging; got %+v", report.Changes)
	}
}

// TestMigrate_FoldsAndArchivesEnvAI verifies that a legacy .env.ai file's
// content is folded into .env.secrets and the file is archived (renamed, not
// deleted) once every one of its keys has been safely resolved.
func TestMigrate_FoldsAndArchivesEnvAI(t *testing.T) {
	dir := t.TempDir()
	writeTestEnvFile(t, dir, ".env.dev", "ENV=dev\n")
	writeTestEnvFile(t, dir, ".env.ai", "NSELF_MASTER_SECRET=super-secret-kek\nAI_PROFILE=auto\n")

	report, err := Migrate(dir)
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if report.NoChangeNeeded {
		t.Fatal("NoChangeNeeded = true, want .env.ai keys to be detected as drift (removed from the new cascade)")
	}
	if !report.AIArchived {
		t.Error("AIArchived = false, want true")
	}

	if _, err := os.Stat(filepath.Join(dir, ".env.ai")); !os.IsNotExist(err) {
		t.Error(".env.ai still exists — expected it to be renamed away")
	}
	if _, err := os.Stat(filepath.Join(dir, ".env.ai.migrated")); err != nil {
		t.Errorf(".env.ai.migrated not found: %v", err)
	}

	secrets, err := os.ReadFile(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("reading .env.secrets: %v", err)
	}
	got := string(secrets)
	if !strings.Contains(got, "NSELF_MASTER_SECRET=super-secret-kek") {
		t.Errorf(".env.secrets missing folded NSELF_MASTER_SECRET: %q", got)
	}
	if !strings.Contains(got, "AI_PROFILE=auto") {
		t.Errorf(".env.secrets missing folded AI_PROFILE: %q", got)
	}
}
