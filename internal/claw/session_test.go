package claw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/ports"
)

// TestAttachSessionURL verifies that the WS URL uses AICCPort and the ?token= pattern.
func TestAttachSessionURL(t *testing.T) {
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	jwt := "test-jwt-token"

	wsURL, err := AttachSessionURL(sessionID, jwt)
	if err != nil {
		t.Fatalf("AttachSessionURL() unexpected error: %v", err)
	}

	wantPrefix := fmt.Sprintf("ws://localhost:%d/sessions/%s/stream", ports.AICCPort, sessionID)
	if !strings.HasPrefix(wsURL, wantPrefix) {
		t.Errorf("AttachSessionURL() = %q, want prefix %q", wsURL, wantPrefix)
	}
	if !strings.Contains(wsURL, "token=") {
		t.Errorf("AttachSessionURL() = %q, missing ?token= query param", wsURL)
	}
}

// TestAttachSessionURL_UsesPort3760 verifies the WS URL always targets AICCPort (3760).
func TestAttachSessionURL_UsesPort3760(t *testing.T) {
	wsURL, err := AttachSessionURL("session-abc", "jwt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(wsURL, ":3760") {
		t.Errorf("attach URL %q does not target port 3760 (AICCPort)", wsURL)
	}
}

// TestAttachSessionURL_EmptyID verifies that an empty session ID is rejected.
func TestAttachSessionURL_EmptyID(t *testing.T) {
	_, err := AttachSessionURL("", "jwt")
	if err == nil {
		t.Error("AttachSessionURL(\"\", ...) should return error for empty session ID")
	}
}

// TestStartSession_Success verifies that StartSession parses the session_id response.
func TestStartSession_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sessions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(StartSessionResponse{SessionID: "new-session-id"})
	}))
	defer srv.Close()

	// Override aiCCBaseURL by monkey-patching via test helper is not available
	// without injecting. Instead test the http client logic via the test server
	// by patching the URL via a custom RoundTripper.
	client := &http.Client{
		Transport: rewriteHostTransport{
			target: srv.URL,
			base:   http.DefaultTransport,
		},
	}

	id, err := StartSession(context.Background(), client, "claude")
	if err != nil {
		t.Fatalf("StartSession() unexpected error: %v", err)
	}
	if id != "new-session-id" {
		t.Errorf("StartSession() = %q, want %q", id, "new-session-id")
	}
}

// TestStartSession_MissingProvider verifies rejection on empty provider.
func TestStartSession_MissingProvider(t *testing.T) {
	_, err := StartSession(context.Background(), &http.Client{}, "")
	if err == nil {
		t.Error("StartSession with empty provider should return error")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error message %q should mention 'provider'", err.Error())
	}
}

// TestListSessions_Success verifies that ListSessions decodes the session list.
func TestListSessions_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	sessions := []Session{
		{ID: "sess-1", Provider: "claude", Status: "active", StartedAt: now},
		{ID: "sess-2", Provider: "codex", Status: "ended", StartedAt: now},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sessions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sessions)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: rewriteHostTransport{target: srv.URL, base: http.DefaultTransport},
	}

	got, err := ListSessions(context.Background(), client)
	if err != nil {
		t.Fatalf("ListSessions() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("ListSessions() returned %d sessions, want 2", len(got))
	}
	if got[0].ID != "sess-1" {
		t.Errorf("ListSessions()[0].ID = %q, want sess-1", got[0].ID)
	}
}

// TestStopSession_Success verifies that StopSession calls DELETE /sessions/:id.
func TestStopSession_Success(t *testing.T) {
	calledPath := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledPath = r.URL.Path
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := &http.Client{
		Transport: rewriteHostTransport{target: srv.URL, base: http.DefaultTransport},
	}

	err := StopSession(context.Background(), client, "session-xyz")
	if err != nil {
		t.Fatalf("StopSession() unexpected error: %v", err)
	}
	if calledPath != "/sessions/session-xyz" {
		t.Errorf("StopSession called path %q, want /sessions/session-xyz", calledPath)
	}
}

// TestStopSession_EmptyID verifies rejection on empty session ID.
func TestStopSession_EmptyID(t *testing.T) {
	err := StopSession(context.Background(), &http.Client{}, "")
	if err == nil {
		t.Error("StopSession with empty ID should return error")
	}
}

// rewriteHostTransport rewrites all requests to target the test server URL,
// allowing tests to use the real http.DefaultTransport against httptest servers
// without modifying the internal port constants.
type rewriteHostTransport struct {
	target string
	base   http.RoundTripper
}

func (r rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request and rewrite host to the test server.
	clone := req.Clone(req.Context())
	targetURL := r.target + clone.URL.Path
	if clone.URL.RawQuery != "" {
		targetURL += "?" + clone.URL.RawQuery
	}
	parsed, err := clone.URL.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("rewriting URL: %w", err)
	}
	clone.URL = parsed
	clone.Host = parsed.Host
	return r.base.RoundTrip(clone)
}
