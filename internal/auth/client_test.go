// Package auth — unit tests for the HTTP auth client.
// Tests use httptest.NewServer to mock the auth server; no real network calls.
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// serverFunc is a convenience type for httptest handler functions.
type serverFunc func(w http.ResponseWriter, r *http.Request)

// newAuthServer starts a test HTTP server with the given handler and sets
// NSELF_AUTH_SERVER_URL so all client functions hit it.
// The returned cleanup function must be called when the test ends.
func newAuthServer(t *testing.T, h serverFunc) (cleanup func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(h))
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)
	return srv.Close
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ─── DeviceAuthorize ──────────────────────────────────────────────────────────

func TestDeviceAuthorize_Success(t *testing.T) {
	want := DeviceCodeResponse{
		DeviceCode:      "dev-code-123",
		UserCode:        "ABCD-EFGH",
		VerificationURL: "https://nself.org/auth/cli?code=ABCD-EFGH",
		ExpiresInSec:    300,
		IntervalSec:     5,
	}
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/device/authorize" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, want)
	})
	defer cleanup()

	got, err := DeviceAuthorize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DeviceCode != want.DeviceCode {
		t.Errorf("DeviceCode: got %q, want %q", got.DeviceCode, want.DeviceCode)
	}
	if got.UserCode != want.UserCode {
		t.Errorf("UserCode: got %q, want %q", got.UserCode, want.UserCode)
	}
}

func TestDeviceAuthorize_ServerError(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "service_unavailable",
			"message": "auth server offline",
		})
	})
	defer cleanup()

	_, err := DeviceAuthorize()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeviceAuthorize_MalformedJSON(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "{not valid json")
	})
	defer cleanup()

	_, err := DeviceAuthorize()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

// ─── PollToken ────────────────────────────────────────────────────────────────

func TestPollToken_Pending(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/device/token" {
			w.WriteHeader(http.StatusAccepted) // authorization_pending
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	tok, err := PollToken("test-device-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != nil {
		t.Errorf("expected nil token for pending state, got %+v", tok)
	}
}

func TestPollToken_Success(t *testing.T) {
	want := TokenResponse{
		AccessToken:  "jwt.access.token",
		SessionToken: "session-opaque-token",
		Email:        "user@example.com",
		Tier:         "basic",
		ExpiresAt:    "2027-01-01T00:00:00Z",
	}
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/device/token" {
			writeJSON(w, http.StatusOK, want)
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	tok, err := PollToken("test-device-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok == nil {
		t.Fatal("expected token, got nil")
	}
	if tok.Email != want.Email {
		t.Errorf("Email: got %q, want %q", tok.Email, want.Email)
	}
	if tok.Tier != want.Tier {
		t.Errorf("Tier: got %q, want %q", tok.Tier, want.Tier)
	}
}

func TestPollToken_Error(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "expired_token",
			"message": "device code expired",
		})
	})
	defer cleanup()

	_, err := PollToken("expired-code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPollToken_MalformedSuccess(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "{bad json}")
	})
	defer cleanup()

	_, err := PollToken("any")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

// ─── RefreshToken ─────────────────────────────────────────────────────────────

func TestRefreshToken_Success(t *testing.T) {
	want := TokenResponse{
		AccessToken: "new.jwt.token",
		Email:       "user@example.com",
		Tier:        "pro",
		ExpiresAt:   "2027-06-01T00:00:00Z",
	}
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/refresh" && r.Method == http.MethodPost {
			// Verify cookie is forwarded
			if r.Header.Get("Cookie") == "" {
				http.Error(w, "missing cookie", http.StatusBadRequest)
				return
			}
			writeJSON(w, http.StatusOK, want)
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	tok, err := RefreshToken("old.token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != want.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", tok.AccessToken, want.AccessToken)
	}
}

func TestRefreshToken_Unauthorized(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_token",
			"message": "session revoked",
		})
	})
	defer cleanup()

	_, err := RefreshToken("revoked.token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── RevokeSession ────────────────────────────────────────────────────────────

func TestRevokeSession_Success(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/signout" {
			writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	if err := RevokeSession("valid.token", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRevokeSession_AllSessions(t *testing.T) {
	var gotURL string
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.RequestURI()
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	defer cleanup()

	if err := RevokeSession("valid.token", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotURL != "/auth/signout?all=true" {
		t.Errorf("expected URL /auth/signout?all=true, got %q", gotURL)
	}
}

func TestRevokeSession_Error(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_token",
			"message": "not logged in",
		})
	})
	defer cleanup()

	if err := RevokeSession("bad.token", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── GetSession ───────────────────────────────────────────────────────────────

func TestGetSession_Success(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/session" {
			resp := AccountInfo{Authenticated: true}
			resp.Account.Email = "user@example.com"
			resp.Account.Tier = "basic"
			resp.Account.EmailVerified = true
			writeJSON(w, http.StatusOK, resp)
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	info, err := GetSession("valid.token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.Authenticated {
		t.Error("expected Authenticated=true")
	}
	if info.Account.Email != "user@example.com" {
		t.Errorf("Email: got %q", info.Account.Email)
	}
}

func TestGetSession_Unauthorized(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer cleanup()

	_, err := GetSession("expired.token")
	if err == nil {
		t.Fatal("expected ErrNotLoggedIn, got nil")
	}
	if err != ErrNotLoggedIn {
		t.Errorf("expected ErrNotLoggedIn, got %v", err)
	}
}

func TestGetSession_ServerError(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "database unreachable",
		})
	})
	defer cleanup()

	_, err := GetSession("any.token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── GetLicenses ──────────────────────────────────────────────────────────────

func TestGetLicenses_Success(t *testing.T) {
	licenses := []LicenseInfo{
		{
			ID:      "lic-001",
			Product: "nself",
			Tier:    "basic",
			Bundles: []string{"nChat", "nClaw"},
			IsActive: true,
		},
	}
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/licenses" {
			writeJSON(w, http.StatusOK, map[string]interface{}{"licenses": licenses})
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	got, err := GetLicenses("valid.token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 license, got %d", len(got))
	}
	if got[0].ID != "lic-001" {
		t.Errorf("ID: got %q", got[0].ID)
	}
}

func TestGetLicenses_Empty(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"licenses": []LicenseInfo{}})
	})
	defer cleanup()

	got, err := GetLicenses("valid.token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d", len(got))
	}
}

func TestGetLicenses_Error(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "token expired",
		})
	})
	defer cleanup()

	_, err := GetLicenses("expired.token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── AuthAPIError.Error ───────────────────────────────────────────────────────

func TestAuthAPIError_Error(t *testing.T) {
	e := &AuthAPIError{Code: "test_code", Message: "test message", Status: 422}
	got := e.Error()
	if got == "" {
		t.Error("expected non-empty error string")
	}
	// Verify key fields appear in the message.
	for _, want := range []string{"test_code", "test message", "422"} {
		if !containsString(got, want) {
			t.Errorf("Error() %q missing %q", got, want)
		}
	}
}

// ─── AuthServerURL / CLIAuthBaseURL (env override) ───────────────────────────

func TestAuthServerURL_Default(t *testing.T) {
	t.Setenv("NSELF_AUTH_SERVER_URL", "")
	got := AuthServerURL()
	if got != "https://api.nself.org" {
		t.Errorf("default AuthServerURL: got %q", got)
	}
}

func TestAuthServerURL_Override(t *testing.T) {
	t.Setenv("NSELF_AUTH_SERVER_URL", "http://localhost:9999")
	got := AuthServerURL()
	if got != "http://localhost:9999" {
		t.Errorf("override AuthServerURL: got %q", got)
	}
}

func TestCLIAuthBaseURL_Default(t *testing.T) {
	t.Setenv("NSELF_CLI_AUTH_URL", "")
	got := CLIAuthBaseURL()
	if got != "https://nself.org/auth/cli" {
		t.Errorf("default CLIAuthBaseURL: got %q", got)
	}
}

func TestCLIAuthBaseURL_Override(t *testing.T) {
	t.Setenv("NSELF_CLI_AUTH_URL", "http://localhost:3010/auth/cli")
	got := CLIAuthBaseURL()
	if got != "http://localhost:3010/auth/cli" {
		t.Errorf("override CLIAuthBaseURL: got %q", got)
	}
}

// ─── helpers (test-internal) ──────────────────────────────────────────────────

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
