package database

// Purpose: nself db verify --role <role> — role-scoped GraphQL introspection
// against a live Hasura instance, answering "is this feature reachable by an
// actual role" without a hand-rolled curl command.
// Inputs: context, *config.Config, role name.
// Outputs: RoleReachability{Queries, Mutations int}, or an error.
// Constraints: sent with BOTH the admin secret (read via
// readHasuraAdminSecretFromContainer — the running container, never .env)
// AND the X-Hasura-Role header. This is Hasura's documented role-impersonation
// mechanism: the admin secret authenticates the request, and the role header
// tells Hasura which role's permissions to apply to it, so the response is
// exactly what that role — and only that role — can reach.
//
// Verified live 2026-08-31 that the ticket's originally-specified "role
// header with NO admin secret" does NOT work: without an admin secret (or a
// JWT), Hasura ignores X-Hasura-Role entirely and always falls back to
// HASURA_GRAPHQL_UNAUTHORIZED_ROLE (nSelf default: "public") regardless of
// the header's value — a real role name and a nonexistent one returned
// identical counts. Admin secret + role header, by contrast, produced 4
// queries/2 mutations for "user" vs. 33/77 for unrestricted admin access on
// the same instance — a real distinction, matching the shape of the
// 2026-08-21 Unity report's "3250 mutations vs 283 for pia_player" finding
// (which used exactly this combination, not a bare role header).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/httptimeout"
)

const roleIntrospectionQuery = `query IntrospectionQuery {
  __schema {
    queryType { fields { name } }
    mutationType { fields { name } }
  }
}`

// RoleReachability is the count of top-level query/mutation fields visible
// to a role, from a role-scoped introspection query.
type RoleReachability struct {
	Role      string
	Queries   int
	Mutations int
}

func hasuraGraphQLURL(cfg *config.Config) string {
	port := cfg.Hasura.Port
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf("http://localhost:%d/v1/graphql", port)
}

// VerifyRoleReachability runs an introspection query against the live
// GraphQL endpoint impersonating the given role (admin secret + X-Hasura-Role
// — see package doc for why a bare role header does not work).
func VerifyRoleReachability(ctx context.Context, cfg *config.Config, role string) (RoleReachability, error) {
	secret, err := readHasuraAdminSecretFromContainer(ctx, cfg)
	if err != nil {
		return RoleReachability{}, err
	}

	body, err := json.Marshal(map[string]string{"query": roleIntrospectionQuery})
	if err != nil {
		return RoleReachability{}, fmt.Errorf("marshal introspection query: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hasuraGraphQLURL(cfg), bytes.NewReader(body))
	if err != nil {
		return RoleReachability{}, fmt.Errorf("create introspection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", secret)
	req.Header.Set("X-Hasura-Role", role)

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return RoleReachability{}, fmt.Errorf("introspection request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var respBody bytes.Buffer
	if _, err := respBody.ReadFrom(resp.Body); err != nil {
		return RoleReachability{}, fmt.Errorf("read introspection response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return RoleReachability{}, fmt.Errorf("hasura graphql API returned %d: %s", resp.StatusCode, respBody.String())
	}

	var parsed struct {
		Data struct {
			Schema struct {
				QueryType struct {
					Fields []struct {
						Name string `json:"name"`
					} `json:"fields"`
				} `json:"queryType"`
				MutationType *struct {
					Fields []struct {
						Name string `json:"name"`
					} `json:"fields"`
				} `json:"mutationType"`
			} `json:"__schema"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(respBody.Bytes(), &parsed); err != nil {
		return RoleReachability{}, fmt.Errorf("parse introspection response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return RoleReachability{}, fmt.Errorf("introspection query failed for role %q: %s", role, parsed.Errors[0].Message)
	}

	result := RoleReachability{Role: role, Queries: len(parsed.Data.Schema.QueryType.Fields)}
	if parsed.Data.Schema.MutationType != nil {
		result.Mutations = len(parsed.Data.Schema.MutationType.Fields)
	}
	return result, nil
}
