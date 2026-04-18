package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// FKIndexInfo describes a foreign key constraint and whether it has an index.
type FKIndexInfo struct {
	TableSchema    string
	TableName      string
	ColumnName     string
	ForeignSchema  string
	ForeignTable   string
	ForeignColumn  string
	HasIndex       bool
	ConstraintName string
}

// auditFKIndexSQL finds all FK columns and checks whether a supporting index exists.
const auditFKIndexSQL = `SELECT
    tc.table_schema,
    tc.table_name,
    kcu.column_name,
    ccu.table_schema AS foreign_schema,
    ccu.table_name AS foreign_table,
    ccu.column_name AS foreign_column,
    tc.constraint_name,
    EXISTS (
        SELECT 1
        FROM pg_index pi
        JOIN pg_class c ON c.oid = pi.indrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(pi.indkey)
        WHERE n.nspname = tc.table_schema
          AND c.relname = tc.table_name
          AND a.attname = kcu.column_name
    ) AS has_index
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
JOIN information_schema.referential_constraints rc
    ON tc.constraint_name = rc.constraint_name
    AND tc.table_schema = rc.constraint_schema
JOIN information_schema.constraint_column_usage ccu
    ON rc.unique_constraint_name = ccu.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema NOT IN ('pg_catalog','information_schema','pg_toast','hdb_catalog')
ORDER BY tc.table_schema, tc.table_name, kcu.column_name`

// AuditFKIndexes finds all foreign key columns and reports whether each has a supporting index.
// Only tables in np_* and public schemas are checked.
func AuditFKIndexes(ctx context.Context, cfg *config.Config) ([]FKIndexInfo, error) {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	out, err := querySQL(ctx, cfg, db, auditFKIndexSQL)
	if err != nil {
		return nil, fmt.Errorf("audit fk indexes: %w", err)
	}

	if out == "" {
		return nil, nil
	}

	var results []FKIndexInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// psql -tAc outputs pipe-separated columns.
		parts := strings.Split(line, "|")
		if len(parts) != 8 {
			continue
		}
		info := FKIndexInfo{
			TableSchema:    strings.TrimSpace(parts[0]),
			TableName:      strings.TrimSpace(parts[1]),
			ColumnName:     strings.TrimSpace(parts[2]),
			ForeignSchema:  strings.TrimSpace(parts[3]),
			ForeignTable:   strings.TrimSpace(parts[4]),
			ForeignColumn:  strings.TrimSpace(parts[5]),
			ConstraintName: strings.TrimSpace(parts[6]),
			HasIndex:       strings.TrimSpace(parts[7]) == "t",
		}
		results = append(results, info)
	}
	return results, nil
}

// GenerateFKIndexSQL generates CREATE INDEX SQL for a missing FK index.
func GenerateFKIndexSQL(info FKIndexInfo) string {
	idxName := fmt.Sprintf("idx_%s_%s", info.TableName, info.ColumnName)
	return fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON %s.%s (%s);",
		idxName, info.TableSchema, info.TableName, info.ColumnName,
	)
}

// ApplyFKIndexes creates missing indexes for all FK columns lacking one.
// Returns count of indexes created and any errors.
func ApplyFKIndexes(ctx context.Context, cfg *config.Config, findings []FKIndexInfo) (created int, errs []error) {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	for _, info := range findings {
		if info.HasIndex {
			continue
		}
		sql := GenerateFKIndexSQL(info)
		if err := runSQLOnDB(ctx, cfg, db, sql); err != nil {
			errs = append(errs, fmt.Errorf("create index for %s.%s(%s): %w", info.TableSchema, info.TableName, info.ColumnName, err))
			continue
		}
		created++
	}
	return created, errs
}
