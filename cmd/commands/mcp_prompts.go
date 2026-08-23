package commands

// mcp_prompts.go — MCP prompts for the nSelf MCP server.
//
// Purpose: CLI-R15's three prompt templates. A prompt is a canned starting
//   message that tells an agent which tools/resources to reach for and in
//   what order, for a task shape common enough to be worth naming.
// Inputs:  prompt arguments (see each Prompt's WithArgument declarations).
// Outputs: mcp.GetPromptResult with a single user-role text message.
// Constraints: prompts only ever reference tools/resources this server
//   actually registers — no prompt tells an agent to call something that
//   doesn't exist.
// SPORT: CLI-CMD-MCP-001

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerMCPPrompts attaches the three nSelf prompt templates.
func registerMCPPrompts(s *server.MCPServer) {
	s.AddPrompt(
		mcp.NewPrompt("diagnose-failure",
			mcp.WithPromptDescription("Investigate why an nSelf service is unhealthy or a project won't start"),
			mcp.WithArgument("service", mcp.ArgumentDescription("Service name to focus on (optional — omit to check the whole stack)")),
		),
		mcpDiagnoseFailurePrompt(),
	)
	s.AddPrompt(
		mcp.NewPrompt("add-service",
			mcp.WithPromptDescription("Enable an optional service (or install a plugin) and bring it up safely"),
			mcp.WithArgument("name", mcp.ArgumentDescription("Service or plugin name to add"), mcp.RequiredArgument()),
		),
		mcpAddServicePrompt(),
	)
	s.AddPrompt(
		mcp.NewPrompt("prepare-deploy",
			mcp.WithPromptDescription("Run the pre-deploy checklist before promoting a project to staging or prod"),
			mcp.WithArgument("env", mcp.ArgumentDescription("Target environment (staging or prod; default staging)")),
		),
		mcpPrepareDeployPrompt(),
	)
}

func mcpUserPrompt(description, text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []mcp.PromptMessage{
			{Role: mcp.RoleUser, Content: mcp.TextContent{Type: "text", Text: text}},
		},
	}
}

func mcpDiagnoseFailurePrompt() server.PromptHandlerFunc {
	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		service := req.Params.Arguments["service"]
		focus := "the whole stack"
		if service != "" {
			focus = fmt.Sprintf("the %q service", service)
		}
		text := fmt.Sprintf(`Diagnose why %s is failing.

1. Call nself_status to see per-service health.
2. Call nself_doctor for the full 12-section diagnostic (Docker, disk, ports, certs, etc.).
3. Call nself_logs%s to read recent output.
4. Cross-reference: a service stuck in "starting" usually means its healthcheck
   dependency (often postgres or hasura) isn't healthy yet — check that one first.
5. If a fix requires a config change, use nself_config_set, then nself_build and
   nself_restart to apply it.
6. Re-run nself_status to confirm the fix.`, focus, promptServiceArgSuffix(service))
		return mcpUserPrompt("Diagnose an unhealthy nSelf service", text), nil
	}
}

func promptServiceArgSuffix(service string) string {
	if service == "" {
		return ""
	}
	return fmt.Sprintf(" with service=%q", service)
}

func mcpAddServicePrompt() server.PromptHandlerFunc {
	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		name := req.Params.Arguments["name"]
		text := fmt.Sprintf(`Add %q to this nSelf project.

1. Read the nself://services resource to check whether %q is a built-in optional
   service (has an EnableEnv) or needs a plugin instead.
2. If it's a built-in optional service: call nself_config_set with its EnableEnv
   key set to "true".
3. If it's not in the catalog: call nself_plugin_list to check whether it's
   already installed, then nself_plugin_install with name=%q if not.
4. Call nself_build to regenerate docker-compose.yml and nginx config for the
   new service.
5. Call nself_start (safe to re-run — it's idempotent) to bring it up.
6. Call nself_status and nself_urls to confirm %q is healthy and routed.`, name, name, name, name)
		return mcpUserPrompt("Enable a new nSelf service or plugin", text), nil
	}
}

func mcpPrepareDeployPrompt() server.PromptHandlerFunc {
	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		env := req.Params.Arguments["env"]
		if env == "" {
			env = "staging"
		}
		text := fmt.Sprintf(`Run the pre-deploy checklist for %s before promoting this project.

1. Call nself_doctor and confirm there are no "fail" status checks.
2. Call nself_db_migrate_status and confirm there are no pending migrations
   that should have shipped with this change.
3. Call nself_backup_list and confirm a recent backup exists — take one first
   via the CLI ('nself backup create') if the most recent is stale.
4. Call nself_deploy_status for a fast local read of current state (this tool
   does not SSH-probe remote servers; use 'nself deploy status' from the CLI
   for the full picture).
5. Deployment itself (SSH to remote hosts, promote, rollback) is intentionally
   NOT exposed as an MCP tool — hand the checklist result to the operator and
   have them run 'nself deploy' / 'nself promote' from the CLI.`, env)
		return mcpUserPrompt("Pre-deploy checklist", text), nil
	}
}
