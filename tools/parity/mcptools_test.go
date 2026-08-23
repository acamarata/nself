package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMcpToolTokensAgainstFixture pins the exact-token matching rule
// documented in mcptools.go against a small fixture mirroring the real
// mcp.go/mcp_sentry.go shape, so a future edit to the matching rule (or an
// accidental loosening to substring matching) fails loudly here instead of
// silently changing what CI reports.
func TestMcpToolTokensAgainstFixture(t *testing.T) {
	dir := t.TempDir()
	mainFile := writeTempFile(t, dir, "mcp.go", `package commands
func registerMCPTools() {
	s.AddTool(mcp.NewTool("nself_list_plugins", mcp.WithDescription("x")), h())
	s.AddTool(mcp.NewTool("nself_run_migration", mcp.WithDescription("x")), h())
	s.AddTool(mcp.NewTool("nself_tail_logs", mcp.WithDescription("x")), h())
	s.AddTool(mcp.NewTool("nself_doctor", mcp.WithDescription("x")), h())
}
`)
	sentryFile := writeTempFile(t, dir, "mcp_sentry.go", `package commands
func registerSentryMCPTools() {
	s.AddTool(mcp.NewTool("sentry_monitors_list", mcp.WithDescription("x")), h())
	s.AddTool(mcp.NewTool("sentry_status", mcp.WithDescription("x")), h())
}
`)

	tokens, err := mcpToolTokens(mainFile, sentryFile)
	if err != nil {
		t.Fatalf("mcpToolTokens: %v", err)
	}

	credited := []string{"doctor", "logs", "sentry", "status"}
	for _, name := range credited {
		if !tokens[name] {
			t.Errorf("expected %q to be MCP-covered, got false", name)
		}
	}

	// "migrate" (vs. token "migration") and "plugin" (vs. token "plugins")
	// are the real near-misses the exact-token rule is designed to reject —
	// tokens like "run"/"tail"/"monitors" ARE credited (they're literal
	// tokens), they just don't happen to name real top-level commands.
	notCredited := []string{"migrate", "plugin"}
	for _, name := range notCredited {
		if tokens[name] {
			t.Errorf("expected %q to NOT be MCP-covered (exact-token rule), got true", name)
		}
	}
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
