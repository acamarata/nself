// Package mail provides an HTTP client for the ping_api /mail/* endpoints.
//
// The CLI's `nself mail` command wraps this client. ping_api in turn proxies
// to the mux + Postmark plugins to deliver transactional and broadcast email
// through the nSelf stack.
//
// Authentication uses a bearer license token (NSELF_LICENSE_KEY or the
// configured license keys via internal/license.CollectLicenseKeys). The
// Postmark plugin is part of the nClaw / ɳSelf+ paid bundles, so the server
// will reject calls when no valid license is presented.
//
// All methods are context-aware and return typed responses.
package mail

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

const (
	// DefaultPingURL is the production mail proxy.
	DefaultPingURL = "https://ping.nself.org"

	// defaultTimeout is the per-call HTTP timeout.
	defaultTimeout = 30 * time.Second

	// userAgent identifies the CLI to ping_api for log correlation.
	userAgent = "nself-cli/mail"
)

// Errors returned by the client. Callers can compare with errors.Is.
var (
	// ErrUnauthorized is returned when ping_api responds 401/403.
	// Indicates a missing, expired, or revoked license key.
	ErrUnauthorized = errors.New("unauthorized: nself mail requires nSelf+ or nClaw bundle (Postmark plugin)")
	// ErrUnreachable is returned when the network round-trip fails.
	ErrUnreachable = errors.New("ping.nself.org unreachable, check connectivity")
	// ErrServer is returned for any 5xx response.
	ErrServer = errors.New("ping_api server error")
)

// Client is a thin wrapper over http.Client with auth + base URL pinned.
type Client struct {
	BaseURL    string
	LicenseKey string
	HTTP       *http.Client
}

// New constructs a Client. baseURL falls back to DefaultPingURL when empty.
// licenseKey may be empty; in that case auth-required endpoints will fail
// with ErrUnauthorized.
func New(baseURL, licenseKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultPingURL
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		LicenseKey: licenseKey,
		HTTP:       &http.Client{Timeout: defaultTimeout},
	}
}

// SendRequest is the JSON body for POST /mail/send.
type SendRequest struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	BodyType string `json:"body_type,omitempty"` // "text" (default) or "html"
}

// SendResponse is the response body for POST /mail/send.
type SendResponse struct {
	MessageID string `json:"message_id"`
	Accepted  bool   `json:"accepted"`
	To        string `json:"to"`
}

// Send delivers a single transactional email through mux + Postmark.
func (c *Client) Send(ctx context.Context, req SendRequest) (*SendResponse, error) {
	var out SendResponse
	if err := c.do(ctx, http.MethodPost, "/mail/send", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BroadcastRequest is the JSON body for POST /mail/broadcast.
type BroadcastRequest struct {
	ListID     string `json:"list_id"`
	TemplateID string `json:"template_id"`
}

// BroadcastResponse is the response body for POST /mail/broadcast.
type BroadcastResponse struct {
	BatchID    string `json:"batch_id"`
	Recipients int    `json:"recipients"`
	Queued     bool   `json:"queued"`
}

// Broadcast queues a list-based broadcast email job.
func (c *Client) Broadcast(ctx context.Context, req BroadcastRequest) (*BroadcastResponse, error) {
	var out BroadcastResponse
	if err := c.do(ctx, http.MethodPost, "/mail/broadcast", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StatusResponse is the response body for GET /mail/status.
type StatusResponse struct {
	MessageID   string `json:"message_id"`
	Status      string `json:"status"` // delivered, bounced, spam_complaint, queued, sent
	To          string `json:"to,omitempty"`
	DeliveredAt string `json:"delivered_at,omitempty"`
	BouncedAt   string `json:"bounced_at,omitempty"`
	BounceType  string `json:"bounce_type,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

// Status queries the delivery status for a previously sent message.
func (c *Client) Status(ctx context.Context, messageID string) (*StatusResponse, error) {
	q := url.Values{"message_id": []string{messageID}}
	var out StatusResponse
	if err := c.do(ctx, http.MethodGet, "/mail/status?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Template describes a Postmark template registered with nSelf.
type Template struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Subject  string `json:"subject,omitempty"`
	Updated  string `json:"updated,omitempty"`
	Provider string `json:"provider,omitempty"` // typically "postmark"
}

// ListTemplatesResponse is the response body for GET /mail/templates.
type ListTemplatesResponse struct {
	Templates []Template `json:"templates"`
}

// ListTemplates fetches all Postmark templates registered with the nSelf stack.
func (c *Client) ListTemplates(ctx context.Context) ([]Template, error) {
	var out ListTemplatesResponse
	if err := c.do(ctx, http.MethodGet, "/mail/templates", nil, &out); err != nil {
		return nil, err
	}
	return out.Templates, nil
}

// DKIMVerifyResponse is the response body for GET /mail/dkim/verify.
type DKIMVerifyResponse struct {
	Domain   string `json:"domain"`
	Valid    bool   `json:"valid"`
	Selector string `json:"selector,omitempty"`
	Record   string `json:"record,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// VerifyDKIM checks for a valid DKIM record on the supplied domain.
func (c *Client) VerifyDKIM(ctx context.Context, domain string) (*DKIMVerifyResponse, error) {
	q := url.Values{"domain": []string{domain}}
	var out DKIMVerifyResponse
	if err := c.do(ctx, http.MethodGet, "/mail/dkim/verify?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// do performs the HTTP round-trip and decodes JSON. body may be nil for GET.
// On non-2xx responses, returns ErrUnauthorized / ErrServer / a wrapped error
// with the response body as context.
func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshalling request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.LicenseKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.LicenseKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrUnauthorized
	case resp.StatusCode >= 500:
		return fmt.Errorf("%w (HTTP %d): %s", ErrServer, resp.StatusCode, truncate(respBody, 256))
	case resp.StatusCode >= 400:
		return fmt.Errorf("ping_api %s %s: HTTP %d: %s", method, path, resp.StatusCode, truncate(respBody, 256))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
