package commands

// mcp_tools_core.go — read-only MCP tool handlers: status, doctor, urls,
// logs, service list, env list.
//
// Purpose: CLI-R15. The MCP v2 pass adds tools for the commands an agent
//   actually needs to observe an nSelf project, instead of the previous
//   6 Hasura-only tools. Every handler here calls the same internal
//   package the equivalent `nself <cmd>` uses — see mcp.go's
//   registerCoreMCPTools for the tool/description/schema declarations.
//   Config/migration/backup/plugin/deploy read tools live in
//   mcp_tools_ops.go; this file held all of them together until it crossed
//   the 300-line cap.
// Inputs:  MCP tool call arguments; the current working directory (each
//   handler resolves the nSelf project root the same way the CLI commands
//   do, via config.FindNSelfRoot).
// Outputs: mcp.CallToolResult with StructuredContent set to a typed result
//   (see mcp.WithOutputSchema[T]() at each registration site) so an agent
//   can parse the response without guessing a shape from prose.
// Constraints: no handler in this file writes to os.Stdout — several of the
//   analogous cmd/commands/*.go RunE functions print banners/success lines
//   via fmt.Printf or internal/ui (which target os.Stdout), and doing that
//   from inside a tool handler would corrupt the JSON-RPC stream when the
//   server is running over the stdio transport. Handlers here call the
//   pure, non-printing data layer instead (health.RunAllChecks,
//   doctor.DeepChecks, buildURLOutput, service.PS) and let the MCP result
//   carry the data.
// SPORT: CLI-CMD-MCP-001

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/doctor"
	"github.com/nself-org/cli/internal/env"
	"github.com/nself-org/cli/internal/health"
	"github.com/nself-org/cli/internal/service"
)

// mcpLoadProject resolves the current nSelf project root from the working
// directory and loads its config. Shared by every core tool handler so the
// "no project found" error message stays consistent with the CLI commands.
func mcpLoadProject() (*config.Config, string, error) {
	raw, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("getting working directory: %w", err)
	}
	cwd, err := config.FindNSelfRoot(raw)
	if err != nil {
		return nil, "", fmt.Errorf("no nself project found in %s — run 'nself init' first", raw)
	}
	cfg, err := config.Load(cwd)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, cwd, nil
}

// mcpErrorResult wraps an error as a text tool result. MCP tool errors are
// reported as content, not as Go errors, so an agent can read and self-correct
// (returning a Go error instead surfaces as a protocol-level failure).
func mcpErrorResult(format string, args ...interface{}) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf(format, args...)), nil
}

// ── status ───────────────────────────────────────────────────────────────

func mcpStatusHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, cwd, err := mcpLoadProject()
		if err != nil {
			return mcpErrorResult("Error: %v", err)
		}
		report, err := health.RunAllChecks(ctx, cfg, cwd)
		if err != nil {
			return mcpErrorResult("Error running health checks: %v", err)
		}
		return mcp.NewToolResultStructuredOnly(report), nil
	}
}

// ── doctor ───────────────────────────────────────────────────────────────

// DoctorResult wraps doctor.DeepChecks' slice in a top-level object, since
// the MCP output-schema contract requires an object at the root.
type DoctorResult struct {
	Checks []doctor.CheckResult `json:"checks"`
}

func mcpDoctorHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := os.Getwd()
		if err != nil {
			return mcpErrorResult("Error: %v", err)
		}
		root, err := config.FindNSelfRoot(raw)
		if err != nil {
			return mcpErrorResult("no nself project found in %s — run 'nself init' first", raw)
		}
		checks := doctor.DeepChecks(ctx, root, false)
		return mcp.NewToolResultStructuredOnly(DoctorResult{Checks: checks}), nil
	}
}

// ── urls ─────────────────────────────────────────────────────────────────

func mcpURLsHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, _, err := mcpLoadProject()
		if err != nil {
			return mcpErrorResult("Error: %v", err)
		}
		out := buildURLOutput(cfg, true)
		return mcp.NewToolResultStructuredOnly(out), nil
	}
}

// ── logs ─────────────────────────────────────────────────────────────────

// LogsResult is the structured output of the nself_logs tool.
type LogsResult struct {
	Service string `json:"service,omitempty"`
	Lines   int    `json:"lines"`
	Output  string `json:"output"`
}

// mcpLogsHandler runs `docker compose logs` directly (the same external
// binary cmd/commands/logs.go shells out to — docker itself has no Go
// client vendored here) rather than re-exec'ing nself. It never follows
// (a tool call must return), matching a one-shot "give me the tail" use.
func mcpLogsHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		svc, _ := args["service"].(string)
		lines := 50
		if n, ok := args["lines"].(float64); ok && n > 0 {
			lines = int(n)
		}

		_, cwd, err := mcpLoadProject()
		if err != nil {
			return mcpErrorResult("Error: %v", err)
		}

		composeArgs := []string{"compose", "logs", "--no-log-prefix", "--tail", fmt.Sprintf("%d", lines)}
		if svc != "" {
			composeArgs = append(composeArgs, svc)
		}
		out, runErr := mcpRunExternal(ctx, cwd, "docker", composeArgs...)
		if runErr != nil && out == "" {
			return mcpErrorResult("Error tailing logs: %v", runErr)
		}
		return mcp.NewToolResultStructuredOnly(LogsResult{Service: svc, Lines: lines, Output: out}), nil
	}
}

// ── service list ─────────────────────────────────────────────────────────

// ServiceListResult wraps service.PS's slice in a top-level object.
type ServiceListResult struct {
	Services []service.PSEntry `json:"services"`
}

func mcpServiceListHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entries, err := service.PS(ctx)
		if err != nil {
			return mcpErrorResult("Error: %v", err)
		}
		return mcp.NewToolResultStructuredOnly(ServiceListResult{Services: entries}), nil
	}
}

// ── env list ─────────────────────────────────────────────────────────────

// EnvListResult wraps env.List's slice. Only names + active flag + file path
// are exposed — no values, so there is nothing here that needs redaction.
type EnvListResult struct {
	Envs []env.EnvInfo `json:"envs"`
}

func mcpEnvListHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := os.Getwd()
		if err != nil {
			return mcpErrorResult("Error: %v", err)
		}
		root, err := config.FindNSelfRoot(raw)
		if err != nil {
			return mcpErrorResult("no nself project found in %s — run 'nself init' first", raw)
		}
		return mcp.NewToolResultStructuredOnly(EnvListResult{Envs: env.List(root)}), nil
	}
}
