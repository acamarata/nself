package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestRunGauthStatus_Success tests the status command with mock HTTP server.
func TestRunGauthStatus_Success(t *testing.T) {
	// Mock plugin-gauth server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" && r.Method == "GET" {
			json.NewEncoder(w).Encode(gauthStatusResponse{
				Accounts: []gauthAccountStatus{
					{
						AccountID: "account-1",
						ExpiresAt: time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
						Cached:    true,
					},
					{
						AccountID: "account-2",
						ExpiresAt: time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
						Cached:    false,
					},
				},
			})
		}
	}))
	defer mockServer.Close()

	// Create a command context
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Set up environment
	old := os.Getenv("NSELF_PLUGIN_GAUTH_URL")
	t.Setenv("NSELF_PLUGIN_GAUTH_URL", mockServer.URL)
	defer func() {
		if old != "" {
			os.Setenv("NSELF_PLUGIN_GAUTH_URL", old)
		} else {
			os.Unsetenv("NSELF_PLUGIN_GAUTH_URL")
		}
	}()

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Reset global flags for this test
	gauthStatusJSON = false
	gauthStatusAccount = ""
	defer func() { gauthStatusJSON = false; gauthStatusAccount = "" }()

	err := runGauthStatus(cmd, []string{})
	if err != nil {
		t.Fatalf("runGauthStatus failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "account-1") || !strings.Contains(output, "account-2") {
		t.Errorf("expected both accounts in output, got:\n%s", output)
	}
}

// TestRunGauthRefresh_Success tests the refresh command.
func TestRunGauthRefresh_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/refresh") && r.Method == "POST" {
			json.NewEncoder(w).Encode(gauthRefreshResponse{
				AccessToken: "ya29.new-token",
				ExpiresAt:   time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC),
			})
		}
	}))
	defer mockServer.Close()

	old := os.Getenv("NSELF_PLUGIN_GAUTH_URL")
	t.Setenv("NSELF_PLUGIN_GAUTH_URL", mockServer.URL)
	defer func() {
		if old != "" {
			os.Setenv("NSELF_PLUGIN_GAUTH_URL", old)
		}
	}()

	gauthRefreshAccount = "my-account"
	defer func() { gauthRefreshAccount = ""; gauthRefreshForce = false }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	err := runGauthRefresh(cmd, []string{})
	if err != nil {
		t.Fatalf("runGauthRefresh failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-account") || !strings.Contains(output, "2026-06-25") {
		t.Errorf("expected account and expiry in output, got:\n%s", output)
	}
}

// TestRunGauthRevoke_Success tests the revoke command.
func TestRunGauthRevoke_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/token") && r.Method == "DELETE" {
			json.NewEncoder(w).Encode(gauthRevokeResponse{Success: true})
		}
	}))
	defer mockServer.Close()

	old := os.Getenv("NSELF_PLUGIN_GAUTH_URL")
	t.Setenv("NSELF_PLUGIN_GAUTH_URL", mockServer.URL)
	defer func() {
		if old != "" {
			os.Setenv("NSELF_PLUGIN_GAUTH_URL", old)
		}
	}()

	gauthRevokeAccount = "my-account"
	defer func() { gauthRevokeAccount = "" }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	err := runGauthRevoke(cmd, []string{})
	if err != nil {
		t.Fatalf("runGauthRevoke failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "revoked") || !strings.Contains(output, "my-account") {
		t.Errorf("expected revoke confirmation in output, got:\n%s", output)
	}
}

// TestPluginGauthURL_Default verifies default URL is localhost:5009.
func TestPluginGauthURL_Default(t *testing.T) {
	old := os.Getenv("NSELF_PLUGIN_GAUTH_URL")
	os.Unsetenv("NSELF_PLUGIN_GAUTH_URL")
	defer func() {
		if old != "" {
			os.Setenv("NSELF_PLUGIN_GAUTH_URL", old)
		}
	}()

	url := pluginGauthURL()
	if url != "http://localhost:5009" {
		t.Errorf("expected default URL http://localhost:5009, got %s", url)
	}
}

// TestPluginGauthURL_EnvOverride verifies env var overrides default.
func TestPluginGauthURL_EnvOverride(t *testing.T) {
	old := os.Getenv("NSELF_PLUGIN_GAUTH_URL")
	t.Setenv("NSELF_PLUGIN_GAUTH_URL", "http://custom.local:8080")
	defer func() {
		if old != "" {
			os.Setenv("NSELF_PLUGIN_GAUTH_URL", old)
		} else {
			os.Unsetenv("NSELF_PLUGIN_GAUTH_URL")
		}
	}()

	url := pluginGauthURL()
	if url != "http://custom.local:8080" {
		t.Errorf("expected env URL http://custom.local:8080, got %s", url)
	}
}

// TestPluginGauthURL_TrimTrailingSlash verifies trailing slash is removed.
func TestPluginGauthURL_TrimTrailingSlash(t *testing.T) {
	old := os.Getenv("NSELF_PLUGIN_GAUTH_URL")
	t.Setenv("NSELF_PLUGIN_GAUTH_URL", "http://localhost:5009/")
	defer func() {
		if old != "" {
			os.Setenv("NSELF_PLUGIN_GAUTH_URL", old)
		} else {
			os.Unsetenv("NSELF_PLUGIN_GAUTH_URL")
		}
	}()

	url := pluginGauthURL()
	if strings.HasSuffix(url, "/") {
		t.Errorf("expected trailing slash to be trimmed, got %s", url)
	}
}
