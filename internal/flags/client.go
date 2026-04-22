// Package flags provides a thin REST client over the nself feature-flags
// plugin (port 3305, nginx-proxied). All requests go through the nginx route
// so no direct port access is ever used.
package flags

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "http://127.0.0.1:3305/v1"

// Client is a thin REST client for the feature-flags plugin.
type Client struct {
	base string
	http *http.Client
}

// NewClient returns a Client routed through the local nginx proxy (port 3305).
// Callers should rely on the default URL; override only in tests.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		base: baseURL,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// --- Response types ---

// Flag is a feature flag as returned by the plugin API.
type Flag struct {
	ID           string          `json:"id"`
	Key          string          `json:"key"`
	Name         *string         `json:"name"`
	Description  *string         `json:"description"`
	Type         string          `json:"type"`
	Enabled      bool            `json:"enabled"`
	RolloutPct   *int            `json:"rollout_pct"`
	DefaultValue json.RawMessage `json:"default_value"`
	Rules        json.RawMessage `json:"rules"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// AuditEntry is a row from np_feature_flags_audit.
type AuditEntry struct {
	ID      string          `json:"id"`
	FlagKey string          `json:"flag_key"`
	Actor   string          `json:"actor"`
	Action  string          `json:"action"`
	Before  json.RawMessage `json:"before"`
	After   json.RawMessage `json:"after"`
	Reason  *string         `json:"reason"`
	Ts      time.Time       `json:"ts"`
}

// SetFlagRequest is the body for the set/update operation.
type SetFlagRequest struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	RolloutPct *int    `json:"rollout_pct,omitempty"`
	Reason     *string `json:"reason,omitempty"`
}

// KillRequest is the body for the kill operation.
type KillRequest struct {
	Reason string `json:"reason"`
}

// --- HTTP helpers ---

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return fmt.Errorf("flags: build request: %w", err)
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	return c.sendJSON(ctx, http.MethodPost, path, body, out)
}

func (c *Client) put(ctx context.Context, path string, body interface{}, out interface{}) error {
	return c.sendJSON(ctx, http.MethodPut, path, body, out)
}

func (c *Client) sendJSON(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("flags: marshal body: %w", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return fmt.Errorf("flags: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out interface{}) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("flags: plugin unreachable (is feature-flags plugin running?): %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("flags: read body: %w", err)
	}

	if resp.StatusCode == http.StatusConflict {
		// Dependency guard — return the body as the error message.
		return fmt.Errorf("flags: %s", string(raw))
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("flags: plugin returned %d: %s", resp.StatusCode, string(raw))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("flags: decode response: %w", err)
	}
	return nil
}

// --- API methods ---

// List returns all feature flags, optionally filtered by type.
func (c *Client) List(ctx context.Context, flagType string) ([]Flag, error) {
	path := "/flags"
	if flagType != "" {
		path += "?type=" + flagType
	}
	var resp struct {
		Flags []Flag `json:"flags"`
		Count int    `json:"count"`
	}
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Flags, nil
}

// Get returns a single flag by key.
func (c *Client) Get(ctx context.Context, key string) (*Flag, error) {
	var f Flag
	if err := c.get(ctx, "/flags/"+key, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Set updates a flag's enabled state and/or rollout percentage.
func (c *Client) Set(ctx context.Context, key string, req SetFlagRequest) (*Flag, error) {
	var f Flag
	if err := c.put(ctx, "/flags/"+key, req, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Enable sets a flag to enabled=true.
func (c *Client) Enable(ctx context.Context, key string) (*Flag, error) {
	t := true
	return c.Set(ctx, key, SetFlagRequest{Enabled: &t})
}

// Disable sets a flag to enabled=false and broadcasts pubsub invalidation.
func (c *Client) Disable(ctx context.Context, key string) (*Flag, error) {
	f := false
	return c.Set(ctx, key, SetFlagRequest{Enabled: &f})
}

// Kill performs an emergency kill-switch on a flag. reason is required.
func (c *Client) Kill(ctx context.Context, key, reason string) (*Flag, error) {
	var f Flag
	if err := c.post(ctx, "/flags/"+key+"/kill", KillRequest{Reason: reason}, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// History returns the audit log for a single flag.
func (c *Client) History(ctx context.Context, key string) ([]AuditEntry, error) {
	var resp struct {
		Entries []AuditEntry `json:"entries"`
	}
	if err := c.get(ctx, "/flags/"+key+"/history", &resp); err != nil {
		return nil, err
	}
	return resp.Entries, nil
}

// Prune lists flags that have exceeded their stale_after_days threshold.
// When dryRun is true, no flags are deleted.
func (c *Client) Prune(ctx context.Context, dryRun bool) ([]Flag, error) {
	path := "/flags/prune"
	if dryRun {
		path += "?dry_run=true"
	}
	var resp struct {
		Stale []Flag `json:"stale"`
	}
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Stale, nil
}
