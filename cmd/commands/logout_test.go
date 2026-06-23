package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/auth"
)

// ── Registration ─────────────────────────────────────────────────────────────

// TestLogoutCmd_Registered verifies logoutCmd is on the global RootCmd.
func TestLogoutCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Use == "logout" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("logoutCmd is not registered on RootCmd")
	}
}

// TestLogoutCmd_AllFlag verifies --all flag is registered.
func TestLogoutCmd_AllFlag(t *testing.T) {
	if logoutCmd.Flags().Lookup("all") == nil {
		t.Error("--all flag not registered on logoutCmd")
	}
}

// ── Exit codes ───────────────────────────────────────────────────────────────

// TestLogoutCmd_NotLoggedIn_ExitsZero verifies that logging out when not already
// logged in produces exit code 0 (no error returned from RunE).
//
// Acceptance criterion: "All 7 commands: exit codes 0/1/2, cross-platform CI green"
// Not-logged-in is an informational state, not an error.
func TestLogoutCmd_NotLoggedIn_ExitsZero(t *testing.T) {
	home := isolateHome(t)
	// Ensure no auth.json exists.
	os.Remove(filepath.Join(home, ".nself", "auth.json"))

	// No revoke server needed — will skip revocation when no token.
	err := logoutCmd.RunE(logoutCmd, []string{})
	if err != nil {
		t.Fatalf("logout when not logged in should return nil (exit 0), got: %v", err)
	}
}

// TestLogoutCmd_LoggedIn_ExitsZero verifies that a successful logout returns nil.
func TestLogoutCmd_LoggedIn_ExitsZero(t *testing.T) {
	home := isolateHome(t)

	writeTestAuthFile(t, home, &auth.AuthFile{
		AccessToken: "logout-test-token",
		Email:       "logout@example.com",
		Tier:        "free",
	})

	// Serve a 200 revoke endpoint.
	revokeHit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "revoke") || strings.Contains(r.URL.Path, "logout") {
			revokeHit = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)

	err := logoutCmd.RunE(logoutCmd, []string{})
	if err != nil {
		t.Fatalf("logout RunE: %v", err)
	}
	_ = revokeHit // best-effort; revoke may or may not be called depending on impl

	// Credentials file should be removed.
	_, statErr := os.Stat(filepath.Join(home, ".nself", "auth.json"))
	if !os.IsNotExist(statErr) {
		t.Error("auth.json should be deleted after successful logout")
	}
}

// TestLogoutCmd_ServerUnreachable_StillExitsZero verifies that if the revoke
// endpoint fails, logout still succeeds (local creds are always deleted).
//
// The implementation uses best-effort revocation — local cleanup always happens.
func TestLogoutCmd_ServerUnreachable_StillExitsZero(t *testing.T) {
	home := isolateHome(t)

	writeTestAuthFile(t, home, &auth.AuthFile{
		AccessToken: "token-for-unreachable-test",
		Email:       "test@example.com",
		Tier:        "free",
	})

	// Point at a closed server so revoke call fails.
	t.Setenv("NSELF_AUTH_SERVER_URL", "http://127.0.0.1:1") // port 1 always refuses

	err := logoutCmd.RunE(logoutCmd, []string{})
	if err != nil {
		t.Fatalf("logout with unreachable server should still exit 0 (local creds cleared): %v", err)
	}

	// Local credentials must be removed regardless of server state.
	_, statErr := os.Stat(filepath.Join(home, ".nself", "auth.json"))
	if !os.IsNotExist(statErr) {
		t.Error("auth.json should be deleted even when revoke server is unreachable")
	}
}

// ── Cross-platform guard ──────────────────────────────────────────────────────

// TestLogoutCmd_CrossPlatform_HomeIsolation verifies the command reads
// home directory through os.UserHomeDir (which respects HOME on Unix,
// USERPROFILE on Windows) — exercised via the isolateHome helper.
func TestLogoutCmd_CrossPlatform_HomeIsolation(t *testing.T) {
	home := isolateHome(t)

	// Write to an isolated HOME — ensures the command uses the env var not a
	// hardcoded path.
	nselfDir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(nselfDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	authPath := filepath.Join(nselfDir, "auth.json")
	af := &auth.AuthFile{AccessToken: "xplatform-tok", Email: "x@test.com", Tier: "free"}
	data, _ := json.MarshalIndent(af, "", "  ")
	if err := os.WriteFile(authPath, data, 0600); err != nil {
		t.Fatalf("writing auth.json: %v", err)
	}

	t.Setenv("NSELF_AUTH_SERVER_URL", "http://127.0.0.1:1") // best-effort revoke; will fail

	err := logoutCmd.RunE(logoutCmd, []string{})
	if err != nil {
		t.Fatalf("cross-platform logout: %v", err)
	}

	if _, statErr := os.Stat(authPath); !os.IsNotExist(statErr) {
		t.Error("cross-platform: auth.json should be removed from isolated HOME")
	}
}
