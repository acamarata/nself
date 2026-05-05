package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// RequiredColumn defines a column that should exist on all np_* tables.
type RequiredColumn struct {
	Name        string
	DataType    string
	Required    bool // if false, column is recommended but not required
	Description string
}

// SchemaDriftResult describes drift for a single table.
type SchemaDriftResult struct {
	TableSchema    string
	TableName      string
	MissingColumns []RequiredColumn
	PresentColumns []string
	DriftScore     int // 0-100, higher = more drift
}

// DriftSummary returns a high-level summary: total tables, compliant, drifted, score.
type DriftSummary struct {
	TotalTables     int
	CompliantTables int
	DriftedTables   int
	MissingRequired int // tables missing at least one REQUIRED column
	OverallScore    int // 0-100
}

// Theme25RequiredColumns are the 5 columns mandated by Theme 25.
var Theme25RequiredColumns = []RequiredColumn{
	{Name: "id", DataType: "uuid", Required: true, Description: "UUID primary key"},
	{Name: "created_at", DataType: "timestamp with time zone", Required: true, Description: "creation timestamp"},
	{Name: "updated_at", DataType: "timestamp with time zone", Required: true, Description: "last update timestamp"},
	{Name: "user_id", DataType: "uuid", Required: false, Description: "owner reference to auth.users"},
	{Name: "deleted_at", DataType: "timestamp with time zone", Required: false, Description: "soft-delete timestamp"},
}

const schemaDriftSQL = `SELECT
    c.table_schema,
    c.table_name,
    c.column_name,
    c.data_type
FROM information_schema.columns c
WHERE c.table_schema LIKE 'np_%'
ORDER BY c.table_schema, c.table_name, c.column_name`

// ScanSchemaDrift scans all np_* tables for the 5 required Theme 25 columns.
// Only tables in schemas starting with "np_" are checked.
func ScanSchemaDrift(ctx context.Context, cfg *config.Config) ([]SchemaDriftResult, error) {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	out, err := querySQL(ctx, cfg, db, schemaDriftSQL)
	if err != nil {
		return nil, fmt.Errorf("scan schema drift: %w", err)
	}

	// Group columns by (schema, table).
	type tableKey struct {
		schema string
		table  string
	}
	// Preserve insertion order using a slice of keys alongside the map.
	var order []tableKey
	seen := make(map[tableKey]bool)
	tableColumns := make(map[tableKey][]string)

	if out != "" {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) != 4 {
				continue
			}
			schema := strings.TrimSpace(parts[0])
			table := strings.TrimSpace(parts[1])
			column := strings.TrimSpace(parts[2])
			// data_type is parts[3] — not used for grouping but column name is.

			key := tableKey{schema: schema, table: table}
			if !seen[key] {
				seen[key] = true
				order = append(order, key)
			}
			tableColumns[key] = append(tableColumns[key], column)
		}
	}

	var results []SchemaDriftResult

	for _, key := range order {
		cols := tableColumns[key]
		colSet := make(map[string]bool, len(cols))
		for _, c := range cols {
			colSet[c] = true
		}

		var missing []RequiredColumn
		for _, req := range Theme25RequiredColumns {
			if !colSet[req.Name] {
				missing = append(missing, req)
			}
		}

		// DriftScore: missing_required*30 + missing_recommended*10, capped at 100.
		score := 0
		for _, m := range missing {
			if m.Required {
				score += 30
			} else {
				score += 10
			}
		}
		if score > 100 {
			score = 100
		}

		results = append(results, SchemaDriftResult{
			TableSchema:    key.schema,
			TableName:      key.table,
			MissingColumns: missing,
			PresentColumns: cols,
			DriftScore:     score,
		})
	}

	return results, nil
}

// GenerateDriftMigration generates SQL to add missing columns to a table.
// Returns up.sql and down.sql content.
func GenerateDriftMigration(result SchemaDriftResult) (upSQL, downSQL string) {
	schema := result.TableSchema
	table := result.TableName

	var upLines []string
	var downLines []string

	for _, col := range result.MissingColumns {
		var addStmt string
		switch col.Name {
		case "id":
			addStmt = fmt.Sprintf(
				"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS id UUID DEFAULT gen_random_uuid() PRIMARY KEY;",
				schema, table,
			)
		case "created_at":
			addStmt = fmt.Sprintf(
				"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();",
				schema, table,
			)
		case "updated_at":
			addStmt = fmt.Sprintf(
				"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();",
				schema, table,
			)
		case "user_id":
			addStmt = fmt.Sprintf(
				"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS user_id UUID;",
				schema, table,
			)
		case "deleted_at":
			addStmt = fmt.Sprintf(
				"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;",
				schema, table,
			)
		default:
			// Unknown column in the required list; skip safely.
			continue
		}

		upLines = append(upLines, addStmt)
		downLines = append(downLines, fmt.Sprintf(
			"ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;",
			schema, table, col.Name,
		))
	}

	upSQL = strings.Join(upLines, "\n")
	downSQL = strings.Join(downLines, "\n")
	return upSQL, downSQL
}

// SummarizeDrift returns a high-level summary across all drift results.
func SummarizeDrift(results []SchemaDriftResult) DriftSummary {
	summary := DriftSummary{
		TotalTables:  len(results),
		OverallScore: 100,
	}

	for _, r := range results {
		if len(r.MissingColumns) == 0 {
			summary.CompliantTables++
		} else {
			summary.DriftedTables++
		}

		for _, m := range r.MissingColumns {
			if m.Required {
				summary.MissingRequired++
				break
			}
		}
	}

	if summary.TotalTables > 0 {
		summary.OverallScore = 100 - (summary.DriftedTables*100)/summary.TotalTables
	}

	return summary
}
