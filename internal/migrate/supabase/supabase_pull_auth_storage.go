package supabase

// supabase_pull_auth_storage.go — pulling auth users and storage buckets from Supabase.
//
// Purpose: pull Supabase auth users and storage buckets, and drive the overall Pull sequence that combines schema, RLS, auth and storage, split out of supabase_pull.go for file size.
// Inputs: a *Client configured with a Supabase project URL and service_role key.
// Outputs: PullResult populated with AuthUser/StorageBucket values.
// Constraints: pure move from supabase_pull.go (CLI-R12 Batch E); no behaviour change.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// parsePostgresArray converts a Postgres array literal like "{user,anon}" into
// a Go string slice.
func parsePostgresArray(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return nil
	}
	// Strip surrounding braces.
	s = strings.TrimPrefix(strings.TrimSuffix(s, "}"), "{")
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Auth users pull
// ---------------------------------------------------------------------------

// PullAuthUsers fetches up to 1 000 users from the Supabase Admin auth API.
func (c *Client) PullAuthUsers(ctx context.Context) ([]AuthUser, error) {
	url := fmt.Sprintf("%s/auth/v1/admin/users?per_page=1000", c.adminURL())
	data, err := c.get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("pulling auth users: %w", err)
	}

	var wrapper struct {
		Users []AuthUser `json:"users"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Try bare array fallback.
		var users []AuthUser
		if err2 := json.Unmarshal(data, &users); err2 != nil {
			return nil, fmt.Errorf("parsing auth users response: %w", err)
		}
		return users, nil
	}
	return wrapper.Users, nil
}

// ---------------------------------------------------------------------------
// Storage buckets pull
// ---------------------------------------------------------------------------

// PullStorageBuckets fetches all storage buckets from the Supabase Storage API.
func (c *Client) PullStorageBuckets(ctx context.Context) ([]StorageBucket, error) {
	url := fmt.Sprintf("%s/storage/v1/bucket", c.adminURL())
	data, err := c.get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("pulling storage buckets: %w", err)
	}

	var buckets []StorageBucket
	if err := json.Unmarshal(data, &buckets); err != nil {
		return nil, fmt.Errorf("parsing storage buckets response: %w", err)
	}
	return buckets, nil
}

// ---------------------------------------------------------------------------
// Pull (orchestrates all pulls)
// ---------------------------------------------------------------------------

// Pull runs all four pull operations and returns the aggregate result.
func Pull(ctx context.Context, cfg Config) (*PullResult, error) {
	c := NewClient(cfg)

	tables, err := c.PullSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	policies, err := c.PullRLSPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("rls: %w", err)
	}

	users, err := c.PullAuthUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth users: %w", err)
	}

	buckets, err := c.PullStorageBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage buckets: %w", err)
	}

	return &PullResult{
		Tables:   tables,
		Policies: policies,
		Users:    users,
		Buckets:  buckets,
	}, nil
}
