package commands

// mcp_sentry.go — ɳSentry tools for the nSelf MCP server.
//
// Purpose: Let AI agents operate monitoring directly ("add a monitor for X,
//   page me if it goes down") — thin wrappers over internal/sentryapi, the
//   same client the 'nself sentry' commands use.
// Inputs:  MCP tool calls; auth resolved via NSELF_SENTRY_API_KEY /
//   ~/.nself/sentry.json; NSELF_SENTRY_API_URL selects SaaS vs self-hosted.
// Outputs: JSON text results (mcp.CallToolResult); errors returned as text so
//   agents can self-correct (e.g. "run 'nself sentry login'").
// Constraints: read+write tools respect tier quotas server-side; no
//   destructive tool (monitor delete stays CLI-only by design).
// SPORT: CLI-CMD-MCP-002
//
// Example agent prompts these tools enable:
//   "Add an uptime monitor for https://api.example.com checked every minute."
//   "Any open incidents right now? Acknowledge the critical one."
//   "Is our status page fully operational?"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nself-org/cli/internal/sentryapi"
)

// registerSentryMCPTools attaches the 5 ɳSentry tools to the MCP server.
func registerSentryMCPTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("sentry_monitors_list",
			mcp.WithDescription("List ɳSentry uptime monitors for the authenticated tenant (id, name, url, kind, interval, status)"),
		),
		mcpSentryMonitorsListHandler(),
	)

	s.AddTool(
		mcp.NewTool("sentry_monitors_add",
			mcp.WithDescription("Create an ɳSentry uptime monitor. Tier quotas apply (free: 10 @ 5-min)."),
			mcp.WithString("url", mcp.Required(), mcp.Description("Target URL or host to monitor")),
			mcp.WithString("name", mcp.Description("Display name (defaults to the URL)")),
			mcp.WithString("kind", mcp.Description("Monitor kind: http, tcp, or ping (default http)")),
			mcp.WithNumber("interval_seconds", mcp.Description("Check interval in seconds (default 60; tier floors apply)")),
		),
		mcpSentryMonitorsAddHandler(),
	)

	s.AddTool(
		mcp.NewTool("sentry_incidents_list",
			mcp.WithDescription("List ɳSentry incidents, optionally filtered by status (open, acknowledged, resolved)"),
			mcp.WithString("status", mcp.Description("Status filter: open, acknowledged, or resolved (empty = all)")),
		),
		mcpSentryIncidentsListHandler(),
	)

	s.AddTool(
		mcp.NewTool("sentry_incidents_ack",
			mcp.WithDescription("Acknowledge an open ɳSentry incident by id"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Incident id")),
		),
		mcpSentryIncidentsAckHandler(),
	)

	s.AddTool(
		mcp.NewTool("sentry_status",
			mcp.WithDescription("Fetch the ɳSentry public status page summary (overall status + per-component health)"),
			mcp.WithString("url", mcp.Description("Status JSON endpoint (default https://status.nself.org/status.json or NSENTRY_STATUS_URL)")),
		),
		mcpSentryStatusHandler(),
	)
}

// mcpSentryClient builds a sentryapi client from env + credentials file.
func mcpSentryClient() *sentryapi.Client {
	apiURL, apiKey := sentryapi.Resolve("", "")
	return sentryapi.New(apiURL, apiKey)
}

// mcpSentryJSON marshals v for a tool result, or reports the error as text.
func mcpSentryJSON(v interface{}) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error encoding result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func mcpSentryMonitorsListHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		monitors, err := mcpSentryClient().ListMonitors(ctx)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}
		if monitors == nil {
			monitors = []sentryapi.Monitor{}
		}
		return mcpSentryJSON(monitors)
	}
}

func mcpSentryMonitorsAddHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		target, _ := args["url"].(string)
		if target == "" {
			return mcp.NewToolResultText("Error: url is required"), nil
		}
		name, _ := args["name"].(string)
		if name == "" {
			name = target
		}
		kind, _ := args["kind"].(string)
		if kind == "" {
			kind = "http"
		}
		interval := 60
		if n, ok := args["interval_seconds"].(float64); ok && n > 0 {
			interval = int(n)
		}

		m, err := mcpSentryClient().CreateMonitor(ctx, sentryapi.CreateMonitorRequest{
			Name: name, URL: target, Kind: kind, IntervalSeconds: interval,
		})
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}
		return mcpSentryJSON(m)
	}
}

func mcpSentryIncidentsListHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		status, _ := req.GetArguments()["status"].(string)
		incidents, err := mcpSentryClient().ListIncidents(ctx, status)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}
		if incidents == nil {
			incidents = []sentryapi.Incident{}
		}
		return mcpSentryJSON(incidents)
	}
}

func mcpSentryIncidentsAckHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, _ := req.GetArguments()["id"].(string)
		if id == "" {
			return mcp.NewToolResultText("Error: id is required"), nil
		}
		inc, err := mcpSentryClient().AckIncident(ctx, id)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}
		return mcpSentryJSON(inc)
	}
}

func mcpSentryStatusHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		statusURL, _ := req.GetArguments()["url"].(string)
		if statusURL == "" {
			statusURL = os.Getenv("NSENTRY_STATUS_URL")
		}
		if statusURL == "" {
			statusURL = "https://status.nself.org/status.json"
		}

		client := &http.Client{Timeout: 10 * time.Second}
		page, err := fetchStatusPage(ctx, client, statusURL)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}
		return mcpSentryJSON(page)
	}
}
