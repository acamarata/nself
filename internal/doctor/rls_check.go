package doctor

// rls_check.go — PERM-RLS-01: RLS enforcement check for np_* tables.
//
// S74-T02 + S74-T-PERM-01 (Yellow Dim amendment):
//   For each np_* table: RLS enabled + at least one policy exists.
//   For multiApp.supported tables: FORCE RLS active.
//   For tenant_id tables: Hasura metadata has tenant_id row filter.
//
// Violations: WARN by default; ERROR with --strict flag.
// Security-Always-Free Doctrine: this check runs without a license.
//
// The Hasura-metadata row-filter half of check (d) lives in
// rls_hasura_filters.go — split out (CLI-R12) as a pure move from this file.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"
)

const (
	// RLSCheckID is the canonical check ID referenced in docs, wiki, and Alertmanager.
	RLSCheckID = "PERM-RLS-01"
)

// rlsTableInfo describes a single np_* table's RLS state from pg_class.
type rlsTableInfo struct {
	TableName                string
	RLSEnabled               bool // pg_class.relrowsecurity
	RLSForced                bool // pg_class.relforcerowsecurity
	PolicyCount              int
	HasTenantIDColumn        bool
	HasSourceAccountIDColumn bool
}

// CheckRLSEnforcement implements PERM-RLS-01.
//
// It queries Postgres pg_class for every np_* table and verifies:
//
//	(a) RLS is enabled (relrowsecurity = true)
//	(b) at least one policy exists
//	(c) PatternTenantScoped tables have FORCE RLS (relforcerowsecurity = true)
//	(d) tables with tenant_id column have Hasura row filter for that column
//
// Violations are WARN by default. Pass strict=true to escalate to fail.
//
// This check is read-only. It never modifies DB state.
func CheckRLSEnforcement(ctx context.Context, strict bool) []CheckResult {
	var results []CheckResult

	dbURL := os.Getenv("NSELF_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": RLS enforcement",
			Status:  "warn",
			Message: "NSELF_DB_URL not set; skipping RLS check",
		}}
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": RLS enforcement",
			Status:  "warn",
			Message: fmt.Sprintf("cannot open DB for RLS check: %v", err),
		}}
	}
	defer func() { _ = db.Close() }()

	// Set a short timeout — this is a diagnostic check.
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	tables, err := queryNPTables(checkCtx, db)
	if err != nil {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": RLS enforcement",
			Status:  "warn",
			Message: fmt.Sprintf("cannot query pg_class for np_* tables: %v", err),
		}}
	}

	if len(tables) == 0 {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": RLS enforcement",
			Status:  "pass",
			Message: "no np_* tables found (fresh install or no plugins loaded)",
		}}
	}

	warnStatus := "warn"
	if strict {
		warnStatus = "fail"
	}

	for _, t := range tables {
		// Check (a): RLS enabled.
		if !t.RLSEnabled {
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: RLS-DISABLED table=%s", RLSCheckID, t.TableName),
				Status:  warnStatus,
				Message: fmt.Sprintf("RLS not enabled on %s (relrowsecurity=false). Run: ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", t.TableName, t.TableName),
				FixCmd:  fmt.Sprintf("nself migrate apply --rls-enable %s", t.TableName),
			})
			continue
		}

		// Check (b): at least one policy exists.
		if t.PolicyCount == 0 {
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: NO-POLICY table=%s", RLSCheckID, t.TableName),
				Status:  warnStatus,
				Message: fmt.Sprintf("RLS enabled on %s but no policies exist — all queries will be blocked by default deny", t.TableName),
			})
			continue
		}

		// Check (c): PatternTenantScoped tables must have FORCE RLS.
		// We detect PatternTenantScoped by the presence of tenant_id column.
		if t.HasTenantIDColumn && !t.RLSForced {
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: RLS-FORCE-MISSING table=%s", RLSCheckID, t.TableName),
				Status:  warnStatus,
				Message: fmt.Sprintf("RLS-FORCE-MISSING table=%s: FORCE ROW LEVEL SECURITY not set. Table owner queries bypass policies without FORCE. Run: ALTER TABLE %s FORCE ROW LEVEL SECURITY;", t.TableName, t.TableName),
				FixCmd:  fmt.Sprintf("nself migrate apply --rls-force %s", t.TableName),
			})
		}
	}

	// Check (d): Hasura row filters for tenant_id tables.
	hasuraResults := checkHasuraRowFilters(ctx, tables, warnStatus)
	results = append(results, hasuraResults...)

	// If no violations found, emit a single pass result.
	if len(results) == 0 {
		results = append(results, CheckResult{
			Section: "security",
			Name:    RLSCheckID + ": RLS enforcement",
			Status:  "pass",
			Message: fmt.Sprintf("all %d np_* tables have RLS enabled, policies present, and tenant_id Hasura filters where applicable", len(tables)),
		})
	}

	return results
}

// queryNPTables fetches RLS state for all np_* tables from pg_class.
func queryNPTables(ctx context.Context, db *sql.DB) ([]rlsTableInfo, error) {
	query := `
		SELECT
			c.relname AS table_name,
			c.relrowsecurity AS rls_enabled,
			c.relforcerowsecurity AS rls_forced,
			COUNT(p.polname) AS policy_count,
			EXISTS(
				SELECT 1 FROM information_schema.columns ic
				WHERE ic.table_name = c.relname
				  AND ic.column_name = 'tenant_id'
				  AND ic.table_schema = 'public'
			) AS has_tenant_id,
			EXISTS(
				SELECT 1 FROM information_schema.columns ic2
				WHERE ic2.table_name = c.relname
				  AND ic2.column_name = 'source_account_id'
				  AND ic2.table_schema = 'public'
			) AS has_source_account_id
		FROM pg_class c
		LEFT JOIN pg_policy p ON p.polrelid = c.oid
		WHERE c.relkind = 'r'
		  AND c.relname LIKE 'np\_%'
		  AND c.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
		GROUP BY c.relname, c.relrowsecurity, c.relforcerowsecurity
		ORDER BY c.relname
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("pg_class query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []rlsTableInfo
	for rows.Next() {
		var t rlsTableInfo
		if err := rows.Scan(&t.TableName, &t.RLSEnabled, &t.RLSForced, &t.PolicyCount, &t.HasTenantIDColumn, &t.HasSourceAccountIDColumn); err != nil {
			return nil, fmt.Errorf("scan pg_class row: %w", err)
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}
