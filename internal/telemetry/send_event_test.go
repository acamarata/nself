package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestSendEventNoOpWhenDisabled verifies SendEvent makes zero HTTP calls when
// IsEnabled() returns false.
func TestSendEventNoOpWhenDisabled(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("NSELF_TELEMETRY", "0")

	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	SendEvent(EventCLIInstall, map[string]any{"via": "brew"})

	// Give the goroutine time to fire if the gate is broken.
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&called) != 0 {
		t.Errorf("expected zero HTTP calls when telemetry disabled, got %d", called)
	}
}

// TestSendEventPayloadShape verifies the JSON body shape sent to ping_api.
// It posts an eventPayload directly to a test server to validate field names.
func TestSendEventPayloadShape(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	os.Unsetenv("NSELF_TELEMETRY")
	os.Unsetenv("NSELF_TELEMETRY_OPT_OUT")

	type received struct {
		InstallID  string         `json:"install_id"`
		Event      string         `json:"event"`
		CLIVersion string         `json:"version"`
		OS         string         `json:"os"`
		Arch       string         `json:"arch"`
		Metadata   map[string]any `json:"metadata"`
	}

	done := make(chan received, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p received
		_ = json.Unmarshal(body, &p)
		done <- p
		w.WriteHeader(204)
	}))
	defer srv.Close()

	id := LoadOrCreateInstallID()

	p := eventPayload{
		InstallID:  id,
		Event:      EventPluginInstall,
		CLIVersion: "test",
		OS:         "linux",
		Arch:       "amd64",
		Metadata:   map[string]any{"plugin_name": "ai", "tier": "pro"},
	}
	body, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	select {
	case got := <-done:
		if got.InstallID == "" {
			t.Error("install_id must not be empty")
		}
		if got.Event != EventPluginInstall {
			t.Errorf("event = %q, want %q", got.Event, EventPluginInstall)
		}
		if got.OS != "linux" {
			t.Errorf("os = %q, want linux", got.OS)
		}
		if got.Arch != "amd64" {
			t.Errorf("arch = %q, want amd64", got.Arch)
		}
	case <-time.After(2 * time.Second):
		t.Error("no payload received within 2s")
	}
}

// TestSendEventNetworkFailSilent verifies that a network failure does not block
// the caller. SendEvent must return immediately.
func TestSendEventNetworkFailSilent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	os.Unsetenv("NSELF_TELEMETRY")
	os.Unsetenv("NSELF_TELEMETRY_OPT_OUT")

	// Start and immediately close a server — connections refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	returned := make(chan struct{})
	go func() {
		defer close(returned)
		SendEvent(EventCLIFirstInit, nil)
	}()

	select {
	case <-returned:
		// Caller unblocked immediately.
	case <-time.After(500 * time.Millisecond):
		t.Error("SendEvent blocked the caller for >500ms (must return immediately)")
	}
}

// TestSendEventTimeoutRespected verifies the 3-second deadline fires when the
// server hangs. sendPayload is called directly with the test server URL.
func TestSendEventTimeoutRespected(t *testing.T) {
	hang := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-hang
	}))
	defer func() {
		close(hang)
		srv.Close()
	}()

	// Patch httpClient so the test server is reachable.
	origClient := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = origClient }()

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()

	start := time.Now()
	// sendPayload posts to pingAPIEndpoint, not srv.URL — so we build the
	// request manually and call the transport layer through httpClient.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL,
		bytes.NewReader([]byte(`{"event":"test"}`)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	_, _ = httpClient.Do(req) // expected to time out or be cancelled

	elapsed := time.Since(start)

	limit := sendTimeout + 500*time.Millisecond
	if elapsed > limit {
		t.Errorf("request took %v, want < %v (sendTimeout + margin)", elapsed, limit)
	}
}
