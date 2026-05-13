package audit

// table_audit.go — S1.T08: nself plugin audit-tables
//
// Reads row counts per np_* table per source_account_id.
// Outputs a JSON baseline suitable for diffing before/after plugin renames.
// Detects:
//   - np_* tables missing source_account_id (isolation unknown)
//   - multi-account contamination (rows from >1 source_account_id)
//
// This command is read-only. It never modifies DB state.
// Security-Always-Free Doctrine: runs without a license.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq" // postgres driver (pure Go, CGO_ENABLED=0 compatible)
)

// TableAuditResult holds the audit result for a single table.
type TableAuditResult struct {
	// Table is the table name (e.g. "np_chat_conversations").
	Table string `json:"table"`
	// HasSourceAccountID reports whether source_account_id exists in the table.
	HasSourceAccountID bool `json:"has_source_account_id"`
	// TotalRows is the row count across all source_account_id values.
	TotalRows int64 `json:"total_rows"`
	// Accounts maps source_account_id → row count. Nil when HasSourceAccountID is false.
	Accounts map[string]int64 `json:"accounts,omitempty"`
	// Warning is non-empty when a problem is detected (missing column or multi-account).
	Warning string `json:"warning,omitempty"`
}

// AuditReport is the top-level JSON structure emitted by RunTableAudit.
type AuditReport struct {
	// AuditAt is the UTC timestamp of the audit run.
	AuditAt string `json:"audit_at"`
	// Tables contains one entry per np_* table found in the public schema.
	Tables []TableAuditResult `json:"tables"`
}

// RunTableAudit connects to Postgres using NSELF_DB_URL (fallback: DATABASE_URL)
// and returns an AuditReport for every np_* table in the public schema.
//
// ctx is forwarded to every DB call. The function adds a 30-second timeout
// around the total audit run to keep the doctor check bounded.
func RunTableAudit(ctx context.Context) (*AuditReport, error) {
	dbURL := os.Getenv("NSELF_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("NSELF_DB_URL not set; cannot run table audit")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("open DB: %w", err)
	}
	defer db.Close()

	auditCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := db.PingContext(auditCtx); err != nil {
		return nil, fmt.Errorf("ping DB: %w", err)
	}

	tables, err := listNPTables(auditCtx, db)
	if err != nil {
		return nil, err
	}

	report := &AuditReport{
		AuditAt: time.Now().UTC().Format(time.RFC3339),
		Tables:  make([]TableAuditResult, 0, len(tables)),
	}

	for _, tbl := range tables {
		result, err := auditTable(auditCtx, db, tbl)
		if err != nil {
			// Non-fatal: record the error as a warning and continue.
			report.Tables = append(report.Tables, TableAuditResult{
				Table:   tbl,
				Warning: fmt.Sprintf("audit failed: %v", err),
			})
			continue
		}
		report.Tables = append(report.Tables, result)
	}

	return report, nil
}

// listNPTables returns all table names matching np_* in the public schema,
// ordered alphabetically.
func listNPTables(ctx context.Context, db *sql.DB) ([]string, error) {
	const q = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_name LIKE 'np\_%'
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list np_* tables: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// hasColumn reports whether the given column exists on table in the public schema.
func hasColumn(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	const q = `
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name  = $1
		  AND column_name = $2
	`
	var n int
	if err := db.QueryRowContext(ctx, q, table, column).Scan(&n); err != nil {
		return false, fmt.Errorf("check column %s.%s: %w", table, column, err)
	}
	return n > 0, nil
}

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
	defer rows.Close()

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
