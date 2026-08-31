// Package supabase implements the Supabase → nSelf migration pipeline.
//
// It connects to a Supabase project via PostgREST + the service_role key,
// pulls schema, data, auth users, and storage buckets, then generates
// nSelf-compatible Drizzle migration SQL files, Hasura metadata YAML, an
// auth user import script, and a human-readable next-steps summary.
package supabase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// Config holds the credentials for one Supabase project.
type Config struct {
	ProjectRef string // e.g. "abcdefghijklmnopqrst"
	ServiceKey string // service_role JWT
}

// Table is a simplified representation of a Supabase table.
type Table struct {
	Schema  string
	Name    string
	Columns []Column
}

// Column describes a single column returned by the Supabase REST schema API.
type Column struct {
	Name       string
	DataType   string
	IsNullable bool
	Default    string
}

// RLSPolicy is a PostgreSQL row-level security policy captured from
// the information_schema via PostgREST.
type RLSPolicy struct {
	TableSchema string
	TableName   string
	PolicyName  string
	Command     string // SELECT | INSERT | UPDATE | DELETE | ALL
	Permissive  bool
	Roles       []string
	Using       string
	WithCheck   string
}

// AuthUser represents a minimal view of a Supabase auth.users row.
type AuthUser struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	EmailConfirmed bool   `json:"email_confirmed_at"`
	CreatedAt      string `json:"created_at"`
}

// StorageBucket is a Supabase Storage bucket.
type StorageBucket struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Public        bool   `json:"public"`
	FileSizeLimit int    `json:"file_size_limit"`
}

// PullResult contains everything pulled from a Supabase project.
type PullResult struct {
	Tables   []Table
	Policies []RLSPolicy
	Users    []AuthUser
	Buckets  []StorageBucket
}

// GeneratedArtifacts contains the files the generator produced in memory
// (keyed by relative output path).
type GeneratedArtifacts struct {
	// MigrationSQL is the Drizzle-compatible SQL migration file content.
	MigrationSQL string
	// HasuraMetadata is the Hasura metadata YAML for the pulled schema.
	HasuraMetadata string
	// AuthImportScript is a shell script that imports users into nSelf.
	AuthImportScript string
	// Summary is a human-readable text summary with next steps.
	Summary string
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client talks to the Supabase REST APIs.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a Client for the given Supabase project.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// baseURL returns the PostgREST endpoint for the project.
func (c *Client) baseURL() string {
	return fmt.Sprintf("https://%s.supabase.co/rest/v1", c.cfg.ProjectRef)
}

// adminURL returns the Supabase Admin API base URL.
func (c *Client) adminURL() string {
	return fmt.Sprintf("https://%s.supabase.co", c.cfg.ProjectRef)
}

// get performs a GET with exponential backoff (max 30 s total, max 8 s interval).
func (c *Client) get(ctx context.Context, url string, extraHeaders map[string]string) ([]byte, error) {
	op := func() ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, backoff.Permanent(fmt.Errorf("building request: %w", err))
		}
		req.Header.Set("apikey", c.cfg.ServiceKey)
		req.Header.Set("Authorization", "Bearer "+c.cfg.ServiceKey)
		req.Header.Set("Accept", "application/json")
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("reading response body: %w", readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, backoff.Permanent(fmt.Errorf("authentication error (HTTP %d) -- check your service_role key", resp.StatusCode))
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, backoff.Permanent(fmt.Errorf("resource not found (HTTP 404) -- check your project ref"))
		}
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error (HTTP %d) -- will retry", resp.StatusCode)
		}
		if resp.StatusCode >= 400 {
			return nil, backoff.Permanent(fmt.Errorf("client error (HTTP %d): %s", resp.StatusCode, string(data)))
		}

		return data, nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 8 * time.Second

	body, err := backoff.Retry(ctx, op,
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	return body, nil
}
