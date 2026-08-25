package doctor

// rls_hasura_filters.go — check (d) of PERM-RLS-01: verifies Hasura select
// permissions carry a row filter referencing tenant_id/source_account_id for
// every np_* table that has the corresponding isolation column. Split out of
// rls_check.go (CLI-R12) as a pure move.
//
// Inputs: a context, the []rlsTableInfo from queryNPTables (rls_check.go),
// and the warn/fail status to use for violations.
// Outputs: []CheckResult — HASURA-FILTER-MISSING violations, or a single
// warn result if the Hasura metadata API could not be queried.
// Constraints: read-only; never modifies Hasura metadata.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

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
		hasUserRole   bool
		filterHasTID  bool // filter JSON references tenant_id
		filterHasSAID bool // filter JSON references source_account_id
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
					if strings.Contains(filterStr, "source_account_id") {
						p.filterHasSAID = true
					}
				}
			}
			perms[name] = p
		}
	}

	var results []CheckResult
	for _, t := range tables {
		needsTID := t.HasTenantIDColumn
		needsSAID := t.HasSourceAccountIDColumn
		// Tables with no isolation column are exempt from the Hasura filter check.
		if !needsTID && !needsSAID {
			continue
		}
		p, found := perms[t.TableName]
		if !found || !p.hasUserRole {
			msg := fmt.Sprintf("HASURA-FILTER-MISSING table=%s role=user: np_* table has isolation column but Hasura select permission for 'user' role is missing or has no row filter", t.TableName)
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: HASURA-FILTER-MISSING table=%s", RLSCheckID, t.TableName),
				Status:  warnStatus,
				Message: msg,
			})
			continue
		}
		if needsTID && !p.filterHasTID {
			msg := fmt.Sprintf("HASURA-FILTER-MISSING table=%s role=user: has tenant_id column but Hasura 'user' row filter does not reference tenant_id", t.TableName)
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("%s: HASURA-FILTER-MISSING table=%s", RLSCheckID, t.TableName),
				Status:  warnStatus,
				Message: msg,
			})
		}
		if needsSAID && !p.filterHasSAID {
			msg := fmt.Sprintf("HASURA-FILTER-MISSING table=%s role=user: has source_account_id column but Hasura 'user' row filter does not reference source_account_id", t.TableName)
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
