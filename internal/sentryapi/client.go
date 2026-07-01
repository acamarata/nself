// Package sentryapi is the typed REST client for the ɳSentry cloud API.
//
// Purpose: One client for both worlds — the hosted SaaS (api.sentry.nself.org)
//
//	and a self-hosted / local sentry bundle (NSELF_SENTRY_API_URL).
//
// Contract (v1, consumed by `nself sentry *` and the MCP sentry tools):
//
//	Auth:   Authorization: Bearer nsk_<key>   (API keys issued per tenant, W1)
//	GET    /v1/me                              → {"tenant_id","email","tier","quotas":{dim:{"used","limit"}}}
//	GET    /v1/monitors                        → {"monitors":[Monitor]}
//	POST   /v1/monitors                        → 201 {"monitor":Monitor}
//	DELETE /v1/monitors/{id}                   → 204
//	POST   /v1/monitors/{id}/pause|resume      → {"monitor":Monitor}
//	GET    /v1/incidents?status=<s>            → {"incidents":[Incident]}
//	POST   /v1/incidents/{id}/ack|resolve      → {"incident":Incident}
//	GET    /v1/status-pages                    → {"status_pages":[StatusPage]}
//	POST   /v1/status-pages                    → 201 {"status_page":StatusPage}
//	GET    /v1/alerts/channels                 → {"channels":[AlertChannel]}
//	POST   /v1/alerts/channels/{id}/test       → {"delivered":bool,"detail":string}
//	Errors: non-2xx {"error":{"code","message"}}; 401 → ErrUnauthorized,
//	        402/429 → ErrQuotaExceeded (upgrade hint).
//
// Constraints: no cobra imports here; pure HTTP + JSON so MCP and commands share it.
// SPORT: CLI-PKG-SENTRYAPI-001
package sentryapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultAPIURL is the hosted ɳSentry SaaS API endpoint.
const DefaultAPIURL = "https://api.sentry.nself.org"

// KeyPrefix is the required prefix for ɳSentry API keys.
const KeyPrefix = "nsk_"

// ErrUnauthorized is returned on HTTP 401 — the caller should suggest
// `nself sentry login`.
var ErrUnauthorized = errors.New("unauthorized: invalid or missing API key — run 'nself sentry login'")

// ErrQuotaExceeded is returned on HTTP 402/429 quota responses.
var ErrQuotaExceeded = errors.New("quota exceeded for your tier — upgrade at https://sentry.nself.org/billing")

// Client is a typed REST client for the ɳSentry API.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New returns a Client for baseURL/apiKey with a sane default timeout.
func New(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultAPIURL
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

// WhoAmI returns the authenticated account (tier + quota usage).
func (c *Client) WhoAmI(ctx context.Context) (*Account, error) {
	var acct Account
	if err := c.do(ctx, http.MethodGet, "/v1/me", nil, &acct); err != nil {
		return nil, err
	}
	return &acct, nil
}

// ListMonitors returns all monitors for the tenant.
func (c *Client) ListMonitors(ctx context.Context) ([]Monitor, error) {
	var env monitorsEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/monitors", nil, &env); err != nil {
		return nil, err
	}
	return env.Monitors, nil
}

// CreateMonitor creates a monitor and returns the created record.
func (c *Client) CreateMonitor(ctx context.Context, req CreateMonitorRequest) (*Monitor, error) {
	var env monitorEnvelope
	if err := c.do(ctx, http.MethodPost, "/v1/monitors", req, &env); err != nil {
		return nil, err
	}
	return &env.Monitor, nil
}

// DeleteMonitor removes a monitor by id.
func (c *Client) DeleteMonitor(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/monitors/"+url.PathEscape(id), nil, nil)
}

// PauseMonitor pauses (paused=true) or resumes (paused=false) a monitor.
func (c *Client) PauseMonitor(ctx context.Context, id string, pause bool) (*Monitor, error) {
	action := "pause"
	if !pause {
		action = "resume"
	}
	var env monitorEnvelope
	path := "/v1/monitors/" + url.PathEscape(id) + "/" + action
	if err := c.do(ctx, http.MethodPost, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Monitor, nil
}

// ListIncidents returns incidents, optionally filtered by status
// (open | acknowledged | resolved; empty = all).
func (c *Client) ListIncidents(ctx context.Context, status string) ([]Incident, error) {
	path := "/v1/incidents"
	if status != "" {
		path += "?status=" + url.QueryEscape(status)
	}
	var env incidentsEnvelope
	if err := c.do(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return env.Incidents, nil
}

// AckIncident acknowledges an open incident.
func (c *Client) AckIncident(ctx context.Context, id string) (*Incident, error) {
	return c.incidentAction(ctx, id, "ack")
}

// ResolveIncident resolves an incident.
func (c *Client) ResolveIncident(ctx context.Context, id string) (*Incident, error) {
	return c.incidentAction(ctx, id, "resolve")
}

// incidentAction POSTs /v1/incidents/{id}/{action} and returns the incident.
func (c *Client) incidentAction(ctx context.Context, id, action string) (*Incident, error) {
	var env incidentEnvelope
	path := "/v1/incidents/" + url.PathEscape(id) + "/" + action
	if err := c.do(ctx, http.MethodPost, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Incident, nil
}

// ListStatusPages returns the tenant's status pages.
func (c *Client) ListStatusPages(ctx context.Context) ([]StatusPage, error) {
	var env statusPagesEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/status-pages", nil, &env); err != nil {
		return nil, err
	}
	return env.StatusPages, nil
}

// CreateStatusPage creates a status page.
func (c *Client) CreateStatusPage(ctx context.Context, req CreateStatusPageRequest) (*StatusPage, error) {
	var env statusPageEnvelope
	if err := c.do(ctx, http.MethodPost, "/v1/status-pages", req, &env); err != nil {
		return nil, err
	}
	return &env.StatusPage, nil
}

// ListAlertChannels returns the tenant's alert channels.
func (c *Client) ListAlertChannels(ctx context.Context) ([]AlertChannel, error) {
	var env channelsEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/alerts/channels", nil, &env); err != nil {
		return nil, err
	}
	return env.Channels, nil
}

// TestAlertChannel sends a test notification through a channel.
func (c *Client) TestAlertChannel(ctx context.Context, id string) (*TestChannelResult, error) {
	var res TestChannelResult
	path := "/v1/alerts/channels/" + url.PathEscape(id) + "/test"
	if err := c.do(ctx, http.MethodPost, path, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// do executes one API request: marshal body → set auth header → decode into out.
//
// Purpose:  Single choke point for auth, error mapping, and JSON codec.
// Inputs:   ctx, method, path (starting with /), body (nil = no body), out (nil = discard).
// Outputs:  Typed errors: ErrUnauthorized (401), ErrQuotaExceeded (402/429),
//
//	otherwise a wrapped error carrying the server's error.message when present.
func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var rdr io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		rdr = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rdr)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("sentry api unreachable (%s): %w", c.BaseURL, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", ErrQuotaExceeded, serverMessage(respBody))
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return fmt.Errorf("sentry api HTTP %d: %s", resp.StatusCode, serverMessage(respBody))
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

// serverMessage extracts error.message from the wire envelope, falling back
// to the (truncated) raw body.
func serverMessage(body []byte) string {
	var e apiError
	if err := json.Unmarshal(body, &e); err == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	s := strings.TrimSpace(string(body))
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	if s == "" {
		s = "(empty body)"
	}
	return s
}
