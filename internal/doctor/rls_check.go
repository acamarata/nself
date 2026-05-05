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

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// RLSCheckID is the canonical check ID referenced in docs, wiki, and Alertmanager.
	RLSCheckID = "PERM-RLS-01"
)

// rlsTableInfo describes a single np_* table's RLS state from pg_class.
type rlsTableInfo struct {
	TableName         string
	RLSEnabled        bool // pg_class.relrowsecurity
	RLSForced         bool // pg_class.relforcerowsecurity
	PolicyCount       int
	HasTenantIDColumn bool
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

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": RLS enforcement",
			Status:  "warn",
			Message: fmt.Sprintf("cannot open DB for RLS check: %v", err),
		}}
	}
	defer db.Close()

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
			) AS has_tenant_id
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
	defer rows.Close()

	var tables []rlsTableInfo
	for rows.Next() {
		var t rlsTableInfo
		if err := rows.Scan(&t.TableName, &t.RLSEnabled, &t.RLSForced, &t.PolicyCount, &t.HasTenantIDColumn); err != nil {
			return nil, fmt.Errorf("scan pg_class row: %w", err)
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

// checkHasuraRowFilters queries the Hasura metadata API to verify that every
// np_* table with a tenant_id column has a row filter on that column for the
// user role. Returns HASURA-FILTER-MISSING results for violations.
func checkHasuraRowFilters(ctx context.Context, tables []rlsTableInfo, warnStatus string) []CheckResult {
	hasuraURL := os.Getenv("HASURA_GRAPHQL_URL")
	if hasuraURL == "" {
		hasuraURL = "http://127.0.0.1:8080"
	}
	adminSecret := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")
	if adminSecret == "" {
		// Cannot query metadata without admin secret. Skip gracefully.
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": Hasura filter check",
			Status:  "warn",
			Message: "HASURA_GRAPHQL_ADMIN_SECRET not set; skipping Hasura row-filter audit",
		}}
	}

	metadataURL := strings.TrimRight(hasuraURL, "/") + "/v1/metadata"

	payload := `{"type":"export_metadata","args":{}}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadataURL, strings.NewReader(payload))
	if err != nil {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": Hasura filter check",
			Status:  "warn",
			Message: fmt.Sprintf("cannot build Hasura metadata request: %v", err),
		}}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", adminSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": Hasura filter check",
			Status:  "warn",
			Message: fmt.Sprintf("Hasura metadata API unreachable: %v (skipping filter audit)", err),
		}}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": Hasura filter check",
			Status:  "warn",
			Message: fmt.Sprintf("Hasura metadata API returned %d (skipping filter audit)", resp.StatusCode),
		}}
	}

	// Parse just enough of the metadata to find select permissions per table.
	var meta struct {
		Sources []struct {
			Tables []struct {
				Table struct {
					Name string `json:"name"`
				} `json:"table"`
				SelectPermissions []struct {
					Role       string `json:"role"`
					Permission struct {
						Filter json.RawMessage `json:"filter"`
					} `json:"permission"`
				} `json:"select_permissions"`
			} `json:"tables"`
		} `json:"sources"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return []CheckResult{{
			Section: "security",
			Name:    RLSCheckID + ": Hasura filter check",
			Status:  "warn",
			Message: fmt.Sprintf("cannot parse Hasura metadata: %v (skipping filter audit)", err),
		}}
	}

	// Build a lookup: table_name -> select permission filter for "user" role.
	type hasuraTablePerms struct {
		hasUserRole  bool
		filterHasTID bool // filter JSON references tenant_id
	}
	perms := make(map[string]hasuraTablePerms)
	for _, src := range meta.Sources {
		for _, tbl := range src.Tables {
			name := tbl.Table.Name
			if !strings.HasPrefix(name, "np_") {
				continue
			}
			p := hasuraTablePerms{}
			for _, sp := range tbl.SelectPermissions {
				if sp.Role == "user" {
					p.hasUserRole = true
					filterStr := string(sp.Permission.Filter)
					if strings.Contains(filterStr, "tenant_id") {
						p.filterHasTID = true
					}
				}
			}
			perms[name] = p
		}
	}

	var results []CheckResult
	for _, t := range tables {
		if !t.HasTenantIDColumn {
			continue
		}
		p, found := perms[t.TableName]
		if !found || !p.hasUserRole || !p.filterHasTID {
			msg := fmt.Sprintf("HASURA-FILTER-MISSING table=%s role=user: np_* table has tenant_id column but Hasura select permission for 'user' role is missing or has no tenant_id row filter", t.TableName)
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: HASURA-FILTER-MISSING table=%s", RLSCheckID, t.TableName),
				Status:  warnStatus,
				Message: msg,
			})
		}
	}

	return results
}
