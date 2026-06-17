package database

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	// pq registers the "postgres" driver used by the embedded-PG SQL path.
	_ "github.com/lib/pq"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// migrationNameRegex restricts migration filenames to safe characters.
// Allows letters, digits, dots, underscores, and hyphens. Prevents
// names with embedded quotes, semicolons, or control characters.
var migrationNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,255}$`)

// validateMigrationName fails closed on migration filenames that could
// escape a SQL string literal even after escapeSQL doubling.
func validateMigrationName(name string) error {
	if !migrationNameRegex.MatchString(name) {
		return fmt.Errorf("invalid migration name %q: only [a-zA-Z0-9._-] allowed", name)
	}
	return nil
}

// MigrationStatus describes the state of a single migration file.
type MigrationStatus struct {
	Name      string
	Applied   bool
	Timestamp time.Time
}

// querySQLEmbedded executes a query against the embedded pglite instance via its
// Unix-domain socket. Returns the first column of the first row as a
// newline-joined string (mirrors the `-tAc` psql output format).
func querySQLEmbedded(ctx context.Context, dsn string, query string) (string, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", fmt.Errorf("embedded sql.Open: %w", err)
	}
	defer db.Close() //nolint:errcheck

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("embedded query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var lines []string
	for rows.Next() {
		var val string
		if scanErr := rows.Scan(&val); scanErr != nil {
			return "", fmt.Errorf("embedded row scan: %w", scanErr)
		}
		lines = append(lines, val)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("embedded rows: %w", err)
	}
	return strings.Join(lines, "\n"), nil
}

// pipeSQLEmbedded executes raw SQL text against the embedded pglite instance
// via its Unix-domain socket. Mirrors pipeSQLToContainer for the embedded path.
func pipeSQLEmbedded(ctx context.Context, dsn string, sqlText string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("embedded sql.Open: %w", err)
	}
	defer db.Close() //nolint:errcheck

	if _, err := db.ExecContext(ctx, sqlText); err != nil {
		return fmt.Errorf("embedded exec: %w", err)
	}
	return nil
}

// querySQL executes a SQL query inside the postgres container and returns stdout.
// Unlike runSQL from init.go, this captures and returns the output text.
// When cfg.EmbeddedPG is true, the query is routed through the pglite UDS instead.
func querySQL(ctx context.Context, cfg *config.Config, database string, sqlText string) (string, error) {
	if cfg.EmbeddedPG {
		dsn := cfg.EmbeddedPGDatabaseURL(embeddedPGRuntimeDir(cfg))
		return querySQLEmbedded(ctx, dsn, sqlText)
	}

	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", database, "-tAc", sqlText,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("psql: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// pipeSQLToContainer pipes raw SQL text into psql inside the postgres container.
// When cfg.EmbeddedPG is true, the SQL is executed against the pglite UDS instead.
func pipeSQLToContainer(ctx context.Context, cfg *config.Config, sqlText string) error {
	if cfg.EmbeddedPG {
		dsn := cfg.EmbeddedPGDatabaseURL(embeddedPGRuntimeDir(cfg))
		return pipeSQLEmbedded(ctx, dsn, sqlText)
	}

	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	args := []string{
		"exec", "-i", container,
		"psql",
		"-U", user,
		"-d", db,
		"-v", "ON_ERROR_STOP=1",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = strings.NewReader(sqlText)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// embeddedPGRuntimeDir returns the runtime directory used by the pglite/wasmtime
// embedded runtime. It reads NSELF_RUNTIME_DIR first (set by `nself start --embedded-pg`)
// then falls back to the project-local default.
func embeddedPGRuntimeDir(cfg *config.Config) string {
	if dir := os.Getenv("NSELF_RUNTIME_DIR"); dir != "" {
		return dir
	}
	// Fall back to project-local default: $PWD/.nself/embedded-pg
	wd, _ := os.Getwd()
	return wd + "/.nself/embedded-pg"
}

// ensureSchemaVersions creates the np_common.schema_versions table if it does not exist.
func ensureSchemaVersions(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	sql := `CREATE SCHEMA IF NOT EXISTS np_common; CREATE TABLE IF NOT EXISTS np_common.schema_versions (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`
	return runSQLOnDB(ctx, cfg, db, sql)
}

// migrationsDir returns the directory to scan for migration SQL files.
// If plugin is non-empty, it uses the plugin-specific migrations path.
// For non-plugin migrations, it detects the layout in order:
//  1. hasura/migrations/default/ (standard Hasura layout)
//  2. hasura/migrations/         (flat Hasura layout)
//  3. migrations/                (legacy fallback)
//
// If none exist on disk, it returns "hasura/migrations/default/" so error
// messages point the user at the canonical location.
func migrationsDir(cfg *config.Config, plugin string) string {
	if plugin != "" {
		pluginDir := cfg.PluginSystem.Dir
		if pluginDir == "" {
			home, _ := os.UserHomeDir()
			pluginDir = filepath.Join(home, ".nself", "plugins")
		}
		return filepath.Join(pluginDir, plugin, "migrations")
	}

	candidates := []string{
		"hasura/migrations/default",
		"hasura/migrations",
		"migrations",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "hasura/migrations/default"
}

// scanMigrations returns sorted SQL file paths from the given directory.
// It handles two layouts:
//   - Flat: SQL files directly in dir (excludes *.down.sql)
//   - Nested (Hasura): subdirectories each containing an up.sql file
//
// Returns an error if the directory does not exist.
func scanMigrations(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("migrations directory not found: %s", dir)
		}
		return nil, fmt.Errorf("scan migrations dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			// Nested layout: look for <entry>/up.sql
			upPath := filepath.Join(dir, e.Name(), "up.sql")
			if _, statErr := os.Stat(upPath); statErr == nil {
				files = append(files, upPath)
			}
		} else if strings.HasSuffix(e.Name(), ".sql") && !strings.HasSuffix(e.Name(), ".down.sql") {
			// Flat layout: SQL file directly in dir
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return filepath.Base(files[i]) < filepath.Base(files[j])
	})
	return files, nil
}

// appliedMigrations returns the set of migration names already recorded.
func appliedMigrations(ctx context.Context, cfg *config.Config) (map[string]time.Time, error) {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	out, err := querySQL(ctx, cfg, db, "SELECT name || '|' || applied_at FROM np_common.schema_versions ORDER BY applied_at")
	if err != nil {
		return nil, err
	}

	result := make(map[string]time.Time)
	if out == "" {
		return result, nil
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		name := parts[0]
		var ts time.Time
		if len(parts) == 2 {
			ts, _ = time.Parse(time.RFC3339, parts[1])
		}
		result[name] = ts
	}
	return result, nil
}

// isNonTransactional checks if a migration SQL contains statements that cannot
// run inside a transaction (e.g., CREATE INDEX CONCURRENTLY).
func isNonTransactional(sql string) bool {
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "CREATE INDEX CONCURRENTLY") ||
		strings.Contains(upper, "DROP INDEX CONCURRENTLY") ||
		strings.Contains(upper, "REINDEX CONCURRENTLY") {
		return true
	}
	// Only `ALTER TYPE ... ADD VALUE` cannot run in a transaction. Other ALTER
	// TYPE forms (RENAME, OWNER TO, SET SCHEMA) are transaction-safe and must
	// keep their atomicity, so do not match those.
	return alterTypeAddValueRegex.MatchString(upper)
}

// alterTypeAddValueRegex matches `ALTER TYPE ... ADD VALUE`, the only ALTER TYPE
// form that PostgreSQL refuses to run inside a transaction block.
var alterTypeAddValueRegex = regexp.MustCompile(`ALTER\s+TYPE\b[\s\S]*?\bADD\s+VALUE\b`)

// statementTimeoutDirective lets a migration opt out of (or change) the default
// 60s statement_timeout, e.g. for large data backfills:
//
//	-- nself:statement-timeout=0       (disable; no limit)
//	-- nself:statement-timeout=600s    (raise to 10 minutes)
var statementTimeoutDirective = regexp.MustCompile(`(?i)--\s*nself:statement-timeout\s*=\s*([0-9]+[a-z]*)`)

// statementTimeoutFor returns the statement_timeout value a migration should
// use. Defaults to "60s"; a migration may override it via the directive comment.
func statementTimeoutFor(sql string) string {
	if m := statementTimeoutDirective.FindStringSubmatch(sql); m != nil {
		return m[1]
	}
	return "60s"
}

// MigrateUp applies all pending migrations from the migrations directory.
// If plugin is non-empty, only that plugin's migrations are applied.
// Uses advisory locks to prevent concurrent runs, records SHA-256 checksums,
// and detects non-transactional statements (CREATE INDEX CONCURRENTLY) to
// run them outside a transaction.
// Returns the count of migrations applied.
func MigrateUp(ctx context.Context, cfg *config.Config, plugin string) (int, error) {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return 0, fmt.Errorf("ensure schema_versions: %w", err)
	}
	if err := ensureMigrationsTable(ctx, cfg); err != nil {
		return 0, fmt.Errorf("ensure migrations table: %w", err)
	}

	// Acquire advisory lock to prevent concurrent migrations.
	if err := acquireAdvisoryLock(ctx, cfg); err != nil {
		return 0, err
	}
	defer releaseAdvisoryLock(ctx, cfg) //nolint:errcheck

	dir := migrationsDir(cfg, plugin)
	files, err := scanMigrations(dir)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		return 0, nil
	}

	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		return 0, fmt.Errorf("check applied migrations: %w", err)
	}

	count := 0
	for _, f := range files {
		name := filepath.Base(f)
		if err := validateMigrationName(name); err != nil {
			return count, err
		}
		if _, ok := applied[name]; ok {
			continue
		}

		data, readErr := os.ReadFile(f)
		if readErr != nil {
			return count, fmt.Errorf("read migration %s: %w", name, readErr)
		}

		// Compute checksum for the ops table.
		checksum, _ := checksumBytes(data)
		if !sha256HexRegex.MatchString(checksum) {
			return count, fmt.Errorf("unexpected checksum format for %s", name)
		}
		migrationID := extractMigrationID(f)
		if err := validateMigrationName(migrationID); err != nil {
			return count, fmt.Errorf("migration id from %s: %w", name, err)
		}

		// Record in legacy schema_versions for backward compat.
		legacyRecord := fmt.Sprintf("INSERT INTO np_common.schema_versions (name) VALUES ('%s');",
			strings.ReplaceAll(name, "'", "''"))

		// Record in nself_ops.migrations with checksum.
		opsRecord := fmt.Sprintf(
			"INSERT INTO nself_ops.migrations (id, name, checksum) VALUES ('%s', '%s', '%s') ON CONFLICT (id) DO NOTHING;",
			strings.ReplaceAll(migrationID, "'", "''"),
			strings.ReplaceAll(name, "'", "''"),
			checksum,
		)

		sqlContent := string(data)

		if isNonTransactional(sqlContent) {
			// Run non-transactional migrations outside a transaction.
			if err := pipeSQLToContainer(ctx, cfg, sqlContent); err != nil {
				return count, fmt.Errorf("migration %s: %w: %v", name, errs.ErrMigrationFailed, err)
			}
			// Record separately (these succeed independently).
			recordSQL := legacyRecord + "\n" + opsRecord
			if err := pipeSQLToContainer(ctx, cfg, recordSQL); err != nil {
				return count, fmt.Errorf("record migration %s: %w", name, err)
			}
		} else {
			// DEP-02: Set lock_timeout and statement_timeout at session start to prevent
			// long-blocking schema changes from stalling production deployments.
			// lock_timeout=5s aborts if the migration cannot acquire the lock quickly.
			// statement_timeout defaults to 60s but a migration with a large data
			// backfill can override it via `-- nself:statement-timeout=...`.
			stmtTimeout := statementTimeoutFor(sqlContent)
			txSQL := "BEGIN;\n" +
				"SET LOCAL lock_timeout = '5s';\n" +
				fmt.Sprintf("SET LOCAL statement_timeout = '%s';\n", stmtTimeout) +
				sqlContent + "\n" + legacyRecord + "\n" + opsRecord + "\nCOMMIT;\n"
			if err := pipeSQLToContainer(ctx, cfg, txSQL); err != nil {
				return count, fmt.Errorf("migration %s: %w: %v", name, errs.ErrMigrationFailed, err)
			}
		}
		count++
	}
	return count, nil
}

// MigrateDown reverts the most recently applied migration.
// It looks for a corresponding .down.sql file next to the original migration.
func MigrateDown(ctx context.Context, cfg *config.Config) error {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return fmt.Errorf("ensure schema_versions: %w", err)
	}

	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	// Find the most recent migration.
	out, err := querySQL(ctx, cfg, db, "SELECT name FROM np_common.schema_versions ORDER BY applied_at DESC LIMIT 1")
	if err != nil {
		return fmt.Errorf("query latest migration: %w", err)
	}
	if out == "" {
		return fmt.Errorf("no migrations to revert")
	}
	name := strings.TrimSpace(out)
	if err := validateMigrationName(name); err != nil {
		return fmt.Errorf("migration name from schema_versions: %w", err)
	}

	// Derive the down file path: foo.sql -> foo.down.sql
	downName := strings.TrimSuffix(name, ".sql") + ".down.sql"
	downPath := filepath.Join(migrationsDir(cfg, ""), downName)

	data, readErr := os.ReadFile(downPath)
	if readErr != nil {
		return fmt.Errorf("down migration not found: %s: %w", downPath, readErr)
	}

	remove := fmt.Sprintf("DELETE FROM np_common.schema_versions WHERE name = '%s';",
		strings.ReplaceAll(name, "'", "''"))

	txSQL := "BEGIN;\n" + string(data) + "\n" + remove + "\nCOMMIT;\n"

	if err := pipeSQLToContainer(ctx, cfg, txSQL); err != nil {
		return fmt.Errorf("revert %s: %w: %v", name, errs.ErrMigrationFailed, err)
	}
	return nil
}

// PendingMigrations returns the list of migration names that have not yet been applied.
func PendingMigrations(ctx context.Context, cfg *config.Config, plugin string) ([]string, error) {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return nil, fmt.Errorf("ensure schema_versions: %w", err)
	}
	dir := migrationsDir(cfg, plugin)
	files, err := scanMigrations(dir)
	if err != nil {
		return nil, err
	}
	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("check applied migrations: %w", err)
	}
	var pending []string
	for _, f := range files {
		name := filepath.Base(f)
		if _, ok := applied[name]; !ok {
			pending = append(pending, name)
		}
	}
	return pending, nil
}

// ApplyFile applies a single external SQL migration file and records it in
// schema_versions by filename + SHA-256 checksum (G-008).
//
// Double-apply protection: if a file with the same filename is already present
// in schema_versions, the function logs the skip and returns (true, nil).
// The checksum is stored in nself_ops.migrations for audit purposes.
//
// This enables plugin-claw external RLS migrations to be applied via CLI
// without requiring 'nself db shell' as a workaround.
func ApplyFile(ctx context.Context, cfg *config.Config, filePath string) (skipped bool, err error) {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return false, fmt.Errorf("ensure schema_versions: %w", err)
	}
	if err := ensureMigrationsTable(ctx, cfg); err != nil {
		return false, fmt.Errorf("ensure migrations table: %w", err)
	}

	name := filepath.Base(filePath)
	if err := validateMigrationName(name); err != nil {
		return false, err
	}

	data, readErr := os.ReadFile(filePath)
	if readErr != nil {
		return false, fmt.Errorf("read migration file %s: %w", name, readErr)
	}

	checksum, csErr := checksumBytes(data)
	if csErr != nil {
		return false, fmt.Errorf("compute checksum for %s: %w", name, csErr)
	}
	if !sha256HexRegex.MatchString(checksum) {
		return false, fmt.Errorf("unexpected checksum format for %s", name)
	}

	// Check if this filename is already in schema_versions.
	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		return false, fmt.Errorf("check applied migrations: %w", err)
	}
	if _, ok := applied[name]; ok {
		// Already applied — warn and skip (do not error).
		return true, nil
	}

	migrationID := extractMigrationID(filePath)
	if migrationID == "" {
		// Fall back to the full filename as ID when no timestamp prefix is found.
		migrationID = name
	}
	if err := validateMigrationName(migrationID); err != nil {
		return false, fmt.Errorf("migration id from %s: %w", name, err)
	}

	legacyRecord := fmt.Sprintf("INSERT INTO np_common.schema_versions (name) VALUES ('%s');",
		strings.ReplaceAll(name, "'", "''"))
	opsRecord := fmt.Sprintf(
		"INSERT INTO nself_ops.migrations (id, name, checksum) VALUES ('%s', '%s', '%s') ON CONFLICT (id) DO NOTHING;",
		strings.ReplaceAll(migrationID, "'", "''"),
		strings.ReplaceAll(name, "'", "''"),
		checksum,
	)

	sqlContent := string(data)

	if isNonTransactional(sqlContent) {
		if err := pipeSQLToContainer(ctx, cfg, sqlContent); err != nil {
			return false, fmt.Errorf("migration %s: %w: %v", name, errs.ErrMigrationFailed, err)
		}
		recordSQL := legacyRecord + "\n" + opsRecord
		if err := pipeSQLToContainer(ctx, cfg, recordSQL); err != nil {
			return false, fmt.Errorf("record migration %s: %w", name, err)
		}
	} else {
		txSQL := "BEGIN;\n" +
			"SET LOCAL lock_timeout = '5s';\n" +
			"SET LOCAL statement_timeout = '60s';\n" +
			sqlContent + "\n" + legacyRecord + "\n" + opsRecord + "\nCOMMIT;\n"
		if err := pipeSQLToContainer(ctx, cfg, txSQL); err != nil {
			return false, fmt.Errorf("migration %s: %w: %v", name, errs.ErrMigrationFailed, err)
		}
	}
	return false, nil
}

// MigrateUpDir applies all .sql files in the given directory in lexicographic
// order, skipping any already recorded in schema_versions (idempotent).
// This is the --migration-dir companion to ApplyFile (G-008).
func MigrateUpDir(ctx context.Context, cfg *config.Config, dir string) (int, error) {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return 0, fmt.Errorf("ensure schema_versions: %w", err)
	}
	if err := ensureMigrationsTable(ctx, cfg); err != nil {
		return 0, fmt.Errorf("ensure migrations table: %w", err)
	}

	files, err := scanMigrations(dir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, f := range files {
		skipped, applyErr := ApplyFile(ctx, cfg, f)
		if applyErr != nil {
			return count, applyErr
		}
		if !skipped {
			count++
		}
	}
	return count, nil
}

// ApplyDir is an alias for MigrateUpDir with the name expected by the task spec
// (G-008). Both functions apply all .sql files in the given directory in
// lexicographic order, skipping files already recorded in schema_versions.
func ApplyDir(ctx context.Context, cfg *config.Config, dirPath string) (int, error) {
	return MigrateUpDir(ctx, cfg, dirPath)
}

// MigrateStatus returns the status of all known migrations (applied and pending).
// It merges on-disk migration files with the schema_versions table, so orphaned
// migrations (applied but no longer on disk) are also reported.
func MigrateStatus(ctx context.Context, cfg *config.Config) ([]MigrationStatus, error) {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return nil, fmt.Errorf("ensure schema_versions: %w", err)
	}

	files, err := scanMigrations(migrationsDir(cfg, ""))
	if err != nil {
		return nil, err
	}

	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("check applied migrations: %w", err)
	}

	var statuses []MigrationStatus
	onDisk := make(map[string]bool)

	for _, f := range files {
		name := filepath.Base(f)
		onDisk[name] = true
		ts, ok := applied[name]
		statuses = append(statuses, MigrationStatus{
			Name:      name,
			Applied:   ok,
			Timestamp: ts,
		})
	}

	// Include any applied migrations not found on disk (orphans).
	for name, ts := range applied {
		if !onDisk[name] {
			statuses = append(statuses, MigrationStatus{
				Name:      name,
				Applied:   true,
				Timestamp: ts,
			})
		}
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})
	return statuses, nil
}
