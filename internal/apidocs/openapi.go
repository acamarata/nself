// Package apidocs generates an OpenAPI 3.1 spec from the Hasura introspection
// result and any registered plugin REST routes. The output is written to
// .nself/dist/openapi.json and served by nginx at /api-docs. A self-contained
// Scalar HTML page is served at /docs.
package apidocs

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed scalar.html
var scalarHTML []byte

// ApiDocsConfig is the input type consumed by the apidocs package.
// The build orchestrator maps config.ApiDocsConfig → apidocs.ApiDocsConfig
// to avoid an import cycle (apidocs must not import config).
type ApiDocsConfig struct {
	Enabled         bool
	Path            string
	Title           string
	Theme           string   // default | moon | purple | solarized
	AuthEnvVar      string   // env var containing anon key for try-out pre-fill
	HideEndpoints   []string // path prefixes to exclude
	GraphQLEnabled  bool
	GraphQLEndpoint string // default: /v1/graphql
}

// OpenAPISpec is the top-level OpenAPI 3.1 document.
type OpenAPISpec struct {
	OpenAPI    string                    `json:"openapi"`
	Info       OpenAPIInfo               `json:"info"`
	Servers    []OpenAPIServer           `json:"servers"`
	Paths      map[string]OpenAPIPathItem `json:"paths"`
	Components OpenAPIComponents          `json:"components"`
}

// OpenAPIInfo holds the API metadata.
type OpenAPIInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
	// x-nself-generated marks this as a generated spec.
	Extensions map[string]interface{} `json:"-"`
}

// OpenAPIServer lists base URLs.
type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// OpenAPIPathItem holds operations for a single path.
type OpenAPIPathItem map[string]*OpenAPIOperation

// OpenAPIOperation is one HTTP operation.
type OpenAPIOperation struct {
	Summary     string                   `json:"summary"`
	OperationID string                   `json:"operationId"`
	Tags        []string                 `json:"tags,omitempty"`
	RequestBody *OpenAPIRequestBody       `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
	Security    []map[string][]string     `json:"security,omitempty"`
	// x-nself-websocket marks subscription operations.
	XNselfWebsocket bool `json:"x-nself-websocket,omitempty"`
}

// OpenAPIRequestBody describes the request payload.
type OpenAPIRequestBody struct {
	Required bool                       `json:"required"`
	Content  map[string]OpenAPIMediaType `json:"content"`
}

// OpenAPIMediaType wraps a schema reference.
type OpenAPIMediaType struct {
	Schema OpenAPISchemaRef `json:"schema"`
}

// OpenAPISchemaRef is a JSON Schema reference or inline schema.
type OpenAPISchemaRef struct {
	Ref        string                    `json:"$ref,omitempty"`
	Type       string                    `json:"type,omitempty"`
	Properties map[string]OpenAPISchemaRef `json:"properties,omitempty"`
	Items      *OpenAPISchemaRef          `json:"items,omitempty"`
	Nullable   bool                       `json:"nullable,omitempty"`
}

// OpenAPIResponse describes one response.
type OpenAPIResponse struct {
	Description string                     `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

// OpenAPIComponents holds reusable schemas and security schemes.
type OpenAPIComponents struct {
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
	Schemas         map[string]OpenAPISchemaRef       `json:"schemas,omitempty"`
}

// OpenAPISecurityScheme defines an auth method.
type OpenAPISecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
}

// GenerateResult is returned from Generate.
type GenerateResult struct {
	// OpenAPIPath is the absolute path to the written openapi.json.
	OpenAPIPath string
	// ScalarHTMLPath is the absolute path to the written scalar.html.
	ScalarHTMLPath string
}

// Generate builds the OpenAPI 3.1 spec and writes both dist files:
//   - .nself/dist/openapi.json
//   - .nself/dist/scalar.html
//
// workdir is the nSelf project root (contains .nself/).
// projectName is used as the default API title.
// baseDomain is the primary domain (for the Servers list).
// cfg is the resolved ApiDocsConfig.
// pluginRoutes are REST routes contributed by plugins (from plugin_routes.go).
func Generate(workdir, projectName, baseDomain string, cfg ApiDocsConfig, pluginRoutes []PluginRoute) (*GenerateResult, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	distDir := filepath.Join(workdir, ".nself", "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return nil, fmt.Errorf("creating dist dir: %w", err)
	}

	title := cfg.Title
	if title == "" {
		title = projectName + " API"
	}
	graphqlEndpoint := cfg.GraphQLEndpoint
	if graphqlEndpoint == "" {
		graphqlEndpoint = "/v1/graphql"
	}

	spec := buildSpec(title, baseDomain, cfg, pluginRoutes, graphqlEndpoint)

	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling openapi spec: %w", err)
	}

	openapiPath := filepath.Join(distDir, "openapi.json")
	if err := os.WriteFile(openapiPath, raw, 0644); err != nil {
		return nil, fmt.Errorf("writing openapi.json: %w", err)
	}

	// Render scalar.html with project-specific values substituted.
	scalarRendered := renderScalarHTML(string(scalarHTML), cfg, title)
	scalarPath := filepath.Join(distDir, "scalar.html")
	if err := os.WriteFile(scalarPath, []byte(scalarRendered), 0644); err != nil {
		return nil, fmt.Errorf("writing scalar.html: %w", err)
	}

	return &GenerateResult{
		OpenAPIPath:    openapiPath,
		ScalarHTMLPath: scalarPath,
	}, nil
}

// buildSpec assembles the OpenAPISpec from the available sources.
func buildSpec(title, baseDomain string, cfg ApiDocsConfig, pluginRoutes []PluginRoute, graphqlEndpoint string) OpenAPISpec {
	spec := OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:   title,
			Version: "1.0",
		},
		Servers: []OpenAPIServer{
			{URL: "https://" + baseDomain, Description: "Production"},
			{URL: "http://localhost", Description: "Local dev"},
		},
		Paths: map[string]OpenAPIPathItem{},
		Components: OpenAPIComponents{
			SecuritySchemes: map[string]OpenAPISecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
			Schemas: map[string]OpenAPISchemaRef{},
		},
	}

	defaultSecurity := []map[string][]string{{"bearerAuth": {}}}

	// Auth endpoints — included when auth service is running.
	authEndpoints := map[string]struct {
		method  string
		summary string
		tag     string
		hasBody bool
	}{
		"/auth/v1/signup/email-password": {
			method:  "post",
			summary: "Sign up with email and password",
			tag:     "Auth",
			hasBody: true,
		},
		"/auth/v1/signin/email-password": {
			method:  "post",
			summary: "Sign in with email and password",
			tag:     "Auth",
			hasBody: true,
		},
		"/auth/v1/signout": {
			method:  "post",
			summary: "Sign out",
			tag:     "Auth",
			hasBody: false,
		},
		"/auth/v1/token": {
			method:  "post",
			summary: "Refresh access token",
			tag:     "Auth",
			hasBody: true,
		},
		"/auth/v1/user": {
			method:  "get",
			summary: "Get current user",
			tag:     "Auth",
			hasBody: false,
		},
	}

	for path, ep := range authEndpoints {
		if isHidden(path, cfg.HideEndpoints) {
			continue
		}
		op := &OpenAPIOperation{
			Summary:     ep.summary,
			OperationID: operationID(ep.method, path),
			Tags:        []string{ep.tag},
			Responses: map[string]OpenAPIResponse{
				"200": {Description: "OK", Content: map[string]OpenAPIMediaType{
					"application/json": {Schema: OpenAPISchemaRef{Type: "object"}},
				}},
				"400": {Description: "Bad request"},
				"401": {Description: "Unauthorized"},
			},
			Security: defaultSecurity,
		}
		if ep.hasBody {
			op.RequestBody = &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMediaType{
					"application/json": {Schema: OpenAPISchemaRef{Type: "object"}},
				},
			}
		}
		spec.Paths[path] = OpenAPIPathItem{ep.method: op}
	}

	// GraphQL endpoint — POST /v1/graphql.
	if cfg.GraphQLEnabled && !isHidden(graphqlEndpoint, cfg.HideEndpoints) {
		spec.Paths[graphqlEndpoint] = OpenAPIPathItem{
			"post": {
				Summary:     "GraphQL query / mutation",
				OperationID: "graphqlExecute",
				Tags:        []string{"GraphQL"},
				RequestBody: &OpenAPIRequestBody{
					Required: true,
					Content: map[string]OpenAPIMediaType{
						"application/json": {Schema: OpenAPISchemaRef{
							Type: "object",
							Properties: map[string]OpenAPISchemaRef{
								"query":     {Type: "string"},
								"variables": {Type: "object"},
							},
						}},
					},
				},
				Responses: map[string]OpenAPIResponse{
					"200": {Description: "GraphQL response", Content: map[string]OpenAPIMediaType{
						"application/json": {Schema: OpenAPISchemaRef{Type: "object"}},
					}},
				},
				Security: defaultSecurity,
			},
		}

		// GraphQL subscriptions — documented but marked non-executable.
		wsPath := graphqlEndpoint + "/subscriptions"
		spec.Paths[wsPath] = OpenAPIPathItem{
			"get": {
				Summary:         "GraphQL subscriptions (WebSocket)",
				OperationID:     "graphqlSubscribe",
				Tags:            []string{"GraphQL"},
				XNselfWebsocket: true,
				Responses: map[string]OpenAPIResponse{
					"101": {Description: "WebSocket upgrade"},
				},
			},
		}
	}

	// Plugin-contributed REST routes.
	for _, pr := range pluginRoutes {
		if isHidden(pr.Path, cfg.HideEndpoints) {
			continue
		}
		method := strings.ToLower(pr.Method)
		op := &OpenAPIOperation{
			Summary:     pr.Summary,
			OperationID: operationID(method, pr.Path),
			Tags:        []string{pr.PluginName},
			Responses: map[string]OpenAPIResponse{
				"200": {Description: "OK", Content: map[string]OpenAPIMediaType{
					"application/json": {Schema: OpenAPISchemaRef{Type: "object"}},
				}},
			},
			Security: defaultSecurity,
		}
		if method == "post" || method == "put" || method == "patch" {
			op.RequestBody = &OpenAPIRequestBody{
				Required: true,
				Content: map[string]OpenAPIMediaType{
					"application/json": {Schema: OpenAPISchemaRef{Type: "object"}},
				},
			}
		}
		if _, exists := spec.Paths[pr.Path]; !exists {
			spec.Paths[pr.Path] = OpenAPIPathItem{}
		}
		spec.Paths[pr.Path][method] = op
	}

	return spec
}

// isHidden returns true when the given path matches any entry in the hide list.
func isHidden(path string, hidden []string) bool {
	for _, h := range hidden {
		if strings.HasPrefix(path, h) {
			return true
		}
	}
	return false
}

// operationID derives a safe operationId from method + path.
// e.g. "post /auth/v1/signup/email-password" → "postAuthV1SignupEmailPassword"
func operationID(method, path string) string {
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '-' || r == '_' || r == '.'
	})
	var b bytes.Buffer
	b.WriteString(strings.ToLower(method))
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// renderScalarHTML substitutes template variables into the embedded HTML.
func renderScalarHTML(tmpl string, cfg ApiDocsConfig, title string) string {
	theme := cfg.Theme
	if theme == "" {
		theme = "default"
	}
	docsPath := cfg.Path
	if docsPath == "" {
		docsPath = "/docs"
	}
	_ = docsPath // used by nginx conf; scalar page self-references via relative URL

	tmpl = strings.ReplaceAll(tmpl, "{{.Title}}", title)
	tmpl = strings.ReplaceAll(tmpl, "{{.Theme}}", theme)
	return tmpl
}

// DefaultApiDocsConfig returns defaults when api_docs is not configured.
func DefaultApiDocsConfig() ApiDocsConfig {
	return ApiDocsConfig{
		Enabled:         true,
		Path:            "/docs",
		Theme:           "default",
		GraphQLEnabled:  true,
		GraphQLEndpoint: "/v1/graphql",
	}
}

// NginxConf returns the nginx site configuration for /api-docs and /docs,
// served on docs.<baseDomain>. Output is a complete server { } block intended
// to live in nginx/sites/ alongside hasura.conf and auth.conf.
//
// docsServePath is the path on the docs subdomain (default /docs); /api-docs
// is always served alongside it.
func NginxConf(docsServePath, baseDomain string) string {
	if docsServePath == "" {
		docsServePath = "/docs"
	}
	if baseDomain == "" {
		baseDomain = "localhost"
	}
	certDir := strings.ReplaceAll(baseDomain, ".", "-")
	return fmt.Sprintf(`# Generated by nself build - DO NOT EDIT MANUALLY
# Service: docs.%s

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name docs.%s;

    ssl_certificate /etc/nginx/ssl/certificates/%s/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/certificates/%s/privkey.pem;

    server_tokens off;

    location /api-docs {
        alias /opt/nself/.dist/openapi.json;
        add_header Content-Type "application/json";
        add_header Access-Control-Allow-Origin *;
        add_header Cache-Control "no-store";
    }
    location %s {
        alias /opt/nself/.dist/scalar.html;
        add_header Content-Type "text/html";
        add_header Cache-Control "no-store";
    }
}
`, baseDomain, baseDomain, certDir, certDir, docsServePath)
}
