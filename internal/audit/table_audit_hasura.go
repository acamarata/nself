package audit

// Purpose: per-table row-count collection (auditTable), Hasura select_permissions filter detection, and project-root discovery backing RunTableAuditWithConfig in table_audit.go.
// Inputs: a *sql.DB, table name, and the project root's Hasura metadata directory.
// Outputs: a TableAuditResult per table and a HasuraFilterStatus lookup.
// Constraints: split out of table_audit.go as a pure move (CLI-R12); no behavior change.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// auditTable collects row counts per source_account_id for a single np_* table.
func auditTable(ctx context.Context, db *sql.DB, table string) (TableAuditResult, error) {
	result := TableAuditResult{Table: table}

	ok, err := hasColumn(ctx, db, table, "source_account_id")
	if err != nil {
		return result, err
	}
	result.HasSourceAccountID = ok

	if !ok {
		// Cannot break down by account — count total rows for reference.
		countQ := fmt.Sprintf("SELECT COUNT(*) FROM %q", table) //nolint:gosec
		if err := db.QueryRowContext(ctx, countQ).Scan(&result.TotalRows); err != nil {
			return result, fmt.Errorf("count rows in %s: %w", table, err)
		}
		result.Warning = "no source_account_id column — cross-app isolation unknown"
		return result, nil
	}

	// Aggregate by source_account_id.
	aggQ := fmt.Sprintf( //nolint:gosec
		"SELECT source_account_id, COUNT(*) FROM %q GROUP BY source_account_id ORDER BY source_account_id",
		table,
	)
	rows, err := db.QueryContext(ctx, aggQ)
	if err != nil {
		return result, fmt.Errorf("aggregate %s by source_account_id: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	result.Accounts = make(map[string]int64)
	for rows.Next() {
		var acct string
		var cnt int64
		if err := rows.Scan(&acct, &cnt); err != nil {
			return result, fmt.Errorf("scan account row for %s: %w", table, err)
		}
		result.Accounts[acct] = cnt
		result.TotalRows += cnt
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate account rows for %s: %w", table, err)
	}

	if len(result.Accounts) > 1 {
		result.Warning = fmt.Sprintf(
			"multi-account contamination: %d source_account_id values found — verify plugin isolation",
			len(result.Accounts),
		)
	}

	return result, nil
}

// --- Hasura metadata helpers ---

// hasuraTableYAML is the minimal shape of a Hasura table YAML file we need.
type hasuraTableYAML struct {
	Table struct {
		Schema string `yaml:"schema"`
		Name   string `yaml:"name"`
	} `yaml:"table"`
	SelectPermissions []struct {
		Role       string `yaml:"role"`
		Permission struct {
			Filter map[string]interface{} `yaml:"filter"`
		} `yaml:"permission"`
	} `yaml:"select_permissions"`
}

// loadHasuraFilters reads every YAML file under {projectRoot}/hasura/metadata/tables/
// and returns a map of table name → true when any select_permissions entry has a
// filter containing "source_account_id". Returns nil when metadata is not found.
func loadHasuraFilters(projectRoot string) map[string]bool {
	if projectRoot == "" {
		return nil
	}
	metaDir := filepath.Join(projectRoot, "hasura", "metadata", "tables")
	entries, err := os.ReadDir(metaDir)
	if err != nil {
		return nil // metadata directory not present
	}

	result := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(metaDir, entry.Name()))
		if err != nil {
			continue
		}
		var tbl hasuraTableYAML
		if err := yaml.Unmarshal(data, &tbl); err != nil {
			continue
		}
		tableName := tbl.Table.Name
		if tableName == "" {
			continue
		}
		for _, perm := range tbl.SelectPermissions {
			if _, ok := perm.Permission.Filter["source_account_id"]; ok {
				result[tableName] = true
				break
			}
		}
		// Mark explicitly as false if not already set (table found but no filter).
		if _, exists := result[tableName]; !exists {
			result[tableName] = false
		}
	}
	return result
}

// hasuraFilterStatus maps a loaded filter index into a HasuraFilterStatus value.
// When filterIndex is nil, the metadata was not found and the status is unknown.
func hasuraFilterStatus(filterIndex map[string]bool, table string) HasuraFilterStatus {
	if filterIndex == nil {
		return HasuraFilterUnknown
	}
	if v, ok := filterIndex[table]; ok {
		if v {
			return HasuraFilterPresent
		}
		return HasuraFilterMissing
	}
	// Table not mentioned in metadata = no filter defined.
	return HasuraFilterMissing
}

// findProjectRoot walks up from cwd looking for .nself/ or docker-compose.yml.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".nself")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found")
		}
		dir = parent
	}
}
