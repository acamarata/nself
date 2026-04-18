package database

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

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

// querySQL executes a SQL query inside the postgres container and returns stdout.
// Unlike runSQL from init.go, this captures and returns the output text.
func querySQL(ctx context.Context, cfg *config.Config, database string, sql string) (string, error) {
	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", database, "-tAc", sql,
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
func pipeSQLToContainer(ctx context.Context, cfg *config.Config, sql string) error {
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
	cmd.Stdin = strings.NewReader(sql)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
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
	return strings.Contains(upper, "CREATE INDEX CONCURRENTLY") ||
		strings.Contains(upper, "DROP INDEX CONCURRENTLY") ||
		strings.Contains(upper, "REINDEX CONCURRENTLY") ||
		strings.Contains(upper, "ALTER TYPE") // ADD VALUE in enums
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
			txSQL := "BEGIN;\n" + sqlContent + "\n" + legacyRecord + "\n" + opsRecord + "\nCOMMIT;\n"
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
