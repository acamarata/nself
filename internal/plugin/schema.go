package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"nself/internal/config"
)

// sanitizeSchemaName converts a plugin name to a valid Postgres identifier.
// Lowercase, hyphens become underscores, prefixed with np_.
func sanitizeSchemaName(pluginName string) string {
	name := strings.ToLower(pluginName)
	name = strings.ReplaceAll(name, "-", "_")
	return "np_" + name
}

// execPSQL runs a SQL statement inside the Postgres container via docker exec.
func execPSQL(ctx context.Context, cfg *config.Config, sql string) error {
	containerName := cfg.ProjectName + "-postgres-1"
	cmd := exec.CommandContext(ctx, "docker", "exec", containerName,
		"psql", "-U", cfg.Postgres.User, "-d", cfg.Postgres.DB,
		"-c", sql,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql exec failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// queryPSQL runs a SQL query and returns the trimmed stdout. It uses -tA
// (tuples-only, unaligned) so the result contains only raw values.
func queryPSQL(ctx context.Context, cfg *config.Config, sql string) (string, error) {
	containerName := cfg.ProjectName + "-postgres-1"
	cmd := exec.CommandContext(ctx, "docker", "exec", containerName,
		"psql", "-U", cfg.Postgres.User, "-d", cfg.Postgres.DB,
		"-tA", "-c", sql,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("psql query failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getSchemaVersion queries np_common.schema_versions for the latest recorded
// version of the named plugin. Returns (0, nil) if no version row exists.
func getSchemaVersion(ctx context.Context, cfg *config.Config, pluginName string) (int, error) {
	sql := fmt.Sprintf(
		"SELECT COALESCE(MAX(version),0) FROM np_common.schema_versions WHERE plugin = '%s';",
		sanitizeSchemaName(pluginName),
	)
	out, err := queryPSQL(ctx, cfg, sql)
	if err != nil {
		// Table may not exist yet on a fresh instance; treat as version 0.
		return 0, nil
	}
	if out == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(out)
	if err != nil {
		return 0, fmt.Errorf("parsing schema version for %q: %w", pluginName, err)
	}
	return v, nil
}

// recordSchemaVersion inserts a version row into np_common.schema_versions
// for the given plugin. Duplicate (plugin, version) pairs are silently ignored.
func recordSchemaVersion(ctx context.Context, cfg *config.Config, pluginName string, version int) error {
	sql := fmt.Sprintf(
		"INSERT INTO np_common.schema_versions (plugin, version) VALUES ('%s', %d) ON CONFLICT DO NOTHING;",
		sanitizeSchemaName(pluginName), version,
	)
	return execPSQL(ctx, cfg, sql)
}

// schemaVersion is the current version of the plugin schema setup logic.
// Bump this when the schema creation steps change in a meaningful way.
const schemaVersion = 1

// CreatePluginSchema creates an isolated Postgres schema and role for a plugin.
// Schema name: np_{name} (lowercase, hyphens to underscores).
// Role name: np_{name}_role.
// All statements are idempotent and safe to run multiple times.
//
// Before executing, it checks np_common.schema_versions for the plugin. If the
// recorded version matches schemaVersion the function returns early (already
// applied). After successful creation it records the version.
func createPluginSchema(ctx context.Context, cfg *config.Config, pluginName string) error {
	schema := sanitizeSchemaName(pluginName)
	role := schema + "_role"

	// Ensure np_common schema and schema_versions tracking table exist first,
	// so the version check below has a table to query.
	commonSQL := `CREATE SCHEMA IF NOT EXISTS np_common;
CREATE TABLE IF NOT EXISTS np_common.schema_versions (
  plugin TEXT NOT NULL,
  version INT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (plugin, version)
);`
	if err := execPSQL(ctx, cfg, commonSQL); err != nil {
		return fmt.Errorf("creating np_common.schema_versions: %w", err)
	}

	// Check whether this plugin's schema has already been applied at the
	// current version. If so, skip all remaining work.
	existingVer, err := getSchemaVersion(ctx, cfg, pluginName)
	if err != nil {
		return fmt.Errorf("checking schema version for %q: %w", pluginName, err)
	}
	if existingVer >= schemaVersion {
		return nil
	}

	// Create role if it does not exist (idempotent via DO block).
	roleSQL := fmt.Sprintf(`DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '%s') THEN
    CREATE ROLE %s NOLOGIN;
  END IF;
END $$;`, role, role)

	if err := execPSQL(ctx, cfg, roleSQL); err != nil {
		return fmt.Errorf("creating role %s: %w", role, err)
	}

	// Create schema if it does not exist.
	schemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schema)
	if err := execPSQL(ctx, cfg, schemaSQL); err != nil {
		return fmt.Errorf("creating schema %s: %w", schema, err)
	}

	// Grant usage and create permissions on the schema to the role.
	grantSQL := fmt.Sprintf(
		"GRANT USAGE ON SCHEMA %s TO %s; GRANT CREATE ON SCHEMA %s TO %s;",
		schema, role, schema, role,
	)
	if err := execPSQL(ctx, cfg, grantSQL); err != nil {
		return fmt.Errorf("granting permissions on %s: %w", schema, err)
	}

	// Set the role's default search_path to its own schema plus public.
	pathSQL := fmt.Sprintf("ALTER ROLE %s SET search_path = %s, public;", role, schema)
	if err := execPSQL(ctx, cfg, pathSQL); err != nil {
		return fmt.Errorf("setting search_path for %s: %w", role, err)
	}

	// Record successful schema creation so subsequent calls skip the work.
	if err := recordSchemaVersion(ctx, cfg, pluginName, schemaVersion); err != nil {
		return fmt.Errorf("recording schema version for %q: %w", pluginName, err)
	}

	return nil
}

// DropPluginSchema removes a plugin's Postgres schema and role.
// Safe to call even if the schema or role does not exist.
func dropPluginSchema(ctx context.Context, cfg *config.Config, pluginName string) error {
	schema := sanitizeSchemaName(pluginName)
	role := schema + "_role"

	dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schema)
	if err := execPSQL(ctx, cfg, dropSchemaSQL); err != nil {
		return fmt.Errorf("dropping schema %s: %w", schema, err)
	}

	dropRoleSQL := fmt.Sprintf("DROP ROLE IF EXISTS %s;", role)
	if err := execPSQL(ctx, cfg, dropRoleSQL); err != nil {
		return fmt.Errorf("dropping role %s: %w", role, err)
	}

	return nil
}
