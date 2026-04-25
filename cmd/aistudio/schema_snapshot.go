package aistudio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// SchemaContext is the payload injected in X-Nself-Schema-Context.
type SchemaContext struct {
	Tables      []TableInfo `json:"tables"`
	ActiveRole  string      `json:"active_role"`
	SchemaHash  string      `json:"schema_hash,omitempty"`
	CapturedAt  string      `json:"captured_at"`
}

// TableInfo holds minimal schema information for one table.
type TableInfo struct {
	Name    string       `json:"name"`
	Schema  string       `json:"schema"`
	Columns []ColumnInfo `json:"columns"`
}

// ColumnInfo holds column metadata.
type ColumnInfo struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
}

// introspectionQuery is a minimal GraphQL introspection query targeting table metadata.
const introspectionQuery = `{
  __schema {
    queryType { name }
    types {
      name
      kind
      fields {
        name
        type {
          name
          kind
          ofType { name kind }
        }
      }
    }
  }
}`

// buildSchemaContext queries Hasura introspection and returns an encoded context header value.
func buildSchemaContext(ctx context.Context, hasuraURL, adminSecret string) (string, error) {
	sc, err := fetchSchema(ctx, hasuraURL, adminSecret)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(sc)
	if err != nil {
		return "", fmt.Errorf("marshal schema context: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// fetchSchema runs an introspection query and builds a SchemaContext.
func fetchSchema(ctx context.Context, hasuraURL, adminSecret string) (*SchemaContext, error) {
	body, _ := json.Marshal(map[string]string{"query": introspectionQuery})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hasuraURL+"/v1/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build schema request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if adminSecret != "" {
		req.Header.Set("x-hasura-admin-secret", adminSecret)
	}
	// Use read-only role for schema inspection.
	req.Header.Set("X-Hasura-Role", "ai_studio_read")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schema introspection request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read schema response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("schema introspection: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Data struct {
			Schema struct {
				Types []struct {
					Name   string `json:"name"`
					Kind   string `json:"kind"`
					Fields []struct {
						Name string `json:"name"`
						Type struct {
							Name   string `json:"name"`
							Kind   string `json:"kind"`
							OfType *struct {
								Name string `json:"name"`
								Kind string `json:"kind"`
							} `json:"ofType"`
						} `json:"type"`
					} `json:"fields"`
				} `json:"types"`
			} `json:"__schema"`
		} `json:"data"`
	}

	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse schema response: %w", err)
	}

	sc := &SchemaContext{
		ActiveRole: "ai_studio_read",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
	}
	for _, t := range result.Data.Schema.Types {
		if t.Kind != "OBJECT" || len(t.Name) == 0 || t.Name[0] == '_' {
			continue
		}
		ti := TableInfo{Name: t.Name, Schema: "public"}
		for _, f := range t.Fields {
			typeName := f.Type.Name
			if typeName == "" && f.Type.OfType != nil {
				typeName = f.Type.OfType.Name
			}
			ti.Columns = append(ti.Columns, ColumnInfo{Name: f.Name, DataType: typeName})
		}
		sc.Tables = append(sc.Tables, ti)
	}
	return sc, nil
}

// hasuraURLFromEnv reads the Hasura URL from the environment with a default fallback.
func hasuraURLFromEnv() string {
	if v := os.Getenv("HASURA_GRAPHQL_ENDPOINT"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// hasuraAdminSecretFromEnv reads the Hasura admin secret from env.
func hasuraAdminSecretFromEnv() string {
	return os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")
}
