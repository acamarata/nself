// Purpose:     decide which top-level commands are covered by an MCP tool.
//
// Inputs:      the two files that register MCP tools: cmd/commands/mcp.go
//
//	(6 infra tools) and cmd/commands/mcp_sentry.go (5 ɳSentry tools),
//	read as plain text — this tool does not import cmd/commands.
//
// Outputs:     a set of command names considered MCP-covered.
//
// Constraints: MATCHING RULE (deliberately conservative, exact-token only —
//
//	no fuzzy/substring matching, so it never over-claims coverage):
//	every mcp.NewTool("tool_name", ...) call is split on '_' into
//	tokens (e.g. "sentry_monitors_list" -> sentry, monitors, list).
//	A command is covered iff its exact Name appears as one of those
//	tokens for ANY registered tool. This intentionally does NOT
//	credit near-misses: nself_run_migration does not cover "migrate"
//	(token is "migration", not "migrate"), nself_list_plugins does not
//	cover "plugin" (token is "plugins", not "plugin"). It DOES credit
//	nself_doctor -> doctor, nself_tail_logs -> logs, sentry_* -> sentry,
//	and sentry_status -> status. Verified by main_test.go against the
//	actual tool names registered as of CLI-R17.
package main

import (
	"os"
	"regexp"
	"strings"
)

var mcpToolNameRe = regexp.MustCompile(`mcp\.NewTool\(\s*"([a-zA-Z0-9_]+)"`)

// mcpToolTokens reads the given files, extracts every registered MCP tool
// name, and returns the set of command names it credits with coverage.
func mcpToolTokens(paths ...string) (map[string]bool, error) {
	tokens := map[string]bool{}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		for _, m := range mcpToolNameRe.FindAllStringSubmatch(string(data), -1) {
			for _, tok := range strings.Split(m[1], "_") {
				if tok != "" {
					tokens[tok] = true
				}
			}
		}
	}
	return tokens, nil
}
