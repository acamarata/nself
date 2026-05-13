package claw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ui"
)

// MigrationEntry describes a single claw schema migration.
type MigrationEntry struct {
	// Name is the base filename of the migration (e.g. "001_init.sql").
	Name string
	// Applied is true when the migration is recorded in np_claw.schema_versions.
	Applied bool
}

// clawMigrationsDir returns the directory that holds claw plugin SQL migrations.
// It checks the plugin's installed location first, then falls back to the
// project-relative hasura/claw/migrations path used during development.
func clawMigrationsDir(cfg *config.Config) string {
	// Plugin install path.
	pluginDir := cfg.PluginSystem.Dir
	if pluginDir == "" {
		home, _ := os.UserHomeDir()
		pluginDir = filepath.Join(home, ".nself", "plugins")
	}
	installed := filepath.Join(pluginDir, "claw", "migrations")
	if _, err := os.Stat(installed); err == nil {
		return installed
	}

	// Development fallback.
	candidates := []string{
		"hasura/claw/migrations",
		"hasura/migrations/claw",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return installed // return canonical path so error messages are actionable
}

// isRollbackSQL reports whether the filename is a rollback/down migration that
// must never be applied as a forward migration.
//
// Canonical forward migrations end in exactly ".sql" (e.g. "001_init.sql").
// Rollback variants use several conventions found in this codebase:
//   - "*.down.sql"         — canonical down suffix (dot-separated)
//   - "*_down.sql"         — alternate naming used in older files
//   - "*_rollback.sql"     — explicit rollback suffix
//   - "*_rollback_*.sql"   — rollback with additional qualifier (e.g. 000_rollback_all.sql)
func isRollbackSQL(name string) bool {
	return strings.HasSuffix(name, ".down.sql") ||
		strings.HasSuffix(name, "_down.sql") ||
		strings.HasSuffix(name, "_rollback.sql") ||
		strings.Contains(name, "_rollback_")
}

// scanClawMigrations returns SQL filenames from dir, sorted lexicographically.
// Only forward migration .sql files are included; rollback/down variants are
// excluded (see isRollbackSQL for the naming patterns that are rejected).
func scanClawMigrations(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("claw migrations directory not found: %s", dir)
		}
		return nil, fmt.Errorf("scan claw migrations dir %s: %w", dir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".sql") && !isRollbackSQL(n) {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names, nil
}

// currentSchemaVersion returns the highest-numbered migration currently applied
// in np_claw.schema_versions, or an empty string if none are applied.
func currentSchemaVersion(ctx context.Context, cfg *config.Config) (string, error) {
	out, err := queryClawSQL(ctx, cfg,
		`SELECT COALESCE(MAX(name), '') FROM np_claw.schema_versions`)
	if err != nil {
		// Table may not exist yet — treat as "no migrations applied".
		return "", nil //nolint:nilerr
	}
	return strings.TrimSpace(out), nil
}

// ensureClawSchemaVersions creates the np_claw schema and schema_versions table
// if they do not already exist.
func ensureClawSchemaVersions(ctx context.Context, cfg *config.Config) error {
	sql := `CREATE SCHEMA IF NOT EXISTS np_claw;
CREATE TABLE IF NOT EXISTS np_claw.schema_versions (
  name TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	return runClawSQL(ctx, cfg, sql)
}

// appliedClawMigrations returns the set of migration names already recorded in
// np_claw.schema_versions.
func appliedClawMigrations(ctx context.Context, cfg *config.Config) (map[string]bool, error) {
	out, err := queryClawSQL(ctx, cfg, "SELECT name FROM np_claw.schema_versions ORDER BY name")
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool)
	if out == "" {
		return result, nil
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result[line] = true
		}
	}
	return result, nil
}

// MigrateResult summarises the outcome of a Migrate call.
type MigrateResult struct {
	Applied []string
	Skipped []string
}

// applyOrdered returns the (toApply, toSkip) split from a sorted list of migration
// names, given the set already applied and optional from/to version constraints.
// This pure function contains the ordering and filtering logic so it can be unit
// tested without a live database.
//
// Rules (applied in order per name):
//  1. If fromVersion is non-empty and name <= fromVersion, skip.
//  2. If toVersion is non-empty and name > toVersion, stop (remaining are not returned).
//  3. If name is in applied, skip.
//  4. Otherwise, include in toApply.
func applyOrdered(names []string, applied map[string]bool, fromVersion, toVersion string) (toApply, toSkip []string) {
	for _, name := range names {
		if fromVersion != "" && name <= fromVersion {
			toSkip = append(toSkip, name)
			continue
		}
		if toVersion != "" && name > toVersion {
			break
		}
		if applied[name] {
			toSkip = append(toSkip, name)
			continue
		}
		toApply = append(toApply, name)
	}
	return toApply, toSkip
}

// Migrate applies all pending claw schema migrations in order.
// fromVersion and toVersion constrain which migrations to run:
//   - fromVersion: skip migrations whose name is lexicographically <= fromVersion.
//     An empty string means "start from the beginning".
//   - toVersion: stop after applying the migration whose name is <= toVersion.
//     An empty string (default) means "apply everything available".
//
// Progress is printed to stdout as each migration runs.
func Migrate(ctx context.Context, cfg *config.Config, fromVersion, toVersion string) (MigrateResult, error) {
	var result MigrateResult

	if err := ensureClawSchemaVersions(ctx, cfg); err != nil {
		return result, fmt.Errorf("ensure claw schema_versions: %w", err)
	}

	dir := clawMigrationsDir(cfg)
	names, err := scanClawMigrations(dir)
	if err != nil {
		return result, err
	}

	if len(names) == 0 {
		ui.Info("No claw migrations found in " + dir)
		return result, nil
	}

	applied, err := appliedClawMigrations(ctx, cfg)
	if err != nil {
		return result, fmt.Errorf("check applied claw migrations: %w", err)
	}

	toApply, toSkip := applyOrdered(names, applied, fromVersion, toVersion)
	result.Skipped = toSkip

	for _, name := range toApply {
		sqlPath := filepath.Join(dir, name)
		data, readErr := os.ReadFile(sqlPath)
		if readErr != nil {
			return result, fmt.Errorf("read claw migration %s: %w", name, readErr)
		}

		ui.Info(fmt.Sprintf("Applying %s...", name))

		record := fmt.Sprintf(
			"\nINSERT INTO np_claw.schema_versions (name) VALUES ('%s') ON CONFLICT DO NOTHING;",
			strings.ReplaceAll(name, "'", "''"),
		)
		txSQL := "BEGIN;\n" + string(data) + record + "\nCOMMIT;\n"

		if err := runClawSQL(ctx, cfg, txSQL); err != nil {
			return result, fmt.Errorf("claw migration %s failed: %w", name, err)
		}

		ui.Success(fmt.Sprintf("Applied  %s", name))
		result.Applied = append(result.Applied, name)
	}

	return result, nil
}

// Status returns the list of all known claw migrations with their applied state.
func Status(ctx context.Context, cfg *config.Config) ([]MigrationEntry, error) {
	if err := ensureClawSchemaVersions(ctx, cfg); err != nil {
		return nil, fmt.Errorf("ensure claw schema_versions: %w", err)
	}

	dir := clawMigrationsDir(cfg)
	names, err := scanClawMigrations(dir)
	if err != nil {
		return nil, err
	}

	applied, err := appliedClawMigrations(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("check applied claw migrations: %w", err)
	}

	var entries []MigrationEntry
	for _, name := range names {
		entries = append(entries, MigrationEntry{
			Name:    name,
			Applied: applied[name],
		})
	}
	return entries, nil
}
