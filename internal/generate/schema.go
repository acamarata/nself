// Package generate provides the schema codegen engine for nself generate.
//
// It fetches the live Hasura GraphQL schema via introspection, parses it into
// an internal IR (tables, enums, relationships, operations), and renders
// type-safe client code for TypeScript, Dart, Swift, Kotlin, and Python.
package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nself-org/cli/internal/httptimeout"
)

// introspectionQuery is the standard GraphQL introspection query.
// It fetches types, fields, and their nullability/list shapes.
const introspectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args { ...InputValue }
    type { ...TypeRef }
    isDeprecated
    deprecationReason
  }
  inputFields { ...InputValue }
  interfaces { ...TypeRef }
  enumValues(includeDeprecated: true) {
    name
    isDeprecated
    deprecationReason
  }
  possibleTypes { ...TypeRef }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType { kind name }
        }
      }
    }
  }
}
`

// introspectionResponse is the top-level JSON envelope from Hasura.
type introspectionResponse struct {
	Data struct {
		Schema introspectionSchema `json:"__schema"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// introspectionSchema mirrors the __schema type from GraphQL introspection.
type introspectionSchema struct {
	QueryType        *namedType  `json:"queryType"`
	MutationType     *namedType  `json:"mutationType"`
	SubscriptionType *namedType  `json:"subscriptionType"`
	Types            []introType `json:"types"`
}

type namedType struct {
	Name string `json:"name"`
}

// introType represents one GraphQL type from the introspection result.
type introType struct {
	Kind        string       `json:"kind"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Fields      []introField `json:"fields"`
	InputFields []introInput `json:"inputFields"`
	EnumValues  []introEnum  `json:"enumValues"`
}

type introField struct {
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	Type              introRef     `json:"type"`
	IsDeprecated      bool         `json:"isDeprecated"`
	DeprecationReason string       `json:"deprecationReason"`
	Args              []introInput `json:"args"`
}

type introInput struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         introRef `json:"type"`
	DefaultValue string   `json:"defaultValue"`
}

type introEnum struct {
	Name         string `json:"name"`
	IsDeprecated bool   `json:"isDeprecated"`
}

// introRef is a recursive type reference (wraps NON_NULL, LIST, SCALAR, etc.).
type introRef struct {
	Kind   string    `json:"kind"`
	Name   string    `json:"name"`
	OfType *introRef `json:"ofType"`
}

// isNullable returns true when the outermost wrapper is not NON_NULL.
func (r introRef) isNullable() bool {
	return r.Kind != "NON_NULL"
}

// scalarName descends through NON_NULL/LIST wrappers and returns the named type.
func (r introRef) scalarName() string {
	if r.Name != "" {
		return r.Name
	}
	if r.OfType != nil {
		return r.OfType.scalarName()
	}
	return "unknown"
}

// FetchSchema sends an introspection query to the Hasura GraphQL endpoint and
// returns the raw parsed introspection response.
func FetchSchema(ctx context.Context, hasuraURL, adminSecret string) (*introspectionSchema, error) {
	payload, err := json.Marshal(map[string]string{"query": introspectionQuery})
	if err != nil {
		return nil, fmt.Errorf("marshal introspection query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hasuraURL+"/v1/graphql", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create introspection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", adminSecret)

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read introspection response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hasura returned HTTP %d: %s", resp.StatusCode, body)
	}

	var result introspectionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse introspection response: %w", err)
	}

	if len(result.Errors) > 0 {
		msgs := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			msgs[i] = e.Message
		}
		return nil, fmt.Errorf("hasura introspection errors: %s", strings.Join(msgs, "; "))
	}

	return &result.Data.Schema, nil
}
