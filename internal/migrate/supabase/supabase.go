// Package supabase implements the Supabase → nSelf migration pipeline.
//
// It connects to a Supabase project via PostgREST + the service_role key,
// pulls schema, data, auth users, and storage buckets, then generates
// nSelf-compatible Drizzle migration SQL files, Hasura metadata YAML, an
// auth user import script, and a human-readable next-steps summary.
package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// Config holds the credentials for one Supabase project.
type Config struct {
	ProjectRef string // e.g. "abcdefghijklmnopqrst"
	ServiceKey string // service_role JWT
}

// Table is a simplified representation of a Supabase table.
type Table struct {
	Schema  string
	Name    string
	Columns []Column
}

// Column describes a single column returned by the Supabase REST schema API.
type Column struct {
	Name       string
	DataType   string
	IsNullable bool
	Default    string
}

// RLSPolicy is a PostgreSQL row-level security policy captured from
// the information_schema via PostgREST.
type RLSPolicy struct {
	TableSchema string
	TableName   string
	PolicyName  string
	Command     string // SELECT | INSERT | UPDATE | DELETE | ALL
	Permissive  bool
	Roles       []string
	Using       string
	WithCheck   string
}

// AuthUser represents a minimal view of a Supabase auth.users row.
type AuthUser struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	EmailConfirmed bool   `json:"email_confirmed_at"`
	CreatedAt      string `json:"created_at"`
}

// StorageBucket is a Supabase Storage bucket.
type StorageBucket struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Public        bool   `json:"public"`
	FileSizeLimit int    `json:"file_size_limit"`
}

// PullResult contains everything pulled from a Supabase project.
type PullResult struct {
	Tables   []Table
	Policies []RLSPolicy
	Users    []AuthUser
	Buckets  []StorageBucket
}

// GeneratedArtifacts contains the files the generator produced in memory
// (keyed by relative output path).
type GeneratedArtifacts struct {
	// MigrationSQL is the Drizzle-compatible SQL migration file content.
	MigrationSQL string
	// HasuraMetadata is the Hasura metadata YAML for the pulled schema.
	HasuraMetadata string
	// AuthImportScript is a shell script that imports users into nSelf.
	AuthImportScript string
	// Summary is a human-readable text summary with next steps.
	Summary string
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client talks to the Supabase REST APIs.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a Client for the given Supabase project.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// baseURL returns the PostgREST endpoint for the project.
func (c *Client) baseURL() string {
	return fmt.Sprintf("https://%s.supabase.co/rest/v1", c.cfg.ProjectRef)
}

// adminURL returns the Supabase Admin API base URL.
func (c *Client) adminURL() string {
	return fmt.Sprintf("https://%s.supabase.co", c.cfg.ProjectRef)
}

// get performs a GET with exponential backoff (max 30 s total, max 8 s interval).
func (c *Client) get(ctx context.Context, url string, extraHeaders map[string]string) ([]byte, error) {
	op := func() ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, backoff.Permanent(fmt.Errorf("building request: %w", err))
		}
		req.Header.Set("apikey", c.cfg.ServiceKey)
		req.Header.Set("Authorization", "Bearer "+c.cfg.ServiceKey)
		req.Header.Set("Accept", "application/json")
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http request: %w", err)
		}
		defer resp.Body.Close()

		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("reading response body: %w", readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, backoff.Permanent(fmt.Errorf("authentication error (HTTP %d) -- check your service_role key", resp.StatusCode))
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, backoff.Permanent(fmt.Errorf("resource not found (HTTP 404) -- check your project ref"))
		}
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error (HTTP %d) -- will retry", resp.StatusCode)
		}
		if resp.StatusCode >= 400 {
			return nil, backoff.Permanent(fmt.Errorf("client error (HTTP %d): %s", resp.StatusCode, string(data)))
		}

		return data, nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 8 * time.Second

	body, err := backoff.Retry(ctx, op,
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// ---------------------------------------------------------------------------
// Schema pull
// ---------------------------------------------------------------------------

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

// parsePostgresArray converts a Postgres array literal like "{user,anon}" into
// a Go string slice.
func parsePostgresArray(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return nil
	}
	// Strip surrounding braces.
	s = strings.TrimPrefix(strings.TrimSuffix(s, "}"), "{")
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Auth users pull
// ---------------------------------------------------------------------------

// PullAuthUsers fetches up to 1 000 users from the Supabase Admin auth API.
func (c *Client) PullAuthUsers(ctx context.Context) ([]AuthUser, error) {
	url := fmt.Sprintf("%s/auth/v1/admin/users?per_page=1000", c.adminURL())
	data, err := c.get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("pulling auth users: %w", err)
	}

	var wrapper struct {
		Users []AuthUser `json:"users"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Try bare array fallback.
		var users []AuthUser
		if err2 := json.Unmarshal(data, &users); err2 != nil {
			return nil, fmt.Errorf("parsing auth users response: %w", err)
		}
		return users, nil
	}
	return wrapper.Users, nil
}

// ---------------------------------------------------------------------------
// Storage buckets pull
// ---------------------------------------------------------------------------

// PullStorageBuckets fetches all storage buckets from the Supabase Storage API.
func (c *Client) PullStorageBuckets(ctx context.Context) ([]StorageBucket, error) {
	url := fmt.Sprintf("%s/storage/v1/bucket", c.adminURL())
	data, err := c.get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("pulling storage buckets: %w", err)
	}

	var buckets []StorageBucket
	if err := json.Unmarshal(data, &buckets); err != nil {
		return nil, fmt.Errorf("parsing storage buckets response: %w", err)
	}
	return buckets, nil
}

// ---------------------------------------------------------------------------
// Pull (orchestrates all pulls)
// ---------------------------------------------------------------------------

// Pull runs all four pull operations and returns the aggregate result.
func Pull(ctx context.Context, cfg Config) (*PullResult, error) {
	c := NewClient(cfg)

	tables, err := c.PullSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	policies, err := c.PullRLSPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("rls: %w", err)
	}

	users, err := c.PullAuthUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth users: %w", err)
	}

	buckets, err := c.PullStorageBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage buckets: %w", err)
	}

	return &PullResult{
		Tables:   tables,
		Policies: policies,
		Users:    users,
		Buckets:  buckets,
	}, nil
}

// ---------------------------------------------------------------------------
// Generator
// ---------------------------------------------------------------------------

// Generate converts a PullResult into nSelf-compatible artifacts.
func Generate(projectRef string, r *PullResult) *GeneratedArtifacts {
	return &GeneratedArtifacts{
		MigrationSQL:     generateMigrationSQL(r),
		HasuraMetadata:   generateHasuraMetadata(projectRef, r),
		AuthImportScript: generateAuthImportScript(r.Users),
		Summary:          generateSummary(projectRef, r),
	}
}

// generateMigrationSQL produces a Drizzle-compatible CREATE TABLE migration.
func generateMigrationSQL(r *PullResult) string {
	var b bytes.Buffer
	ts := time.Now().Format("20060102150405")
	fmt.Fprintf(&b, "-- nSelf migration generated from Supabase export\n")
	fmt.Fprintf(&b, "-- Generated: %s\n", ts)
	fmt.Fprintf(&b, "-- Tables: %d  |  RLS policies: %d\n\n", len(r.Tables), len(r.Policies))

	for _, t := range r.Tables {
		fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s.%s (\n", quoteIdent(t.Schema), quoteIdent(t.Name))
		for i, col := range t.Columns {
			nullable := ""
			if !col.IsNullable {
				nullable = " NOT NULL"
			}
			def := ""
			if col.Default != "" {
				def = fmt.Sprintf(" DEFAULT %s", col.Default)
			}
			comma := ","
			if i == len(t.Columns)-1 {
				comma = ""
			}
			fmt.Fprintf(&b, "  %s %s%s%s%s\n", quoteIdent(col.Name), col.DataType, nullable, def, comma)
		}
		fmt.Fprintf(&b, ");\n\n")
	}

	if len(r.Policies) == 0 {
		fmt.Fprintf(&b, "-- NOTE: RLS policies could not be fetched automatically via PostgREST.\n")
		fmt.Fprintf(&b, "-- Review your Supabase RLS policies and re-apply them using:\n")
		fmt.Fprintf(&b, "--   ALTER TABLE <table> ENABLE ROW LEVEL SECURITY;\n")
		fmt.Fprintf(&b, "--   CREATE POLICY ... ON <table> ...\n\n")
	}

	for _, p := range r.Policies {
		roles := strings.Join(p.Roles, ", ")
		permissive := "PERMISSIVE"
		if !p.Permissive {
			permissive = "RESTRICTIVE"
		}
		fmt.Fprintf(&b, "-- RLS: %s.%s  cmd=%s  %s  roles=[%s]\n", p.TableSchema, p.TableName, p.Command, permissive, roles)
		fmt.Fprintf(&b, "ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;\n", quoteIdent(p.TableSchema), quoteIdent(p.TableName))
		if p.Using != "" {
			fmt.Fprintf(&b, "CREATE POLICY %s ON %s.%s AS %s FOR %s USING (%s);\n",
				quoteIdent(p.PolicyName), quoteIdent(p.TableSchema), quoteIdent(p.TableName),
				permissive, p.Command, p.Using)
		}
		fmt.Fprintf(&b, "\n")
	}

	return b.String()
}

// generateHasuraMetadata produces a minimal Hasura metadata YAML for the tables.
func generateHasuraMetadata(projectRef string, r *PullResult) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# Hasura metadata generated from Supabase project %s\n", projectRef)
	fmt.Fprintf(&b, "# Apply with: hasura metadata apply\n\n")
	fmt.Fprintf(&b, "version: 3\n")
	fmt.Fprintf(&b, "sources:\n")
	fmt.Fprintf(&b, "  - name: default\n")
	fmt.Fprintf(&b, "    kind: postgres\n")
	fmt.Fprintf(&b, "    tables:\n")
	for _, t := range r.Tables {
		fmt.Fprintf(&b, "      - table:\n")
		fmt.Fprintf(&b, "          schema: %s\n", t.Schema)
		fmt.Fprintf(&b, "          name: %s\n", t.Name)
		fmt.Fprintf(&b, "        select_permissions:\n")
		fmt.Fprintf(&b, "          - role: user\n")
		fmt.Fprintf(&b, "            permission:\n")
		fmt.Fprintf(&b, "              filter: {}\n")
		fmt.Fprintf(&b, "              columns: \"*\"\n")
	}
	return b.String()
}

// generateAuthImportScript produces a shell script to import Supabase auth users.
func generateAuthImportScript(users []AuthUser) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "#!/usr/bin/env bash\n")
	fmt.Fprintf(&b, "# nSelf auth user import — generated from Supabase export\n")
	fmt.Fprintf(&b, "# Users: %d\n", len(users))
	fmt.Fprintf(&b, "# Run: bash import-auth-users.sh\n\n")
	fmt.Fprintf(&b, "set -euo pipefail\n\n")

	if len(users) == 0 {
		fmt.Fprintf(&b, "echo 'No auth users to import.'\n")
		return b.String()
	}

	fmt.Fprintf(&b, "NSELF_AUTH_URL=\"${NSELF_AUTH_URL:-http://localhost:4000}\"\n\n")
	fmt.Fprintf(&b, "echo \"Importing %d users via nSelf auth API...\"\n\n", len(users))

	for _, u := range users {
		email := strings.ReplaceAll(u.Email, `"`, `\"`)
		fmt.Fprintf(&b, "# User: %s\n", email)
		fmt.Fprintf(&b, "curl -sf -X POST \"${NSELF_AUTH_URL}/admin/v1/users\" \\\n")
		fmt.Fprintf(&b, "  -H 'Content-Type: application/json' \\\n")
		fmt.Fprintf(&b, "  -d '{\"email\":\"%s\",\"id\":\"%s\",\"email_confirm\":true}' || true\n\n", email, u.ID)
	}

	fmt.Fprintf(&b, "echo 'Import complete. Verify with: nself auth users'\n")
	return b.String()
}

// generateSummary returns a human-readable summary string.
func generateSummary(projectRef string, r *PullResult) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Migration from Supabase project %q complete.\n\n", projectRef)
	fmt.Fprintf(&b, "Pulled:\n")
	fmt.Fprintf(&b, "  Tables:          %d\n", len(r.Tables))
	fmt.Fprintf(&b, "  RLS policies:    %d (see migration SQL for manual steps)\n", len(r.Policies))
	fmt.Fprintf(&b, "  Auth users:      %d\n", len(r.Users))
	fmt.Fprintf(&b, "  Storage buckets: %d\n\n", len(r.Buckets))
	fmt.Fprintf(&b, "Generated files:\n")
	fmt.Fprintf(&b, "  supabase-migration.sql       — run via: nself db migrate\n")
	fmt.Fprintf(&b, "  hasura-metadata.yaml         — apply via: hasura metadata apply\n")
	fmt.Fprintf(&b, "  import-auth-users.sh         — run to import users\n\n")
	fmt.Fprintf(&b, "Next steps:\n")
	fmt.Fprintf(&b, "  1. nself init                   — create a new nSelf project (if not done)\n")
	fmt.Fprintf(&b, "  2. nself start                  — boot the stack\n")
	fmt.Fprintf(&b, "  3. psql -f supabase-migration.sql  — apply schema\n")
	fmt.Fprintf(&b, "  4. hasura metadata apply           — load Hasura metadata\n")
	fmt.Fprintf(&b, "  5. bash import-auth-users.sh       — import auth users\n")
	if len(r.Buckets) > 0 {
		fmt.Fprintf(&b, "  6. Configure MinIO buckets to match Supabase Storage buckets:\n")
		for _, bucket := range r.Buckets {
			visibility := "private"
			if bucket.Public {
				visibility = "public"
			}
			fmt.Fprintf(&b, "       nself storage create-bucket %s --visibility=%s\n", bucket.Name, visibility)
		}
	}
	return b.String()
}

// quoteIdent wraps a SQL identifier in double-quotes.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
