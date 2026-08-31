package commands

// mcp_tools_data.go — Hasura/Postgres data-layer MCP tool handlers.
//
// Purpose: the three tools that predate CLI-R15 and talk to the running
//   Hasura instance directly over HTTP rather than through a `nself`
//   subcommand: schema introspection, permissions snapshot, and gated raw
//   migration SQL. Kept distinct from mcp_tools_core.go because they reach
//   into the project's live database, not the CLI's own command layer.
// Inputs:  MCP tool call arguments; NSELF_HASURA_GRAPHQL_URL/
//   HASURA_GRAPHQL_URL for the endpoint, HASURA_GRAPHQL_ADMIN_SECRET/
//   NSELF_HASURA_ADMIN_SECRET for auth.
// Outputs: mcp.CallToolResult with structured JSON content.
// Constraints: nself_run_migration enforces internal/sqlallowlist before any
//   execution path — see sqlallowlist.ValidateMigrationSQL — blocking DROP/
//   TRUNCATE/DELETE/ALTER ROLE/GRANT/REVOKE/psql meta-commands even when the
//   caller sets confirm=true programmatically.
// SPORT: CLI-CMD-MCP-001

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/nself-org/cli/internal/sqlallowlist"
)

// resolveHasuraEndpoint returns the Hasura base URL from env, defaulting to localhost:8080.
func resolveHasuraEndpoint() string {
	if u := os.Getenv("NSELF_HASURA_GRAPHQL_URL"); u != "" {
		return u
	}
	if u := os.Getenv("HASURA_GRAPHQL_URL"); u != "" {
		return u
	}
	return "http://127.0.0.1:8080"
}

// mcpPostJSON sends a JSON POST request with the Hasura admin secret header.
func mcpPostJSON(ctx context.Context, url, adminSecret string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if adminSecret != "" {
		req.Header.Set("X-Hasura-Admin-Secret", adminSecret)
	}

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func mcpGetSchemaHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hasuraURL := resolveHasuraEndpoint()
		adminSecret := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")
		if adminSecret == "" {
			adminSecret = os.Getenv("NSELF_HASURA_ADMIN_SECRET")
		}

		introspectURL := strings.TrimRight(hasuraURL, "/") + "/v1/graphql"
		body := map[string]string{"query": introspectionQuery}
		respBytes, err := mcpPostJSON(ctx, introspectURL, adminSecret, body)
		if err != nil {
			return mcpErrorResult("Schema introspection error: %v", err)
		}

		var compacted bytes.Buffer
		if err := json.Compact(&compacted, respBytes); err != nil {
			return mcp.NewToolResultText(string(respBytes)), nil
		}
		return mcp.NewToolResultText(compacted.String()), nil
	}
}

func mcpGetPermissionsHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		hasuraURL := resolveHasuraEndpoint()
		adminSecret := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")
		if adminSecret == "" {
			adminSecret = os.Getenv("NSELF_HASURA_ADMIN_SECRET")
		}

		metadataURL := strings.TrimRight(hasuraURL, "/") + "/v1/metadata"
		payload := map[string]interface{}{"type": "export_metadata", "args": map[string]interface{}{}}
		respBytes, err := mcpPostJSON(ctx, metadataURL, adminSecret, payload)
		if err != nil {
			return mcpErrorResult("Permissions snapshot error: %v", err)
		}

		var meta map[string]interface{}
		if err := json.Unmarshal(respBytes, &meta); err != nil {
			return mcp.NewToolResultText(string(respBytes)), nil
		}

		sources, _ := meta["sources"].([]interface{})
		type tablePerms struct {
			Table       string      `json:"table"`
			Permissions interface{} `json:"permissions"`
		}
		var result []tablePerms
		for _, src := range sources {
			srcMap, ok := src.(map[string]interface{})
			if !ok {
				continue
			}
			tables, _ := srcMap["tables"].([]interface{})
			for _, t := range tables {
				tMap, ok := t.(map[string]interface{})
				if !ok {
					continue
				}
				tableInfo, _ := tMap["table"].(map[string]interface{})
				tableName, _ := tableInfo["name"].(string)
				perms := map[string]interface{}{}
				for _, key := range []string{"select_permissions", "insert_permissions", "update_permissions", "delete_permissions"} {
					if v, ok := tMap[key]; ok {
						perms[key] = v
					}
				}
				result = append(result, tablePerms{Table: tableName, Permissions: perms})
			}
		}
		return mcp.NewToolResultStructuredOnly(result), nil
	}
}

func mcpRunMigrationHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sql, _ := args["sql"].(string)
		confirm, _ := args["confirm"].(bool)

		if sql == "" {
			return mcpErrorResult("Error: sql is required")
		}
		if !confirm {
			return mcpErrorResult("Error: confirm must be true to execute the migration. This is a safety gate.")
		}

		// DDL allowlist: reject destructive or privilege-altering SQL before
		// any execution path. This blocks AI Studio sessions from running
		// DROP TABLE, TRUNCATE, DELETE FROM, ALTER ROLE, GRANT/REVOKE, or
		// psql meta-commands even when confirm=true is set programmatically.
		if err := sqlallowlist.ValidateMigrationSQL(sql); err != nil {
			return mcpErrorResult("Error: %v", err)
		}

		raw, err := os.Getwd()
		if err != nil {
			return mcpErrorResult("Error: %v", err)
		}
		cwd, err := config.FindNSelfRoot(raw)
		if err != nil {
			return mcpErrorResult("no nself project found in %s — run 'nself init' first", raw)
		}
		out, err := mcpExecSelf(ctx, cwd, "db", "migrate", "--sql", sql)
		if err != nil {
			out, err = mcpApplyMigrationDirect(ctx, sql)
			if err != nil {
				return mcpErrorResult("Migration failed: %v", err)
			}
		}
		return mcp.NewToolResultText("Migration applied successfully.\n" + out), nil
	}
}

// mcpApplyMigrationDirect applies SQL directly via the Postgres connection string.
// It enforces the DDL allowlist as a defence-in-depth layer even on this fallback
// path — the primary check in mcpRunMigrationHandler runs first, but a second
// guard here ensures no direct caller can bypass it.
func mcpApplyMigrationDirect(ctx context.Context, sql string) (string, error) {
	if err := sqlallowlist.ValidateMigrationSQL(sql); err != nil {
		return "", err
	}

	pgURL := os.Getenv("POSTGRES_URL")
	if pgURL == "" {
		pgURL = os.Getenv("DATABASE_URL")
	}
	if pgURL == "" {
		return "", fmt.Errorf("no POSTGRES_URL or DATABASE_URL set; cannot apply migration directly")
	}
	cmd := exec.CommandContext(ctx, "psql", pgURL, "-c", sql)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// introspectionQuery is the standard GraphQL introspection query.
const introspectionQuery = `{
  __schema {
    types {
      name
      kind
      fields {
        name
        type { name kind ofType { name kind } }
      }
    }
    queryType { name }
    mutationType { name }
    subscriptionType { name }
  }
}`
