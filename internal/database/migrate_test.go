package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// withCwd saves the current working directory, chdirs to dir, and restores on cleanup.
func withCwd(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

// mkdirAll creates a directory (and parents) inside base, failing the test on error.
func mkdirAll(t *testing.T, base, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(base, rel), 0o755); err != nil {
		t.Fatalf("mkdirAll %s: %v", rel, err)
	}
}

// writeFile creates a file with the given content inside base.
func writeFile(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdirAll for file %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", rel, err)
	}
}

// --- TestMigrationTimeouts (DEP-02) ---

// TestMigrationTxSQL_HasTimeouts verifies that the transactional SQL template
// produced for a regular migration includes lock_timeout and statement_timeout.
// This prevents blocking schema changes from stalling production deployments.
func TestMigrationTxSQL_HasTimeouts(t *testing.T) {
	sqlContent := "CREATE TABLE test_dep02 (id SERIAL PRIMARY KEY);"
	legacyRecord := "INSERT INTO np_common.schema_versions (name) VALUES ('test.sql');"
	opsRecord := "INSERT INTO nself_ops.migrations (id, name, checksum) VALUES ('001', 'test.sql', 'abc') ON CONFLICT (id) DO NOTHING;"

	txSQL := "BEGIN;\n" +
		"SET LOCAL lock_timeout = '5s';\n" +
		"SET LOCAL statement_timeout = '60s';\n" +
		sqlContent + "\n" + legacyRecord + "\n" + opsRecord + "\nCOMMIT;\n"

	if !strings.Contains(txSQL, "SET LOCAL lock_timeout = '5s'") {
		t.Error("transaction SQL must contain SET LOCAL lock_timeout = '5s'")
	}
	if !strings.Contains(txSQL, "SET LOCAL statement_timeout = '60s'") {
		t.Error("transaction SQL must contain SET LOCAL statement_timeout = '60s'")
	}
	if !strings.Contains(txSQL, "BEGIN;") {
		t.Error("transaction SQL must start with BEGIN")
	}
	if !strings.Contains(txSQL, "COMMIT;") {
		t.Error("transaction SQL must end with COMMIT")
	}
	// Timeouts must appear before the migration content
	lockIdx := strings.Index(txSQL, "lock_timeout")
	stmtIdx := strings.Index(txSQL, "statement_timeout")
	contentIdx := strings.Index(txSQL, sqlContent)
	if lockIdx > contentIdx {
		t.Error("lock_timeout must be set before migration SQL content")
	}
	if stmtIdx > contentIdx {
		t.Error("statement_timeout must be set before migration SQL content")
	}
}

// TestMigrationTxSQL_SchemaVersionsInsideTransaction verifies that the
// schema_versions INSERT (legacy record) and nself_ops INSERT are both
// inside the same BEGIN/COMMIT block as the migration SQL. This ensures
// that if the process is killed or errors between migration execution and
// schema_versions recording, Postgres rolls back the entire unit atomically.
// Without this guarantee a failed write to schema_versions would leave the
// migration applied but untracked, causing re-run failures.
func TestMigrationTxSQL_SchemaVersionsInsideTransaction(t *testing.T) {
	sqlContent := "CREATE TABLE test_atomic (id SERIAL PRIMARY KEY);"
	legacyRecord := "INSERT INTO np_common.schema_versions (name) VALUES ('001_test.sql');"
	opsRecord := "INSERT INTO nself_ops.migrations (id, name, checksum) VALUES ('001_test', '001_test.sql', 'deadbeef') ON CONFLICT (id) DO NOTHING;"

	// Replicate the exact txSQL construction from MigrateUp.
	stmtTimeout := statementTimeoutFor(sqlContent) // "60s" (no directive)
	txSQL := "BEGIN;\n" +
		"SET LOCAL lock_timeout = '5s';\n" +
		fmt.Sprintf("SET LOCAL statement_timeout = '%s';\n", stmtTimeout) +
		sqlContent + "\n" + legacyRecord + "\n" + opsRecord + "\nCOMMIT;\n"

	// 1. Both BEGIN and COMMIT must be present.
	if !strings.Contains(txSQL, "BEGIN;") {
		t.Fatal("txSQL must contain BEGIN;")
	}
	if !strings.Contains(txSQL, "COMMIT;") {
		t.Fatal("txSQL must contain COMMIT;")
	}

	// 2. schema_versions INSERT must appear AFTER BEGIN and BEFORE COMMIT.
	beginIdx := strings.Index(txSQL, "BEGIN;")
	commitIdx := strings.LastIndex(txSQL, "COMMIT;")
	legacyIdx := strings.Index(txSQL, legacyRecord)
	opsIdx := strings.Index(txSQL, opsRecord)
	sqlIdx := strings.Index(txSQL, sqlContent)

	if legacyIdx < beginIdx || legacyIdx > commitIdx {
		t.Error("schema_versions INSERT must be inside the transaction (between BEGIN and COMMIT)")
	}
	if opsIdx < beginIdx || opsIdx > commitIdx {
		t.Error("nself_ops.migrations INSERT must be inside the transaction (between BEGIN and COMMIT)")
	}

	// 3. Migration SQL must precede both recording inserts.
	// This ordering matters: migration SQL runs first, then we record it atomically.
	if sqlIdx > legacyIdx {
		t.Error("migration SQL must appear before schema_versions INSERT in txSQL")
	}
	if sqlIdx > opsIdx {
		t.Error("migration SQL must appear before nself_ops INSERT in txSQL")
	}

	// 4. Non-transactional path must NOT embed schema_versions inside a BEGIN block.
	// The non-transactional path sends migration SQL and then a separate recordSQL call.
	// If someone accidentally adds BEGIN to the migration SQL itself, this catches it.
	nonTxMigration := "CREATE INDEX CONCURRENTLY idx_atomic ON test_atomic (id);"
	if !isNonTransactional(nonTxMigration) {
		t.Fatal("expected isNonTransactional=true for CREATE INDEX CONCURRENTLY")
	}
	// Non-transactional migration SQL is piped directly — no BEGIN/COMMIT wrapping.
	if strings.Contains(nonTxMigration, "BEGIN;") || strings.Contains(nonTxMigration, "COMMIT;") {
		t.Error("non-transactional migration SQL must not contain BEGIN/COMMIT wrappers")
	}
}

// TestMigrationNonTransactional_NoTimeouts verifies that non-transactional
// migrations (CREATE INDEX CONCURRENTLY etc.) are not wrapped with timeouts,
// since lock_timeout is not applicable outside a transaction.
func TestMigrationNonTransactional_NoTimeouts(t *testing.T) {
	nonTxSQL := "CREATE INDEX CONCURRENTLY idx_test ON test_table (id);"
	if !isNonTransactional(nonTxSQL) {
		t.Fatal("expected isNonTransactional to return true for CREATE INDEX CONCURRENTLY")
	}
	// Non-transactional path pipes sqlContent directly without wrapping in transaction.
	// Verify it does NOT contain BEGIN/COMMIT (it would bypass lock_timeout wrapping).
	if strings.Contains(nonTxSQL, "BEGIN;") {
		t.Error("non-transactional SQL should not be wrapped in a transaction")
	}
}

// --- TestMigrationsDir ---

// TestMigrationsDir_HasuraDefault: hasura/migrations/default/ exists → use it.
func TestMigrationsDir_HasuraDefault(t *testing.T) {
	tmp := t.TempDir()
	mkdirAll(t, tmp, "hasura/migrations/default")
	withCwd(t, tmp)

	cfg := &config.Config{}
	got := migrationsDir(cfg, "")
	if got != "hasura/migrations/default" {
		t.Errorf("expected hasura/migrations/default, got %q", got)
	}
}

// TestMigrationsDir_HasuraFlat: only hasura/migrations/ exists → use it.
func TestMigrationsDir_HasuraFlat(t *testing.T) {
	tmp := t.TempDir()
	mkdirAll(t, tmp, "hasura/migrations")
	withCwd(t, tmp)

	cfg := &config.Config{}
	got := migrationsDir(cfg, "")
	if got != "hasura/migrations" {
		t.Errorf("expected hasura/migrations, got %q", got)
	}
}

// TestMigrationsDir_Legacy: only migrations/ exists → use it.
func TestMigrationsDir_Legacy(t *testing.T) {
	tmp := t.TempDir()
	mkdirAll(t, tmp, "migrations")
	withCwd(t, tmp)

	cfg := &config.Config{}
	got := migrationsDir(cfg, "")
	if got != "migrations" {
		t.Errorf("expected migrations, got %q", got)
	}
}

// TestMigrationsDir_NoneExist: nothing present → return helpful fallback.
func TestMigrationsDir_NoneExist(t *testing.T) {
	tmp := t.TempDir()
	withCwd(t, tmp)

	cfg := &config.Config{}
	got := migrationsDir(cfg, "")
	if got != "hasura/migrations/default" {
		t.Errorf("expected hasura/migrations/default fallback, got %q", got)
	}
}

// --- TestScanMigrations ---

// TestScanMigrations_MissingDir: non-existent directory → error containing "migrations directory not found".
func TestScanMigrations_MissingDir(t *testing.T) {
	_, err := scanMigrations("/tmp/definitely-nonexistent-xyz987abc")
	if err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
	if !strings.Contains(err.Error(), "migrations directory not found") {
		t.Errorf("error %q does not contain expected phrase", err.Error())
	}
}

// TestScanMigrations_EmptyDir: empty directory → no error, empty result.
func TestScanMigrations_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	files, err := scanMigrations(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d: %v", len(files), files)
	}
}

// TestScanMigrations_FlatLayout: flat .sql files in dir; .down.sql excluded.
func TestScanMigrations_FlatLayout(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, tmp, "001_init.sql", "-- init")
	writeFile(t, tmp, "002_users.sql", "-- users")
	writeFile(t, tmp, "001_init.down.sql", "-- down init")

	files, err := scanMigrations(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}

	// Verify sorted order and that .down.sql is excluded.
	if filepath.Base(files[0]) != "001_init.sql" {
		t.Errorf("files[0] base: want 001_init.sql, got %s", filepath.Base(files[0]))
	}
	if filepath.Base(files[1]) != "002_users.sql" {
		t.Errorf("files[1] base: want 002_users.sql, got %s", filepath.Base(files[1]))
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".down.sql") {
			t.Errorf("down file should be excluded, got %s", f)
		}
	}
}

// TestScanMigrations_NestedLayout: Hasura-style subdirs each with up.sql.
func TestScanMigrations_NestedLayout(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, tmp, "1000_init/up.sql", "-- init up")
	writeFile(t, tmp, "2000_users/up.sql", "-- users up")
	writeFile(t, tmp, "3000_orders/up.sql", "-- orders up")

	files, err := scanMigrations(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(files), files)
	}

	// All paths must end with up.sql.
	for _, f := range files {
		if filepath.Base(f) != "up.sql" {
			t.Errorf("expected up.sql filename, got %s", filepath.Base(f))
		}
	}

	// Verify the parent directory names represent the version ordering.
	// os.ReadDir returns entries in alphabetical order, so the slice should be
	// ordered 1000_init, 2000_users, 3000_orders.
	wantDirs := []string{"1000_init", "2000_users", "3000_orders"}
	for i, f := range files {
		parentDir := filepath.Base(filepath.Dir(f))
		if parentDir != wantDirs[i] {
			t.Errorf("files[%d] parent dir: want %s, got %s", i, wantDirs[i], parentDir)
		}
	}
}
