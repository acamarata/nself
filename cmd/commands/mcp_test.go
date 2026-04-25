package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TestMCPCmdRegistered verifies that the mcp subcommand is registered on RootCmd.
func TestMCPCmdRegistered(t *testing.T) {
	for _, c := range RootCmd.Commands() {
		if c.Name() == "mcp" {
			return
		}
	}
	t.Fatal("mcp command not registered on RootCmd")
}

// TestMCPCmdFlags verifies the expected flags are declared.
func TestMCPCmdFlags(t *testing.T) {
	flags := []string{"transport", "port", "no-mdns"}
	for _, name := range flags {
		if mcpCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not declared on mcp command", name)
		}
	}
}

// TestRegisterMCPTools checks that all 6 tools are registered.
func TestRegisterMCPTools(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.1", server.WithToolCapabilities(true))
	registerMCPTools(s)

	wantTools := []string{
		"nself_list_plugins",
		"nself_get_schema",
		"nself_get_permissions",
		"nself_run_migration",
		"nself_tail_logs",
		"nself_doctor",
	}

	// Verify via the MCP server's list-tools capability.
	ctx := context.Background()
	result := s.HandleMessage(ctx, json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal tools/list result: %v", err)
	}

	for _, name := range wantTools {
		if !strings.Contains(string(resultJSON), name) {
			t.Errorf("tool %q not found in tools/list response", name)
		}
	}
}

// TestMCPHandshake simulates the MCP initialize handshake over the server.
func TestMCPHandshake(t *testing.T) {
	s := server.NewMCPServer("nSelf MCP Server", "1.0.0", server.WithToolCapabilities(true))
	registerMCPTools(s)

	ctx := context.Background()
	initMsg := json.RawMessage(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test-client", "version": "0.0.1"}
		}
	}`)

	result := s.HandleMessage(ctx, initMsg)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal initialize result: %v", err)
	}

	// Response must contain the server name and capabilities.
	resultStr := string(resultJSON)
	if !strings.Contains(resultStr, "nSelf MCP Server") {
		t.Errorf("server name not found in initialize response: %s", resultStr)
	}
	if !strings.Contains(resultStr, "tools") {
		t.Errorf("tools capability not found in initialize response: %s", resultStr)
	}
}

// TestMCPMigrationConfirmGate verifies the confirmation gate on nself_run_migration.
func TestMCPMigrationConfirmGate(t *testing.T) {
	handler := mcpRunMigrationHandler()

	// Without confirm=true, should be rejected.
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"sql":     "ALTER TABLE np_plugins ADD COLUMN test TEXT;",
		"confirm": false,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Must mention the safety gate.
	content := extractTextContent(result)
	if !strings.Contains(content, "confirm") && !strings.Contains(content, "safety") {
		t.Errorf("expected rejection message, got: %s", content)
	}
}

// TestMCPMigrationNoSQL verifies that empty SQL is rejected.
func TestMCPMigrationNoSQL(t *testing.T) {
	handler := mcpRunMigrationHandler()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"sql":     "",
		"confirm": true,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	content := extractTextContent(result)
	if !strings.Contains(strings.ToLower(content), "sql") {
		t.Errorf("expected sql-required error, got: %s", content)
	}
}

// TestMCPTailLogsNoService verifies that empty service name is rejected.
func TestMCPTailLogsNoService(t *testing.T) {
	handler := mcpTailLogsHandler()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	content := extractTextContent(result)
	if !strings.Contains(strings.ToLower(content), "service") {
		t.Errorf("expected service-required error, got: %s", content)
	}
}

// TestMCPGetSchemaReachesHasura verifies the schema handler calls Hasura correctly.
func TestMCPGetSchemaReachesHasura(t *testing.T) {
	// Set up a fake Hasura server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/graphql" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"__schema":{"types":[],"queryType":{"name":"query_root"}}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	t.Setenv("NSELF_HASURA_GRAPHQL_URL", ts.URL)

	handler := mcpGetSchemaHandler()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	content := extractTextContent(result)
	if !strings.Contains(content, "query_root") {
		t.Errorf("expected schema content, got: %s", content)
	}
}

// TestMCPGetPermissionsReachesHasura verifies the permissions handler calls Hasura metadata API.
func TestMCPGetPermissionsReachesHasura(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/metadata" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"sources":[{"tables":[{"table":{"name":"np_plugins"},"select_permissions":[{"role":"user"}]}]}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	t.Setenv("NSELF_HASURA_GRAPHQL_URL", ts.URL)

	handler := mcpGetPermissionsHandler()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	content := extractTextContent(result)
	if !strings.Contains(content, "np_plugins") {
		t.Errorf("expected permissions content, got: %s", content)
	}
}

// TestMCPResolveHasuraEndpoint checks env-var resolution order.
func TestMCPResolveHasuraEndpoint(t *testing.T) {
	// Default fallback.
	t.Setenv("NSELF_HASURA_GRAPHQL_URL", "")
	t.Setenv("HASURA_GRAPHQL_URL", "")
	got := resolveHasuraEndpoint()
	if got != "http://127.0.0.1:8080" {
		t.Errorf("expected default localhost URL, got %q", got)
	}

	// NSELF_HASURA_GRAPHQL_URL takes priority.
	t.Setenv("NSELF_HASURA_GRAPHQL_URL", "http://example.com:8080")
	got = resolveHasuraEndpoint()
	if got != "http://example.com:8080" {
		t.Errorf("expected NSELF_HASURA_GRAPHQL_URL, got %q", got)
	}
}

// TestMCPPortConstant verifies the port constant matches the spec.
func TestMCPPortConstant(t *testing.T) {
	if mcpServerPort != 3825 {
		t.Errorf("mcpServerPort must be 3825 (spec), got %d", mcpServerPort)
	}
}

// TestMCPMDNSServiceName verifies the mDNS service type matches the spec.
func TestMCPMDNSServiceName(t *testing.T) {
	if mcpServiceName != "_nself._tcp" {
		t.Errorf("mcpServiceName must be _nself._tcp (spec), got %q", mcpServiceName)
	}
}

// extractTextContent pulls the text from the first text content block.
// Content is a sealed interface; use AsTextContent helper from the mcp package.
func extractTextContent(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			return tc.Text
		}
	}
	return ""
}
