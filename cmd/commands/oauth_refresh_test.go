package commands

// oauth_refresh_test.go — Unit tests for the oauth refresh command.
// P4-E5-W3-S06-T20 added CLI-plumbing registration checks (now retained
// below as TestOAuthCmd_Registration / TestOAuthRefreshCmd_HasRunE).
// P6-E11-W2-S3-T18 (security command test floor) replaces the shallow
// "err != nil, skip if it succeeds" MissingConfig test with real
// integration tests against an httptest server, and adds the property the
// ticket names explicitly: OAuth tokens must never appear in command
// output.
//
// internal/oauth/manual_refresh_test.go already unit-tests oauth.Refresh
// itself (listProviders/refreshProvider against a mock server). The gap
// this file closes is the cobra wrapper layer: does runOAuthRefresh
// actually forward --account-id/--all/--base-url to oauth.Refresh
// correctly, and does it propagate a provider-side failure as a non-nil
// error rather than swallowing it?
//
// Purpose: verify command registration, flag→oauth.Refresh wiring, error
// propagation, and token-output safety.
// Inputs:  cobra command execution against a local httptest.Server.
// Outputs: error when a mocked account fails; nil when all succeed.
// Constraints: no real OAuth provider is contacted; the mock server stands
// in for the nSelf API's /api/oauth/* endpoints.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestOAuthCmd_Registration verifies the oauth command and subcommands are registered.
func TestOAuthCmd_Registration(t *testing.T) {
	t.Parallel()

	if oauthCmd == nil {
		t.Fatal("oauthCmd is nil — command not registered")
	}
	if oauthCmd.Use != "oauth" {
		t.Errorf("oauthCmd.Use = %q, want %q", oauthCmd.Use, "oauth")
	}
	if oauthRefreshCmd == nil {
		t.Fatal("oauthRefreshCmd is nil — subcommand not registered")
	}
	if oauthRefreshCmd.Use == "" {
		t.Error("oauthRefreshCmd.Use is empty")
	}
}

// TestOAuthRefreshCmd_HasRunE verifies the RunE field is set (not Run), per
// cobra conventions enforced by cli/rules/go.md.
func TestOAuthRefreshCmd_HasRunE(t *testing.T) {
	t.Parallel()

	if oauthRefreshCmd.RunE == nil {
		t.Error("oauthRefreshCmd.RunE is nil — command must use RunE, not Run")
	}
	if oauthRefreshCmd.Run != nil {
		t.Error("oauthRefreshCmd.Run is set — cobra convention requires RunE only")
	}
}

// oauthMockServer builds an httptest server implementing the two endpoints
// oauth.Refresh calls, returning per-account success/failure per the
// `results` map (accountID -> success). tokenLeakCanary is a fake token
// value the server includes in its JSON response body (mimicking a
// misbehaving provider plugin) to prove the CLI never echoes it back out.
func oauthMockServer(t *testing.T, results map[string]bool, tokenLeakCanary string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/oauth/providers", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]string{"google"})
	})
	mux.HandleFunc("/api/plugins/google/oauth/refresh-now", func(w http.ResponseWriter, r *http.Request) {
		type acct struct {
			AccountID string `json:"account_id"`
			Success   bool   `json:"success"`
			Error     string `json:"error,omitempty"`
			// Token is never a real field on the wire format, but a
			// misbehaving plugin could add one — the assertion below proves
			// the CLI would not surface it even if it did.
			Token string `json:"token,omitempty"`
		}
		var out []acct
		for id, ok := range results {
			a := acct{AccountID: id, Success: ok, Token: tokenLeakCanary}
			if !ok {
				a.Error = "refresh_token expired and re-auth required"
			}
			out = append(out, a)
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	return httptest.NewServer(mux)
}

// runOAuthRefreshCapturingStdout runs runOAuthRefresh with a real background
// context (cmd.Context() is nil until cobra's Execute() sets it, which this
// direct-RunE-call style of test bypasses) and returns (error, stdout).
func runOAuthRefreshCapturingStdout(t *testing.T, args []string, baseURL string, all bool) (error, string) {
	t.Helper()
	oauthRefreshCmd.SetContext(context.Background())
	_ = oauthRefreshCmd.Flags().Set("base-url", baseURL)
	_ = oauthRefreshCmd.Flags().Set("all", boolStr(all))
	defer func() {
		_ = oauthRefreshCmd.Flags().Set("base-url", "")
		_ = oauthRefreshCmd.Flags().Set("all", "false")
		_ = oauthRefreshCmd.Flags().Set("account-id", "")
	}()

	// runOAuthRefresh writes success lines to os.Stdout and failure lines to
	// os.Stderr (not cmd.OutOrStdout/ErrOrStderr), so redirect both real
	// streams into the same pipe for the duration of the call. Not safe
	// alongside t.Parallel() siblings that also redirect these streams (none
	// in this file do).
	origOut, origErr := os.Stdout, os.Stderr
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	os.Stdout, os.Stderr = w, w
	err := runOAuthRefresh(oauthRefreshCmd, args)
	_ = w.Close()
	os.Stdout, os.Stderr = origOut, origErr

	var buf bytes.Buffer
	if _, copyErr := io.Copy(&buf, r); copyErr != nil {
		t.Fatalf("reading captured output: %v", copyErr)
	}
	return err, buf.String()
}

// TestOAuthRefresh_AllAccountsSucceed_NoError verifies the happy path wires
// --base-url and --all through to oauth.Refresh, actually reaching the mock
// server (proving the flags are not silently ignored).
func TestOAuthRefresh_AllAccountsSucceed_NoError(t *testing.T) {
	srv := oauthMockServer(t, map[string]bool{"user@example.com": true}, "FAKE-LEAKED-TOKEN-abc123")
	defer srv.Close()

	err, out := runOAuthRefreshCapturingStdout(t, nil, srv.URL, true)
	if err != nil {
		t.Fatalf("expected success when all accounts refresh cleanly, got: %v", err)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("stdout %q does not report an OK result", out)
	}

	// Security property named in oauth_refresh.go's own doc comment: "OAuth
	// tokens are never exposed in command output."
	if strings.Contains(out, "FAKE-LEAKED-TOKEN-abc123") {
		t.Fatal("command output leaked a token value from the provider response")
	}
}

// TestOAuthRefresh_AccountFails_ErrorPropagates verifies a provider-reported
// failure surfaces as a non-nil error from runOAuthRefresh — a command that
// swallowed this would report success (exit 0) after a real refresh
// failure, hiding a token that still needs manual re-auth.
func TestOAuthRefresh_AccountFails_ErrorPropagates(t *testing.T) {
	srv := oauthMockServer(t, map[string]bool{"broken@example.com": false}, "")
	defer srv.Close()

	err, out := runOAuthRefreshCapturingStdout(t, nil, srv.URL, true)
	if err == nil {
		t.Fatal("expected an error when a mocked account fails to refresh, got nil")
	}
	if !strings.Contains(out, "FAIL") {
		t.Errorf("stdout %q does not report the FAIL result", out)
	}
}

// TestOAuthRefresh_NoProviderNoAll_Errors verifies the command requires
// either a provider argument or --all — silently defaulting to "refresh
// nothing" would look like success while doing nothing.
func TestOAuthRefresh_NoProviderNoAll_Errors(t *testing.T) {
	oauthRefreshCmd.SetContext(context.Background())
	_ = oauthRefreshCmd.Flags().Set("base-url", "http://127.0.0.1:1") // unreachable, must not matter
	_ = oauthRefreshCmd.Flags().Set("all", "false")
	defer func() {
		_ = oauthRefreshCmd.Flags().Set("base-url", "")
	}()

	err := runOAuthRefresh(oauthRefreshCmd, nil)
	if err == nil {
		t.Fatal("expected error when neither a provider arg nor --all is given, got nil")
	}
}
