package apidocs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultApiDocsConfig(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	if !cfg.Enabled {
		t.Error("default config should have Enabled=true")
	}
	if cfg.Path != "/docs" {
		t.Errorf("expected Path=/docs, got %q", cfg.Path)
	}
	if cfg.Theme != "default" {
		t.Errorf("expected Theme=default, got %q", cfg.Theme)
	}
	if !cfg.GraphQLEnabled {
		t.Error("default config should have GraphQLEnabled=true")
	}
	if cfg.GraphQLEndpoint != "/v1/graphql" {
		t.Errorf("expected GraphQLEndpoint=/v1/graphql, got %q", cfg.GraphQLEndpoint)
	}
}

func TestBuildSpec_ContainsGraphQL(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	spec := buildSpec("Test API", "example.com", cfg, nil, "/v1/graphql")

	if _, ok := spec.Paths["/v1/graphql"]; !ok {
		t.Error("expected /v1/graphql in paths")
	}
	if _, ok := spec.Paths["/v1/graphql/subscriptions"]; !ok {
		t.Error("expected /v1/graphql/subscriptions in paths")
	}
}

func TestBuildSpec_ContainsAuthEndpoints(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	spec := buildSpec("Test API", "example.com", cfg, nil, "/v1/graphql")

	authPaths := []string{
		"/auth/v1/signup/email-password",
		"/auth/v1/signin/email-password",
		"/auth/v1/signout",
		"/auth/v1/token",
		"/auth/v1/user",
	}
	for _, p := range authPaths {
		if _, ok := spec.Paths[p]; !ok {
			t.Errorf("expected auth path %q in spec", p)
		}
	}
}

func TestBuildSpec_HideEndpoints(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	cfg.HideEndpoints = []string{"/auth/v1"}

	spec := buildSpec("Test API", "example.com", cfg, nil, "/v1/graphql")

	for path := range spec.Paths {
		if strings.HasPrefix(path, "/auth/v1") {
			t.Errorf("hidden path %q should not appear in spec", path)
		}
	}
}

func TestBuildSpec_PluginRoutes(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	routes := []PluginRoute{
		{PluginName: "ai", Method: "POST", Path: "/ai/v1/complete", Summary: "AI completion"},
	}
	spec := buildSpec("Test API", "example.com", cfg, routes, "/v1/graphql")

	pathItem, ok := spec.Paths["/ai/v1/complete"]
	if !ok {
		t.Fatal("expected /ai/v1/complete in paths")
	}
	op, ok := pathItem["post"]
	if !ok {
		t.Fatal("expected POST operation on /ai/v1/complete")
	}
	if op.Summary != "AI completion" {
		t.Errorf("expected summary 'AI completion', got %q", op.Summary)
	}
}

func TestBuildSpec_SecurityScheme(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	spec := buildSpec("Test API", "example.com", cfg, nil, "/v1/graphql")

	scheme, ok := spec.Components.SecuritySchemes["bearerAuth"]
	if !ok {
		t.Fatal("expected bearerAuth security scheme")
	}
	if scheme.Type != "http" {
		t.Errorf("expected scheme type 'http', got %q", scheme.Type)
	}
}

func TestBuildSpec_ValidOpenAPI31(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	spec := buildSpec("Test API", "example.com", cfg, nil, "/v1/graphql")

	if spec.OpenAPI != "3.1.0" {
		t.Errorf("expected openapi version 3.1.0, got %q", spec.OpenAPI)
	}

	// Must round-trip through JSON without error.
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("spec must be JSON-serializable: %v", err)
	}
	var roundTrip OpenAPISpec
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatalf("spec must round-trip through JSON: %v", err)
	}
}

func TestOperationID(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   string
	}{
		{"post", "/auth/v1/signup/email-password", "postAuthV1SignupEmailPassword"},
		{"get", "/auth/v1/user", "getAuthV1User"},
		{"post", "/v1/graphql", "postV1Graphql"},
	}
	for _, tc := range cases {
		got := operationID(tc.method, tc.path)
		if got != tc.want {
			t.Errorf("operationID(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestNginxConf(t *testing.T) {
	conf := NginxConf("/docs", "example.com")
	if !strings.Contains(conf, "server {") {
		t.Error("nginx conf must wrap location blocks in a server { } stanza")
	}
	if !strings.Contains(conf, "server_name docs.example.com") {
		t.Error("nginx conf must serve on docs.<baseDomain>")
	}
	if !strings.Contains(conf, "location /api-docs") {
		t.Error("nginx conf must contain /api-docs location")
	}
	if !strings.Contains(conf, "location /docs") {
		t.Error("nginx conf must contain /docs location")
	}
	if !strings.Contains(conf, "openapi.json") {
		t.Error("nginx conf must reference openapi.json")
	}
	if !strings.Contains(conf, "scalar.html") {
		t.Error("nginx conf must reference scalar.html")
	}
	if !strings.Contains(conf, "/etc/nginx/ssl/certificates/example-com/fullchain.pem") {
		t.Error("nginx conf must reference the per-domain SSL cert directory")
	}
}

func TestRenderScalarHTML(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	cfg.Theme = "moon"
	rendered := renderScalarHTML("title={{.Title}} theme={{.Theme}}", cfg, "My API")
	if !strings.Contains(rendered, "title=My API") {
		t.Error("title not substituted in scalar HTML")
	}
	if !strings.Contains(rendered, "theme=moon") {
		t.Error("theme not substituted in scalar HTML")
	}
}

func TestBuildSpec_GraphQLDisabled(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	cfg.GraphQLEnabled = false

	spec := buildSpec("Test API", "example.com", cfg, nil, "/v1/graphql")

	if _, ok := spec.Paths["/v1/graphql"]; ok {
		t.Error("graphql path should not appear when GraphQL is disabled")
	}
}

func TestBuildSpec_Servers(t *testing.T) {
	cfg := DefaultApiDocsConfig()
	spec := buildSpec("Test API", "my.app", cfg, nil, "/v1/graphql")

	if len(spec.Servers) < 2 {
		t.Fatalf("expected at least 2 servers, got %d", len(spec.Servers))
	}
	found := false
	for _, s := range spec.Servers {
		if strings.Contains(s.URL, "my.app") {
			found = true
		}
	}
	if !found {
		t.Error("expected production server URL to contain baseDomain")
	}
}
