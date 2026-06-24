// Package claw (session.go) provides client functions for the nself-ai-cc
// session relay service at AICCPort (3760).
//
// Purpose: Wire nself claw session subcommands to the nself-ai-cc HTTP API.
//   Handles session lifecycle: start, list, stop, and WebSocket attach.
// Inputs:  http.Client, provider string, session ID, nSelf JWT (for WS)
// Outputs: session ID on start; session list JSON; WS relay for attach
// Constraints:
//   - All requests target http://localhost:{AICCPort}; no env-var override.
//   - AttachSession uses ?token=<nself-jwt> (CLI-only auth per OD-E1-04).
//     TODO(P5): migrate to Authorization header once nself-ai-cc adds
//     WS Bearer auth middleware (tracking: P5-E1-WS-AUTH).
//   - No OAuth token extraction from PTY output (ToS boundary enforced
//     by nself-ai-cc server; CLI must not attempt to read provider tokens).
// SPORT: F02-COMMAND-INVENTORY.md (nself claw session *) · F10-PORT-REGISTRY.md
package claw

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ports"
)

// Session represents an active or completed PTY session record from nself-ai-cc.
type Session struct {
	ID         string    `json:"id"`
	Provider   string    `json:"provider"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

// StartSessionResponse is the response body from POST /sessions.
type StartSessionResponse struct {
	SessionID string `json:"session_id"`
}

// aiCCBaseURL returns the base URL for the nself-ai-cc service.
// Always http://localhost:AICCPort — no env-var override per spec §11.
func aiCCBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", ports.AICCPort)
}

// StartSession creates a new PTY session for the given provider by calling
// POST /sessions on nself-ai-cc. Returns the session ID on success.
func StartSession(ctx context.Context, client *http.Client, provider string) (string, error) {
	if provider == "" {
		return "", fmt.Errorf("provider is required\nHint: run `nself claw session start <provider>` (e.g. claude, codex)")
	}

	body := fmt.Sprintf(`{"provider":%q}`, provider)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		aiCCBaseURL()+"/sessions",
		strings.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("building session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("nself-ai-cc not running (port %d unreachable)\nHint: run `nself start` to launch the AI services", ports.AICCPort)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("session start failed (HTTP %d): %s\nHint: check `nself claw session list` for active sessions", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var out StartSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decoding session response: %w", err)
	}
	if out.SessionID == "" {
		return "", fmt.Errorf("server returned empty session_id\nHint: this may be a server-side bug; check `nself doctor`")
	}
	return out.SessionID, nil
}

// ListSessions calls GET /sessions on nself-ai-cc and returns the slice of sessions.
func ListSessions(ctx context.Context, client *http.Client) ([]Session, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		aiCCBaseURL()+"/sessions",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("building list request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nself-ai-cc not running (port %d unreachable)\nHint: run `nself start` to launch the AI services", ports.AICCPort)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list sessions failed (HTTP %d): %s\nHint: check `nself claw session list` for service health", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("decoding sessions response: %w", err)
	}
	return sessions, nil
}

// StopSession calls DELETE /sessions/:id on nself-ai-cc to terminate a session.
func StopSession(ctx context.Context, client *http.Client, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required\nHint: run `nself claw session list` to find active session IDs")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		aiCCBaseURL()+"/sessions/"+url.PathEscape(sessionID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("building stop request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("nself-ai-cc not running (port %d unreachable)\nHint: run `nself start` to launch the AI services", ports.AICCPort)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("stop session failed (HTTP %d): %s\nHint: session may already be stopped; check `nself claw session list`", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// AttachSessionURL builds the WebSocket URL for attaching to a session.
// The ?token= query parameter carries the nSelf JWT (CLI-only auth per OD-E1-04).
//
// TODO(P5): remove ?token= and switch to Authorization header once nself-ai-cc
// adds WS Bearer auth middleware (tracking: P5-E1-WS-AUTH).
func AttachSessionURL(sessionID, nselfJWT string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required\nHint: run `nself claw session list` to find active session IDs")
	}
	u := fmt.Sprintf("ws://localhost:%d/sessions/%s/stream",
		ports.AICCPort, url.PathEscape(sessionID))
	if nselfJWT != "" {
		u += "?token=" + url.QueryEscape(nselfJWT)
	}
	return u, nil
}
