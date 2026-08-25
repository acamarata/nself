package apidocs

// openapi_spec.go — building the OpenAPI spec from Hasura introspection.
//
// Purpose: assemble the OpenAPISpec paths and schemas from the Hasura introspection result and registered plugin routes, used by Generate in openapi.go, split out for file size.
// Inputs: the Hasura introspection result and plugin route metadata.
// Outputs: an OpenAPISpec with populated paths and components.
// Constraints: pure move from openapi.go (CLI-R12 Batch E); no behaviour change.

import (
	"bytes"
	_ "embed"
	"strings"
)

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
