package gdpr

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ExportFormat is the output encoding for a GDPR export archive.
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
)

// ExportResult carries the raw archive bytes and metadata.
type ExportResult struct {
	RequestID   string
	UserID      string
	Format      ExportFormat
	Data        []byte
	GeneratedAt time.Time
}

// ExportUserData queries every plugin-registered table for rows matching
// userID, serialises them into a ZIP archive (one file per plugin/table),
// and returns the result. dryRun=true returns an empty archive and only
// enumerates what would be exported.
func ExportUserData(ctx context.Context, db *sql.DB, requestID, userID string, format ExportFormat, dryRun bool) (*ExportResult, error) {
	strategies, err := RegistryTableStrategies(ctx, db, SubjectTypeUser)
	if err != nil {
		return nil, fmt.Errorf("gdpr export: %w", err)
	}

	// Also include core np_* tables that always hold user data.
	strategies["core"] = coreUserTables()

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	for pluginName, tables := range strategies {
		for _, tbl := range tables {
			rows, err := queryTableForUser(ctx, db, tbl, userID)
			if err != nil {
				// Non-fatal: log and continue so one bad table doesn't block the export.
				rows = []map[string]interface{}{{"_error": err.Error()}}
			}

			if dryRun {
				continue
			}

			filename := fmt.Sprintf("%s/%s.json", pluginName, tbl.Table)
			if format == FormatCSV {
				filename = fmt.Sprintf("%s/%s.csv", pluginName, tbl.Table)
			}
			fw, err := zw.Create(filename)
			if err != nil {
				return nil, fmt.Errorf("gdpr export: create zip entry %s: %w", filename, err)
			}

			if format == FormatCSV {
				if err := writeCSV(fw, rows); err != nil {
					return nil, fmt.Errorf("gdpr export: write csv %s: %w", filename, err)
				}
			} else {
				enc := json.NewEncoder(fw)
				enc.SetIndent("", "  ")
				if err := enc.Encode(rows); err != nil {
					return nil, fmt.Errorf("gdpr export: encode json %s: %w", filename, err)
				}
			}
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("gdpr export: close zip: %w", err)
	}

	return &ExportResult{
		RequestID:   requestID,
		UserID:      userID,
		Format:      format,
		Data:        buf.Bytes(),
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// coreUserTables returns the built-in nSelf tables that always contain user
// data and should be included in every export regardless of plugin registry.
func coreUserTables() []TableStrategy {
	return []TableStrategy{
		{Table: "np_users", UserCol: "id", Strategy: "export"},
		{Table: "np_user_sessions", UserCol: "user_id", Strategy: "export"},
		{Table: "np_user_metadata", UserCol: "user_id", Strategy: "export"},
	}
}

// queryTableForUser runs a SELECT * FROM <table> WHERE <user_col> = $1 and
// returns results as a slice of string-keyed maps.
func queryTableForUser(ctx context.Context, db *sql.DB, tbl TableStrategy, userID string) ([]map[string]interface{}, error) {
	q := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", tbl.Table, tbl.UserCol)
	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", tbl.Table, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("columns %s: %w", tbl.Table, err)
	}

	var out []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan %s: %w", tbl.Table, err)
		}
		row := map[string]interface{}{}
		for i, col := range cols {
			row[col] = vals[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
