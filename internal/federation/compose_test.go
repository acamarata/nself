package federation

import (
	"strings"
	"testing"
)

// TestSupergraphYAML_FromManifests verifies that three plugin manifests with
// graphql.enabled=true produce a supergraph.yaml containing the correct
// subgraph_url and schema_path entries for each.
func TestSupergraphYAML_FromManifests(t *testing.T) {
	blocks := []ManifestGraphQLBlock{
		{
			PluginName: "ai",
			PluginPort: 8100,
			GraphQL: SubgraphConfig{
				Enabled:      true,
				SubgraphName: "ai",
				SubgraphURL:  "http://localhost:${PORT}/graphql",
				SchemaPath:   "./go/schema.graphql",
			},
		},
		{
			PluginName: "claw",
			PluginPort: 8101,
			GraphQL: SubgraphConfig{
				Enabled:      true,
				SubgraphName: "claw",
				SubgraphURL:  "http://localhost:${PORT}/graphql",
			},
		},
		{
			PluginName: "mux",
			PluginPort: 8102,
			GraphQL: SubgraphConfig{
				Enabled:      true,
				SubgraphName: "mux",
				SubgraphURL:  "http://localhost:${PORT}/graphql",
				SchemaPath:   "./go/schema.graphql",
			},
		},
	}

	entries, err := FromManifestBlocks(blocks, "")
	if err != nil {
		t.Fatalf("FromManifestBlocks: %v", err)
	}
	// hasura + 3 plugins
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	yaml, err := BuildSupergraphYAML(entries)
	if err != nil {
		t.Fatalf("BuildSupergraphYAML: %v", err)
	}

	yamlStr := string(yaml)

	// Verify each subgraph appears with correct URL.
	for _, want := range []struct{ name, url string }{
		{"hasura", "http://localhost:8080/v1/graphql"},
		{"ai", "http://localhost:8100/graphql"},
		{"claw", "http://localhost:8101/graphql"},
		{"mux", "http://localhost:8102/graphql"},
	} {
		if !strings.Contains(yamlStr, want.name+":") {
			t.Errorf("supergraph YAML missing subgraph block %q", want.name)
		}
		if !strings.Contains(yamlStr, want.url) {
			t.Errorf("supergraph YAML missing URL %q for %q", want.url, want.name)
		}
	}

	// Verify schema_path for ai and mux (not claw which has no schema_path).
	if !strings.Contains(yamlStr, "./go/schema.graphql") {
		t.Error("supergraph YAML missing schema_path for ai/mux")
	}
}

// TestSupergraphYAML_DisabledPluginsExcluded verifies that a plugin with
// graphql.enabled=false does not appear in the generated supergraph YAML.
func TestSupergraphYAML_DisabledPluginsExcluded(t *testing.T) {
	blocks := []ManifestGraphQLBlock{
		{
			PluginName: "ai",
			PluginPort: 8100,
			GraphQL: SubgraphConfig{
				Enabled:      true,
				SubgraphName: "ai",
				SubgraphURL:  "http://localhost:8100/graphql",
			},
		},
		{
			PluginName: "disabled-plugin",
			PluginPort: 8199,
			GraphQL: SubgraphConfig{
				Enabled:      false,
				SubgraphName: "disabled",
				SubgraphURL:  "http://localhost:8199/graphql",
			},
		},
	}

	entries, err := FromManifestBlocks(blocks, "")
	if err != nil {
		t.Fatalf("FromManifestBlocks: %v", err)
	}

	yaml, err := BuildSupergraphYAML(entries)
	if err != nil {
		t.Fatalf("BuildSupergraphYAML: %v", err)
	}

	yamlStr := string(yaml)
	if strings.Contains(yamlStr, "disabled") {
		t.Error("disabled plugin appeared in supergraph YAML — should be excluded")
	}
	if !strings.Contains(yamlStr, "ai") {
		t.Error("enabled plugin ai missing from supergraph YAML")
	}
}

// TestNginxDirective_FederationMode verifies that NginxDirective with
// federationEnabled=true proxies /v1/graphql to port 4000 (Apollo Router).
func TestNginxDirective_FederationMode(t *testing.T) {
	out := NginxDirective(true)
	if !strings.Contains(out, "127.0.0.1:4000") {
		t.Errorf("federation nginx directive must proxy to port 4000, got:\n%s", out)
	}
	if strings.Contains(out, "8080") {
		t.Errorf("federation nginx directive must NOT reference Hasura port 8080, got:\n%s", out)
	}
}

// TestNginxDirective_NonFederationMode verifies that NginxDirective with
// federationEnabled=false proxies /v1/graphql to port 8080 (Hasura).
func TestNginxDirective_NonFederationMode(t *testing.T) {
	out := NginxDirective(false)
	if !strings.Contains(out, "127.0.0.1:8080") {
		t.Errorf("non-federation nginx directive must proxy to Hasura port 8080, got:\n%s", out)
	}
	if strings.Contains(out, "4000") {
		t.Errorf("non-federation nginx directive must NOT reference Apollo Router port 4000, got:\n%s", out)
	}
}

// TestFromManifestBlocks_DuplicateSubgraphName verifies that two plugins
// with the same subgraph_name produce an error (fail-closed).
func TestFromManifestBlocks_DuplicateSubgraphName(t *testing.T) {
	blocks := []ManifestGraphQLBlock{
		{
			PluginName: "ai",
			PluginPort: 8100,
			GraphQL: SubgraphConfig{
				Enabled:      true,
				SubgraphName: "shared",
				SubgraphURL:  "http://localhost:8100/graphql",
			},
		},
		{
			PluginName: "claw",
			PluginPort: 8101,
			GraphQL: SubgraphConfig{
				Enabled:      true,
				SubgraphName: "shared",
				SubgraphURL:  "http://localhost:8101/graphql",
			},
		},
	}
	_, err := FromManifestBlocks(blocks, "")
	if err == nil {
		t.Fatal("expected error for duplicate subgraph name, got nil")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("error message should mention 'already registered', got: %v", err)
	}
}

// TestFromManifestBlocks_HasuraNameReserved verifies that a plugin may not
// claim the "hasura" subgraph name.
func TestFromManifestBlocks_HasuraNameReserved(t *testing.T) {
	blocks := []ManifestGraphQLBlock{
		{
			PluginName: "evil",
			PluginPort: 9999,
			GraphQL: SubgraphConfig{
				Enabled:      true,
				SubgraphName: "hasura",
				SubgraphURL:  "http://localhost:9999/graphql",
			},
		},
	}
	_, err := FromManifestBlocks(blocks, "")
	if err == nil {
		t.Fatal("expected error for reserved 'hasura' name, got nil")
	}
	if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("error message should mention 'reserved', got: %v", err)
	}
}

// TestInterpolatePort verifies ${PORT} substitution in subgraph URLs.
func TestInterpolatePort(t *testing.T) {
	cases := []struct {
		url      string
		port     int
		expected string
	}{
		{"http://localhost:${PORT}/graphql", 8100, "http://localhost:8100/graphql"},
		{"http://localhost:8100/graphql", 8100, "http://localhost:8100/graphql"},
		{"http://localhost:${PORT}/graphql", 0, "http://localhost:${PORT}/graphql"},
	}
	for _, c := range cases {
		got := interpolatePort(c.url, c.port)
		if got != c.expected {
			t.Errorf("interpolatePort(%q, %d) = %q, want %q", c.url, c.port, got, c.expected)
		}
	}
}

// TestComposeService_RendersCorrectly verifies that ComposeService produces a
// YAML block containing the CS_7 label and port 4000.
func TestComposeService_RendersCorrectly(t *testing.T) {
	out, err := ComposeService(RouterComposeOptions{
		SupergraphPath:   "/home/user/.nself/federation/supergraph.graphql",
		RouterConfigPath: "/home/user/.nself/federation/router.yaml",
	})
	if err != nil {
		t.Fatalf("ComposeService: %v", err)
	}
	for _, want := range []string{"apollo-router", "4000", "CS_7", "nself.federation"} {
		if !strings.Contains(out, want) {
			t.Errorf("ComposeService output missing %q:\n%s", want, out)
		}
	}
}

// TestComposeService_EmptyPathsError verifies that missing paths return errors.
func TestComposeService_EmptyPathsError(t *testing.T) {
	_, err := ComposeService(RouterComposeOptions{RouterConfigPath: "/etc/router.yaml"})
	if err == nil {
		t.Error("expected error for empty SupergraphPath")
	}
	_, err = ComposeService(RouterComposeOptions{SupergraphPath: "/etc/supergraph.graphql"})
	if err == nil {
		t.Error("expected error for empty RouterConfigPath")
	}
}
