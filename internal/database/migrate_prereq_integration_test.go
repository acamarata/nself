//go:build integration

// Integration coverage for the Hasura-migration-prerequisite refusal
// (migrate_prereq.go) against a REAL Postgres instance — the pure
// combinatorial logic is covered without a database in
// migrate_prereq_test.go; these tests prove MigrateUp/MigrateUpDir actually
// wire it in and that a real to_regclass round trip agrees with the pure
// classification.
//
// Run with:
//
//	INTEGRATION=1 go test -mod=vendor -tags integration -timeout 120s \
//	    ./internal/database/... -run TestMigrateUp_Integration_.*Prerequisite
package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/errs"
)

// TestMigrateUp_Integration_RefusesMissingAlterPrerequisite reproduces the
// real bug: a migration ALTERs a table nothing in the pending batch creates
// and it does not exist live either (the "fresh database, only the numbered
// chain replayed" case). MigrateUp must refuse before applying anything,
// naming the missing table.
func TestMigrateUp_Integration_RefusesMissingAlterPrerequisite(t *testing.T) {
	skipUnlessIntegration(t)
	cfg := startTestPostgres(t)

	dir := t.TempDir()
	t.Chdir(dir)

	migDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	const sql = "ALTER TABLE licenses ADD COLUMN tier VARCHAR(20) NOT NULL DEFAULT 'pro';"
	if err := os.WriteFile(filepath.Join(migDir, "009_licensing_tiers.sql"), []byte(sql), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	count, err := MigrateUp(ctx, cfg, "")
	if err == nil {
		t.Fatal("MigrateUp succeeded against an empty database ALTERing a table nothing creates, want a refusal")
	}
	if !errors.Is(err, errs.ErrMigrationPrerequisiteMissing) {
		t.Fatalf("MigrateUp error does not wrap ErrMigrationPrerequisiteMissing: %v", err)
	}
	if !strings.Contains(err.Error(), `"licenses"`) {
		t.Fatalf("refusal does not name the missing table: %v", err)
	}
	if count != 0 {
		t.Fatalf("MigrateUp reported %d applied migrations on a refused batch, want 0 (refuse-first: nothing runs)", count)
	}

	// Confirm nothing was actually attempted: no ledger row for the file.
	out, queryErr := querySQL(ctx, cfg, cfg.Postgres.DB,
		"SELECT count(*) FROM nself_ops.migrations WHERE id = '009_licensing_tiers'")
	if queryErr != nil {
		t.Fatalf("query nself_ops.migrations: %v", queryErr)
	}
	if strings.TrimSpace(out) != "0" {
		t.Fatalf("nself_ops.migrations has a row for the refused migration: %s", out)
	}
}

// TestMigrateUp_Integration_ProceedsWhenPrerequisiteAlreadyLive proves the
// non-refusal path: the same ALTER succeeds once the target table already
// exists in the live schema (the production/staging case, where the table
// was created out of band by Hasura's migrations).
func TestMigrateUp_Integration_ProceedsWhenPrerequisiteAlreadyLive(t *testing.T) {
	skipUnlessIntegration(t)
	cfg := startTestPostgres(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := pipeSQLToContainer(ctx, cfg, "CREATE TABLE licenses (id serial PRIMARY KEY);"); err != nil {
		t.Fatalf("seed licenses table: %v", err)
	}

	dir := t.TempDir()
	t.Chdir(dir)
	migDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	const sql = "ALTER TABLE licenses ADD COLUMN tier VARCHAR(20) NOT NULL DEFAULT 'pro';"
	if err := os.WriteFile(filepath.Join(migDir, "009_licensing_tiers.sql"), []byte(sql), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	count, err := MigrateUp(ctx, cfg, "")
	if err != nil {
		t.Fatalf("MigrateUp: %v, want success since licenses already exists live", err)
	}
	if count != 1 {
		t.Fatalf("MigrateUp applied %d migrations, want 1", count)
	}

	out, queryErr := querySQL(ctx, cfg, cfg.Postgres.DB,
		"SELECT column_name FROM information_schema.columns WHERE table_name = 'licenses' AND column_name = 'tier'")
	if queryErr != nil || strings.TrimSpace(out) != "tier" {
		t.Fatalf("licenses.tier column was not added: out=%q err=%v", out, queryErr)
	}
}

// TestMigrateUpDir_Integration_RefusesThenIdempotentOnceFixed exercises the
// --migration-dir entry point (what a numbered chain like
// backend/migrations/ actually runs through) end to end: first without the
// prerequisite (refuses, nothing applied), then with it created out of band
// (succeeds), then a second identical run (idempotent: no re-apply, no
// re-refusal now that the migration is already recorded).
func TestMigrateUpDir_Integration_RefusesThenIdempotentOnceFixed(t *testing.T) {
	skipUnlessIntegration(t)
	cfg := startTestPostgres(t)

	dir := t.TempDir()
	t.Chdir(dir)
	migDir := filepath.Join(dir, "backend_migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const sql = "ALTER TABLE licenses ADD COLUMN tier VARCHAR(20) NOT NULL DEFAULT 'pro';"
	if err := os.WriteFile(filepath.Join(migDir, "009_licensing_tiers.sql"), []byte(sql), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 1. Refuses against the empty database.
	if _, err := MigrateUpDir(ctx, cfg, migDir); err == nil {
		t.Fatal("MigrateUpDir succeeded before licenses existed, want a refusal")
	} else if !errors.Is(err, errs.ErrMigrationPrerequisiteMissing) {
		t.Fatalf("MigrateUpDir error does not wrap ErrMigrationPrerequisiteMissing: %v", err)
	}

	// 2. Create the prerequisite out of band (standing in for the Hasura
	// migration this refusal points the operator at) and retry: succeeds.
	if err := pipeSQLToContainer(ctx, cfg, "CREATE TABLE licenses (id serial PRIMARY KEY);"); err != nil {
		t.Fatalf("seed licenses table: %v", err)
	}
	count, err := MigrateUpDir(ctx, cfg, migDir)
	if err != nil {
		t.Fatalf("MigrateUpDir after seeding prerequisite: %v", err)
	}
	if count != 1 {
		t.Fatalf("MigrateUpDir applied %d migrations, want 1", count)
	}

	// 3. Re-run: idempotent. Already applied, so no re-apply and — the
	// specific regression this locks — no re-evaluation of the prerequisite
	// gate either (it must see an empty pending set and skip straight
	// through, not re-refuse or re-touch the table).
	count, err = MigrateUpDir(ctx, cfg, migDir)
	if err != nil {
		t.Fatalf("MigrateUpDir second run: %v, want a clean idempotent no-op", err)
	}
	if count != 0 {
		t.Fatalf("MigrateUpDir second run applied %d migrations, want 0 (already applied)", count)
	}
}
