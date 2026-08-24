package supabase

// supabase_pull.go — pulling schema, RLS, auth users and storage from Supabase.
//
// Purpose: use the Client in supabase.go to pull the Postgres schema, RLS policies, auth users and storage buckets from a live Supabase project, split out for file size.
// Inputs: a *Client configured with a Supabase project URL and service_role key.
// Outputs: PullResult populated with Table/RLSPolicy/AuthUser/StorageBucket values.
// Constraints: pure move from supabase.go (CLI-R12 Batch E); no behaviour change.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PullSchema fetches tables and columns from the Supabase project.
// It uses the PostgREST OpenAPI endpoint which returns schema information.
func (c *Client) PullSchema(ctx context.Context) ([]Table, error) {
	url := fmt.Sprintf("https://%s.supabase.co/rest/v1/", c.cfg.ProjectRef)
	data, err := c.get(ctx, url, map[string]string{
		"Accept": "application/openapi+json",
	})
	if err != nil {
		return nil, fmt.Errorf("pulling schema: %w", err)
	}

	var spec struct {
		Definitions map[string]struct {
			Properties map[string]struct {
				Type        string `json:"type"`
				Format      string `json:"format"`
				Default     string `json:"default"`
				Description string `json:"description"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"definitions"`
	}

	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing schema response: %w", err)
	}

	var tables []Table
	for name, def := range spec.Definitions {
		t := Table{
			Schema: "public",
			Name:   name,
		}
		requiredSet := make(map[string]bool, len(def.Required))
		for _, r := range def.Required {
			requiredSet[r] = true
		}
		for colName, prop := range def.Properties {
			t.Columns = append(t.Columns, Column{
				Name:       colName,
				DataType:   coalesceType(prop.Type, prop.Format),
				IsNullable: !requiredSet[colName],
				Default:    prop.Default,
			})
		}
		tables = append(tables, t)
	}

	return tables, nil
}

// coalesceType maps OpenAPI type+format to a SQL-ish type string.
func coalesceType(t, format string) string {
	if format != "" {
		switch format {
		case "uuid":
			return "uuid"
		case "timestamp with time zone", "timestamptz":
			return "timestamptz"
		case "int8", "bigint":
			return "bigint"
		case "int4", "integer":
			return "integer"
		case "float8":
			return "float8"
		case "json", "jsonb":
			return "jsonb"
		case "text":
			return "text"
		case "bool":
			return "boolean"
		}
	}
	switch t {
	case "integer":
		return "integer"
	case "number":
		return "numeric"
	case "boolean":
		return "boolean"
	case "array":
		return "jsonb"
	default:
		return "text"
	}
}

// ---------------------------------------------------------------------------
// RLS pull — queries pg_policies via Supabase Management API
// ---------------------------------------------------------------------------

// pgPoliciesSQL is the query executed against pg_policies.
// The Supabase Management API executes it with service_role privileges,
// giving access to pg_catalog views.
const pgPoliciesSQL = `
SELECT
    schemaname        AS table_schema,
    tablename         AS table_name,
    policyname        AS policy_name,
    cmd               AS command,
    permissive        AS permissive,
    COALESCE(roles::text, '{}') AS roles,
    COALESCE(qual, '')          AS using_expr,
    COALESCE(with_check, '')    AS check_expr
FROM pg_policies
WHERE schemaname NOT IN ('pg_catalog', 'information_schema', 'auth', 'storage', 'realtime', 'supabase_functions')
ORDER BY schemaname, tablename, policyname;
`

// rlsPolicyRow is the JSON shape returned by the Management API SQL endpoint.
type rlsPolicyRow struct {
	TableSchema string `json:"table_schema"`
	TableName   string `json:"table_name"`
	PolicyName  string `json:"policy_name"`
	Command     string `json:"command"`
	Permissive  string `json:"permissive"` // "PERMISSIVE" or "RESTRICTIVE"
	Roles       string `json:"roles"`      // array literal: {user,anon} or {}
	UsingExpr   string `json:"using_expr"`
	CheckExpr   string `json:"check_expr"`
}

// PullRLSPolicies fetches RLS policies from the Supabase project by executing
// a query against pg_policies via the Supabase Management API SQL endpoint.
//
// The Management API endpoint is:
//
//	POST https://api.supabase.com/v1/projects/{ref}/database/query
//	Authorization: Bearer <service_role_key>
//	Content-Type: application/json
//	{"query": "<SQL>"}
//
// If the Management API is unreachable or returns an auth error, an empty
// slice is returned along with an informational error so the caller can
// include a manual-review note in the generated migration SQL.
func (c *Client) PullRLSPolicies(ctx context.Context) ([]RLSPolicy, error) {
	url := fmt.Sprintf("https://api.supabase.com/v1/projects/%s/database/query", c.cfg.ProjectRef)

	payload := fmt.Sprintf(`{"query":%q}`, pgPoliciesSQL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("building RLS query request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.ServiceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network error — return empty + advisory so callers can still generate
		// an artifact with a manual-review note.
		return nil, fmt.Errorf("RLS policy fetch network error (check connectivity): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading RLS policy response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("RLS policy fetch unauthorized (HTTP %d): verify the service_role key has Management API access", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RLS policy fetch failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rows []rlsPolicyRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("parsing RLS policy response: %w", err)
	}

	policies := make([]RLSPolicy, 0, len(rows))
	for _, row := range rows {
		policies = append(policies, RLSPolicy{
			TableSchema: row.TableSchema,
			TableName:   row.TableName,
			PolicyName:  row.PolicyName,
			Command:     row.Command,
			Permissive:  row.Permissive != "RESTRICTIVE",
			Roles:       parsePostgresArray(row.Roles),
			Using:       row.UsingExpr,
			WithCheck:   row.CheckExpr,
		})
	}
	return policies, nil
}
