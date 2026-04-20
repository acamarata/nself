package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func buildOAuthTestServer(t *testing.T, providers []string, failAccounts map[string]bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// List providers.
		if r.Method == http.MethodGet && r.URL.Path == "/api/oauth/providers" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(providers)
			return
		}

		// Per-provider refresh-now.
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/oauth/refresh-now") {
			parts := strings.Split(r.URL.Path, "/")
			// path: /api/plugins/{provider}/oauth/refresh-now
			var provider string
			for i, p := range parts {
				if p == "plugins" && i+1 < len(parts) {
					provider = parts[i+1]
					break
				}
			}

			if failAccounts[provider] {
				http.Error(w, "provider error", http.StatusInternalServerError)
				return
			}

			resp := []map[string]interface{}{
				{"account_id": "test-account", "success": true},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
}

func TestManualRefresh(t *testing.T) {
	srv := buildOAuthTestServer(t, []string{"google"}, nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	result, err := Refresh(context.Background(), RefreshOptions{
		Provider:   "google",
		BaseURL:    srv.URL,
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if result.FailCount != 0 {
		t.Errorf("expected 0 failures, got %d", result.FailCount)
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Error("expected OK in output")
	}
}

func TestManualRefreshAll(t *testing.T) {
	providers := []string{"google", "github"}
	srv := buildOAuthTestServer(t, providers, nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	result, err := Refresh(context.Background(), RefreshOptions{
		All:        true,
		BaseURL:    srv.URL,
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("expected success for --all, got: %v", err)
	}
	if result.FailCount != 0 {
		t.Errorf("expected 0 failures, got %d", result.FailCount)
	}
	// Both providers should have refreshed.
	outStr := stdout.String()
	if !strings.Contains(outStr, "google") || !strings.Contains(outStr, "github") {
		t.Error("expected both providers in output")
	}
}

func TestManualRefreshFail(t *testing.T) {
	srv := buildOAuthTestServer(t, []string{"google"}, map[string]bool{"google": true})
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	result, err := Refresh(context.Background(), RefreshOptions{
		Provider:   "google",
		BaseURL:    srv.URL,
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: srv.Client(),
	})
	if err == nil {
		t.Fatal("expected error for failing provider")
	}
	if result == nil || result.FailCount == 0 {
		t.Error("expected FailCount > 0")
	}
	if !strings.Contains(stderr.String(), "FAIL") {
		t.Error("expected FAIL in stderr")
	}
}

func TestManualRefreshNoProvider(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_, err := Refresh(context.Background(), RefreshOptions{
		BaseURL: "http://localhost",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err == nil {
		t.Fatal("expected error when no provider and --all not set")
	}
}
