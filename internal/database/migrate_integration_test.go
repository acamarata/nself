//go:build integration

// Package database integration tests validate the migration ledger's
// same-day full-id keying and the dry-run SQL parse validator against a
// REAL Postgres instance (Docker container). This closes the live-verification
// gap P6-E11-W2-S1-T1's 2026-09-03 amendment identified: migrate_ledger_test.go
// and migrate_validate_test.go cover pure helpers (migrationKey, wrapDryRunSQL)
// but nothing previously exercised MigrateUp end-to-end against a running
// Postgres — the commit that added validateMigrationSQL (d33274ca, PR #347)
// explicitly said live verification was a follow-up. These tests are that
// follow-up.
//
// Run with:
//
//	INTEGRATION=1 go test -mod=vendor -tags integration -timeout 120s \
//	    ./internal/database/... -run TestMigrateUp_Integration
package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// skipUnlessIntegration mirrors the convention used by
// internal/controlplane/sim/integration_test.go and internal/embedded's
// integration tests: opt-in via INTEGRATION=1, Docker required.
func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("set INTEGRATION=1 to run Docker Postgres integration tests")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH")
	}
}

// startTestPostgres starts a scratch postgres:16-alpine container under a
// unique per-test project name (never collides with a real dev stack's
// "<project>_postgres" container) and registers cleanup to remove it.
// Returns the *config.Config MigrateUp needs. No host port is published:
// pipeSQLToContainer / querySQL dispatch via `docker exec <container> psql`,
// exactly like a real project's non-embedded Postgres path, so a TCP
// connection is never required here either.
func startTestPostgres(t *testing.T) *config.Config {
	t.Helper()

	project := fmt.Sprintf("t1integ%d", time.Now().UnixNano())
	container := project + "_postgres"

	runArgs := []string{
		"run", "-d", "--name", container,
		"-e", "POSTGRES_PASSWORD=test_integration_pw",
		"-e", "POSTGRES_DB=nself",
		"postgres:16-alpine",
	}
	if out, err := exec.Command("docker", runArgs...).CombinedOutput(); err != nil {
		t.Fatalf("docker run postgres:16-alpine: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", container).Run()
	})

	// Probe with a real query against the target database, not pg_isready:
	// the official postgres image restarts once internally after its first
	// initdb-driven boot, and pg_isready can report "accepting connections"
	// against that transient first instance moments before it cycles —
	// a window callers that query the "nself" database immediately
	// (bypassing MigrateUp's own warm-up round trips) can lose the race
	// with. SELECT 1 against the actual database is the same probe every
	// real caller in this package makes, so it never gives a false ready.
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := exec.Command("docker", "exec", container, "psql", "-U", "postgres", "-d", "nself", "-c", "SELECT 1").Run(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			logs, _ := exec.Command("docker", "logs", container).CombinedOutput()
			t.Fatalf("postgres container %s never became ready:\n%s", container, logs)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return &config.Config{
		ProjectName: project,
		Postgres: config.PostgresConfig{
			User:     "postgres",
			DB:       "nself",
			Password: "test_integration_pw",
		},
	}
}

// TestMigrateUp_Integration_SameDayMigrationsBothApply is the live
// reproduction the ticket's guide step 2 mandates: two migrations sharing a
// date prefix must both apply and both land as distinct ledger rows against
// a real Postgres — regression-locking the same-day PK collision bug
// (msg-2026-07-02-nself-migration-ledger-pk-bug.md) end-to-end, not just via
// the pure migrationKey()/pendingMigrationFiles() unit tests, which never
// touch a database.
func TestMigrateUp_Integration_SameDayMigrationsBothApply(t *testing.T) {
	skipUnlessIntegration(t)
	cfg := startTestPostgres(t)

	dir := t.TempDir()
	t.Chdir(dir)

	migDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	fixtures := map[string]string{
		"20260827_a_create_widgets.sql": "CREATE TABLE widgets (id serial primary key);",
		"20260827_b_create_gadgets.sql": "CREATE TABLE gadgets (id serial primary key);",
	}
	for name, sql := range fixtures {
		if err := os.WriteFile(filepath.Join(migDir, name), []byte(sql), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	count, err := MigrateUp(ctx, cfg, "")
	if err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	if count != 2 {
		t.Fatalf("MigrateUp applied %d migrations, want 2 (same-day pair)", count)
	}

	opsOut, err := querySQL(ctx, cfg, cfg.Postgres.DB,
		"SELECT id FROM nself_ops.migrations WHERE id LIKE '20260827%' ORDER BY id")
	if err != nil {
		t.Fatalf("query nself_ops.migrations: %v", err)
	}
	opsRows := strings.Split(strings.TrimSpace(opsOut), "\n")
	if len(opsRows) != 2 || opsRows[0] != "20260827_a_create_widgets" || opsRows[1] != "20260827_b_create_gadgets" {
		t.Fatalf("nself_ops.migrations rows = %q, want 2 distinct same-day ids (the PK-collision regression this test locks)", opsOut)
	}

	legacyOut, err := querySQL(ctx, cfg, cfg.Postgres.DB,
		"SELECT name FROM np_common.schema_versions WHERE name LIKE '20260827%' ORDER BY name")
	if err != nil {
		t.Fatalf("query np_common.schema_versions: %v", err)
	}
	legacyRows := strings.Split(strings.TrimSpace(legacyOut), "\n")
	if len(legacyRows) != 2 {
		t.Fatalf("np_common.schema_versions rows = %q, want 2 distinct same-day names", legacyOut)
	}

	// Confirm the tables the migrations created both actually exist — proof
	// both migrations were REALLY applied, not just recorded in the ledger.
	for _, table := range []string{"widgets", "gadgets"} {
		out, err := querySQL(ctx, cfg, cfg.Postgres.DB,
			fmt.Sprintf("SELECT to_regclass('public.%s')", table))
		if err != nil || strings.TrimSpace(out) == "" {
			t.Fatalf("table %s was not created by MigrateUp: out=%q err=%v", table, out, err)
		}
	}
}

// TestMigrateUp_Integration_DryRunRejectsInvalidConstructBeforeApply is the
// live reproduction the ticket's guide steps 3-4 mandate: a migration
// containing the invalid `ADD CONSTRAINT IF NOT EXISTS` construct must be
// rejected by validateMigrationSQL's dry run BEFORE the real apply, against a
// real Postgres — whose parser is the actual authority on this construct's
// invalidity (no Postgres version supports IF NOT EXISTS on ADD CONSTRAINT).
func TestMigrateUp_Integration_DryRunRejectsInvalidConstructBeforeApply(t *testing.T) {
	skipUnlessIntegration(t)
	cfg := startTestPostgres(t)

	dir := t.TempDir()
	t.Chdir(dir)

	migDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	const badSQL = "CREATE TABLE accounts (id serial primary key, name text);\n" +
		"ALTER TABLE accounts ADD CONSTRAINT IF NOT EXISTS ck_accounts_name CHECK (name <> '');"
	if err := os.WriteFile(filepath.Join(migDir, "20260901_bad_constraint.sql"), []byte(badSQL), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	count, err := MigrateUp(ctx, cfg, "")
	if err == nil {
		t.Fatalf("MigrateUp succeeded on an invalid ADD CONSTRAINT IF NOT EXISTS migration, want a rejected dry run")
	}
	if !errors.Is(err, errs.ErrMigrationValidationFailed) {
		t.Fatalf("MigrateUp error does not wrap ErrMigrationValidationFailed: %v", err)
	}
	if !strings.Contains(err.Error(), "ADD CONSTRAINT IF NOT EXISTS") {
		t.Fatalf("dry-run rejection does not name the invalid construct in its error output: %v", err)
	}
	if count != 0 {
		t.Fatalf("MigrateUp reported %d applied migrations on a rejected dry run, want 0 (real apply must never run after a dry-run failure)", count)
	}

	// Confirm the real apply genuinely never happened: no ledger row, no
	// table. This is what would catch a dry-run/real-apply ordering
	// regression that the error-string assertions above would not.
	opsOut, err := querySQL(ctx, cfg, cfg.Postgres.DB,
		"SELECT count(*) FROM nself_ops.migrations WHERE id = '20260901_bad_constraint'")
	if err != nil {
		t.Fatalf("query nself_ops.migrations: %v", err)
	}
	if strings.TrimSpace(opsOut) != "0" {
		t.Fatalf("nself_ops.migrations has a row for the rejected migration: %s (real apply ran despite dry-run failure)", opsOut)
	}
	regOut, err := querySQL(ctx, cfg, cfg.Postgres.DB, "SELECT to_regclass('public.accounts')")
	if err != nil {
		t.Fatalf("query to_regclass: %v", err)
	}
	if strings.TrimSpace(regOut) != "" {
		t.Fatalf("table accounts exists despite the migration being rejected by the dry run: %q", regOut)
	}
}
