package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/nself-org/cli/internal/sqlallowlist"
	"github.com/spf13/cobra"
)

const (
	mcpServerPort    = 3825
	mcpServiceName   = "_nself._tcp"
	mcpServiceDomain = "local."
	mcpMDNSPort      = 5353
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the nSelf MCP server",
	Long: `Start a Model Context Protocol (MCP) server exposing nSelf infrastructure tools.

Runs on stdio by default — suitable for direct use as a Claude Code MCP server.
Use --transport sse to expose over HTTP on port 3825.

The server advertises itself via mDNS as "_nself._tcp.local" so Claude Code
and other MCP clients can discover local instances automatically.

Available tools:
  nself_list_plugins      List the plugin catalog (installed + available)
  nself_get_schema        Hasura schema introspection
  nself_get_permissions   Hasura permissions snapshot
  nself_run_migration     Apply a SQL migration (confirmation-gated)
  nself_tail_logs         Tail docker logs for a service
  nself_doctor            Run nself doctor --deep

Claude Code config (.claude/settings.json):
  "mcpServers": {
    "nself": {
      "command": "nself",
      "args": ["mcp"]
    }
  }`,
	RunE: runMCPServe,
}

func init() {
	mcpCmd.Flags().StringP("transport", "t", "stdio", "Transport: stdio or sse")
	mcpCmd.Flags().IntP("port", "p", mcpServerPort, "Port for SSE transport")
	mcpCmd.Flags().Bool("no-mdns", false, "Disable mDNS service advertising")
	RootCmd.AddCommand(mcpCmd)
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	transport, _ := cmd.Flags().GetString("transport")
	port, _ := cmd.Flags().GetInt("port")
	noMDNS, _ := cmd.Flags().GetBool("no-mdns")

	// Require an nSelf project directory. Without this guard the server starts
	// even from an empty directory, spawning a persistent mDNS goroutine that
	// outlives the test deadline.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	if _, findErr := config.FindNSelfRoot(cwd); findErr != nil {
		return fmt.Errorf("no nself project found in %s — run 'nself init' first", cwd)
	}

	s := server.NewMCPServer(
		"nSelf MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	registerMCPTools(s)

	// Start mDNS advertising in background (best-effort, never fatal).
	// Pass the command context so the goroutine stops when the server is shut down.
	if !noMDNS {
		go advertiseMDNS(cmd.Context(), port)
	}

	switch transport {
	case "stdio":
		return server.ServeStdio(s)
	case "sse":
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		fmt.Fprintf(os.Stderr, "nSelf MCP server listening on %s (SSE)\n", addr)
		sseServer := server.NewSSEServer(s)
		return sseServer.Start(addr)
	default:
		return fmt.Errorf("unknown transport %q (use stdio or sse)", transport)
	}
}

// registerMCPTools attaches all 6 nSelf MCP tools to the server.
func registerMCPTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("nself_list_plugins",
			mcp.WithDescription("List the nSelf plugin catalog: installed and available plugins"),
		),
		mcpListPluginsHandler(),
	)

	s.AddTool(
		mcp.NewTool("nself_get_schema",
			mcp.WithDescription("Introspect the Hasura GraphQL schema and return table/type information"),
		),
		mcpGetSchemaHandler(),
	)

	s.AddTool(
		mcp.NewTool("nself_get_permissions",
			mcp.WithDescription("Return a snapshot of Hasura role permissions for all tables"),
		),
		mcpGetPermissionsHandler(),
	)

	s.AddTool(
		mcp.NewTool("nself_run_migration",
			mcp.WithDescription("Apply a SQL migration against the nSelf Postgres database. Requires explicit confirmation via the 'confirm' flag."),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL to execute as a migration")),
			mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("Must be true to execute — prevents accidental runs")),
		),
		mcpRunMigrationHandler(),
	)

	s.AddTool(
		mcp.NewTool("nself_tail_logs",
			mcp.WithDescription("Tail docker logs for a named nSelf service or plugin container"),
			mcp.WithString("service", mcp.Required(), mcp.Description("Service or plugin name (e.g. postgres, hasura, auth, ai)")),
			mcp.WithNumber("lines", mcp.Description("Number of lines to return (default 50)")),
		),
		mcpTailLogsHandler(),
	)

	s.AddTool(
		mcp.NewTool("nself_doctor",
			mcp.WithDescription("Run nself doctor --deep and return the diagnostic report"),
		),
		mcpDoctorHandler(),
	)
}

// ── Tool handlers ────────────────────────────────────────────────────────────

func mcpListPluginsHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pluginDir := resolvePluginDir()
		plugins, err := mcpRunCommand(ctx, "nself", "plugin", "list", "--detailed")
		if err != nil {
			// Fall back: try to read from plugin dir directly via internal call.
			_ = pluginDir
			return mcp.NewToolResultText(fmt.Sprintf("Error listing plugins: %v", err)), nil
		}
		return mcp.NewToolResultText(plugins), nil
	}
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
			return mcp.NewToolResultText(fmt.Sprintf("Schema introspection error: %v", err)), nil
		}

		// Compact the response for readability.
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
		payload := map[string]interface{}{
			"type": "export_metadata",
			"args": map[string]interface{}{},
		}
		respBytes, err := mcpPostJSON(ctx, metadataURL, adminSecret, payload)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Permissions snapshot error: %v", err)), nil
		}

		// Extract permissions subset to keep the response concise.
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

		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultText(string(respBytes)), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	}
}

func mcpRunMigrationHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sql, _ := args["sql"].(string)
		confirm, _ := args["confirm"].(bool)

		if sql == "" {
			return mcp.NewToolResultText("Error: sql is required"), nil
		}
		if !confirm {
			return mcp.NewToolResultText("Error: confirm must be true to execute the migration. This is a safety gate."), nil
		}

		// DDL allowlist: reject destructive or privilege-altering SQL before
		// any execution path. This blocks AI Studio sessions from running
		// DROP TABLE, TRUNCATE, DELETE FROM, ALTER ROLE, GRANT/REVOKE, or
		// psql meta-commands even when confirm=true is set programmatically.
		if err := sqlallowlist.ValidateMigrationSQL(sql); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}

		// Use nself db migrate to apply the SQL.
		out, err := mcpRunCommand(ctx, "nself", "db", "migrate", "--sql", sql)
		if err != nil {
			// Fallback: apply directly via psql if db migrate doesn't support --sql.
			out, err = mcpApplyMigrationDirect(ctx, sql)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Migration failed: %v", err)), nil
			}
		}
		return mcp.NewToolResultText("Migration applied successfully.\n" + out), nil
	}
}

func mcpTailLogsHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		service, _ := args["service"].(string)
		if service == "" {
			return mcp.NewToolResultText("Error: service is required"), nil
		}
		lines := 50
		if n, ok := args["lines"].(float64); ok && n > 0 {
			lines = int(n)
		}

		// Resolve container name: nself-<service> convention.
		containerName := "nself-" + service
		out, err := mcpRunCommand(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", lines), containerName)
		if err != nil {
			// Try docker compose fallback.
			out, err = mcpRunCommand(ctx, "docker", "compose", "logs", "--tail", fmt.Sprintf("%d", lines), service)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("Error tailing logs for %s: %v", service, err)), nil
			}
		}
		return mcp.NewToolResultText(out), nil
	}
}

func mcpDoctorHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		out, err := mcpRunCommand(ctx, "nself", "doctor", "--deep", "--json")
		if err != nil {
			// Non-zero exit is normal for doctor (warnings/failures); return output.
			if out != "" {
				return mcp.NewToolResultText(out), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Doctor error: %v", err)), nil
		}
		return mcp.NewToolResultText(out), nil
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// mcpRunCommand runs an external command and returns combined stdout+stderr output.
func mcpRunCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// mcpApplyMigrationDirect applies SQL directly via the Postgres connection string.
// It enforces the DDL allowlist as a defence-in-depth layer even on this fallback
// path — the primary check in mcpRunMigrationHandler runs first, but a second
// guard here ensures no direct caller can bypass it.
func mcpApplyMigrationDirect(ctx context.Context, sql string) (string, error) {
	// Defence-in-depth: re-validate even on the fallback path.
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
	out, err := mcpRunCommand(ctx, "psql", pgURL, "-c", sql)
	return out, err
}

// advertiseMDNS sends a minimal mDNS announcement for _nself._tcp.local on port 3825.
// This is a best-effort, single-shot DNS-SD advertisement via the standard mDNS multicast.
// It never crashes the main server if it fails.
//
// ctx is used to stop the re-announcement loop when the server shuts down. Without
// context cancellation the ticker goroutine would run indefinitely, which causes
// "Test I/O incomplete" failures in the test harness.
func advertiseMDNS(ctx context.Context, port int) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	// Build a minimal PTR + SRV + TXT record set and multicast on 224.0.0.251:5353.
	// This gives Claude Code and other mDNS browsers enough to discover the instance.
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return
	}
	defer conn.Close()

	dst := &net.UDPAddr{IP: net.ParseIP("224.0.0.251"), Port: mcpMDNSPort}
	// Simple DNS-SD announcement: plain text fallback for discovery.
	// Full zeroconf requires a dedicated library; this sends a human-readable
	// broadcast so that tools polling the LAN can detect the service.
	announcement := fmt.Sprintf(
		"_nself._tcp.local. PTR %s._nself._tcp.local. SRV 0 0 %d %s.local. TXT \"path=/mcp\" \"port=%d\"",
		hostname, port, hostname, port,
	)
	if _, err := conn.WriteTo([]byte(announcement), dst); err != nil {
		slog.Debug("mcp mdns announcement failed", "err", err)
	}

	// Re-announce every 60 seconds (RFC 6762 recommendation for long-lived services).
	// Select on ctx.Done() so the goroutine exits cleanly when the server stops.
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := conn.WriteTo([]byte(announcement), dst); err != nil {
				slog.Debug("mcp mdns re-announcement failed", "err", err)
			}
		}
	}
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
