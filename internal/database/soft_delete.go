package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// SoftDeleteInfo describes a table's soft-delete status.
type SoftDeleteInfo struct {
	TableSchema  string
	TableName    string
	HasDeletedAt bool
	HasIndex     bool // index on deleted_at for performance
	HasView      bool // {table}_active view exists
}

// auditSoftDeleteSQL is the query used to detect soft-delete coverage.
const auditSoftDeleteSQL = `SELECT
    n.nspname AS table_schema,
    c.relname AS table_name,
    EXISTS(
        SELECT 1 FROM information_schema.columns ic
        WHERE ic.table_schema = n.nspname
          AND ic.table_name = c.relname
          AND ic.column_name = 'deleted_at'
    ) AS has_deleted_at,
    EXISTS(
        SELECT 1 FROM pg_indexes pi
        WHERE pi.schemaname = n.nspname
          AND pi.tablename = c.relname
          AND pi.indexdef LIKE '%deleted_at%'
    ) AS has_index,
    EXISTS(
        SELECT 1 FROM information_schema.views v
        WHERE v.table_schema = n.nspname
          AND v.table_name = c.relname || '_active'
    ) AS has_view
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r'
  AND n.nspname NOT IN ('pg_catalog','information_schema','pg_toast','hdb_catalog')
ORDER BY n.nspname, c.relname`

// AuditSoftDelete scans tables in schemas starting with "np_" or "public"
// and reports their soft-delete status.
func AuditSoftDelete(ctx context.Context, cfg *config.Config) ([]SoftDeleteInfo, error) {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	out, err := querySQL(ctx, cfg, db, auditSoftDeleteSQL)
	if err != nil {
		return nil, fmt.Errorf("audit soft delete: %w", err)
	}

	if out == "" {
		return nil, nil
	}

	var results []SoftDeleteInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// psql -tAc outputs pipe-separated columns.
		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			continue
		}
		info := SoftDeleteInfo{
			TableSchema:  strings.TrimSpace(parts[0]),
			TableName:    strings.TrimSpace(parts[1]),
			HasDeletedAt: strings.TrimSpace(parts[2]) == "t",
			HasIndex:     strings.TrimSpace(parts[3]) == "t",
			HasView:      strings.TrimSpace(parts[4]) == "t",
		}
		results = append(results, info)
	}
	return results, nil
}

// GenerateSoftDeleteMigration generates the SQL to add soft-delete support
// to a table: adds deleted_at TIMESTAMPTZ column, index, and creates a
// {table}_active view that filters WHERE deleted_at IS NULL.
// Returns up.sql and down.sql content.
func GenerateSoftDeleteMigration(schema, table string) (upSQL, downSQL string) {
	upSQL = fmt.Sprintf(
		"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;\n"+
			"CREATE INDEX IF NOT EXISTS idx_%s_deleted_at ON %s.%s (deleted_at) WHERE deleted_at IS NULL;\n"+
			"CREATE OR REPLACE VIEW %s.%s_active AS\n"+
			"  SELECT * FROM %s.%s WHERE deleted_at IS NULL;\n",
		schema, table,
		table, schema, table,
		schema, table,
		schema, table,
	)
	downSQL = fmt.Sprintf(
		"DROP VIEW IF EXISTS %s.%s_active;\n"+
			"DROP INDEX IF EXISTS %s.idx_%s_deleted_at;\n"+
			"ALTER TABLE %s.%s DROP COLUMN IF EXISTS deleted_at;\n",
		schema, table,
		schema, table,
		schema, table,
	)
	return upSQL, downSQL
}

// ApplySoftDelete runs the soft-delete migration for a specific table.
func ApplySoftDelete(ctx context.Context, cfg *config.Config, schema, table string) error {
	upSQL, _ := GenerateSoftDeleteMigration(schema, table)
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	if err := runSQLOnDB(ctx, cfg, db, upSQL); err != nil {
		return fmt.Errorf("apply soft delete to %s.%s: %w", schema, table, err)
	}
	return nil
}
