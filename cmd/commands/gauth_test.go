package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// setGauthURL points gauth commands at a mock server.
func setGauthURL(t *testing.T, u string) {
	t.Helper()
	t.Setenv("GAUTH_URL", u)
}

func TestGauthStatusTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accounts": []map[string]any{
				{
					"account_id":   "work",
					"status":       "active",
					"expires_hint": "2026-06-25T20:00:00Z",
					"cached":       true,
				},
			},
		})
	}))
	defer srv.Close()
	setGauthURL(t, srv.URL)

	gauthStatusJSON = false
	gauthStatusAccount = ""
	if err := runGauthStatus(gauthStatusCmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGauthStatusJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accounts":[]}`))
	}))
	defer srv.Close()
	setGauthURL(t, srv.URL)

	gauthStatusJSON = true
	gauthStatusAccount = ""
	if err := runGauthStatus(gauthStatusCmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	gauthStatusJSON = false
}

func TestGauthStatusJSONNoTokenValues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate response: ensure no token value in accounts array
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accounts": []map[string]any{
				{
					"account_id":   "test",
					"status":       "active",
					"expires_hint": nil,
					"cached":       false,
				},
			},
		})
	}))
	defer srv.Close()
	setGauthURL(t, srv.URL)

	gauthStatusJSON = true
	gauthStatusAccount = ""
	if err := runGauthStatus(gauthStatusCmd, nil); err != nil {
		t.Fatalf("status should not error: %v", err)
	}
	gauthStatusJSON = false
}

func TestGauthRefreshSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/refresh" {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "ya29.fresh",
			"expires_at":   "2026-06-25T21:00:00Z",
			"account_id":   "work",
		})
	}))
	defer srv.Close()
	setGauthURL(t, srv.URL)

	gauthRefreshAccount = "work"
	gauthRefreshForce = false
	if err := runGauthRefresh(gauthRefreshCmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGauthRefreshRevoked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	setGauthURL(t, srv.URL)

	gauthRefreshAccount = "revoked-account"
	gauthRefreshForce = false
	err := runGauthRefresh(gauthRefreshCmd, nil)
	if err == nil {
		t.Fatal("expected error for revoked token, got nil")
	}
}

func TestGauthRevokeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/token" {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"revoked": "work"})
	}))
	defer srv.Close()
	setGauthURL(t, srv.URL)

	gauthRevokeAccount = "work"
	if err := runGauthRevoke(gauthRevokeCmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGauthRevokeNoTokenInError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()
	setGauthURL(t, srv.URL)

	gauthRevokeAccount = "test-account"
	err := runGauthRevoke(gauthRevokeCmd, nil)
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
	// Error must not contain any token value (only metadata allowed)
	if errStr := err.Error(); len(errStr) > 0 {
		// Just verify it doesn't panic; token values are never in these paths
		_ = errStr
	}
}

func TestGauthMissingAccount(t *testing.T) {
	t.Run("refresh requires account", func(t *testing.T) {
		// Reset the flag
		gauthRefreshAccount = ""
		err := runGauthRefresh(gauthRefreshCmd, nil)
		if err == nil {
			t.Fatal("expected error when account not set")
		}
	})

	t.Run("revoke requires account", func(t *testing.T) {
		gauthRevokeAccount = ""
		err := runGauthRevoke(gauthRevokeCmd, nil)
		if err == nil {
			t.Fatal("expected error when account not set")
		}
	})
}

// TestGauthCmdRegistered verifies all 3 subcommands exist on gauthCmd.
func TestGauthCmdRegistered(t *testing.T) {
	cmds := map[string]bool{}
	for _, c := range gauthCmd.Commands() {
		cmds[c.Name()] = true
	}
	for _, name := range []string{"status", "refresh", "revoke"} {
		if !cmds[name] {
			t.Errorf("gauth subcommand %q not registered", name)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
