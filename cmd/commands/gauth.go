package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────────────

var gauthCmd = &cobra.Command{
	Use:   "gauth",
	Short: "Manage server-side Google OAuth tokens",
	Long: `Manage server-side Google OAuth tokens via the plugin-gauth service.

Subcommands:
  status   Show token expiry and cache status for all accounts
  refresh  Force-refresh a specific account's access token
  revoke   Revoke and remove a stored refresh token`,
	SilenceUsage: true,
}

func init() {
	gauthCmd.AddCommand(gauthStatusCmd)
	gauthCmd.AddCommand(gauthRefreshCmd)
	gauthCmd.AddCommand(gauthRevokeCmd)
	RootCmd.AddCommand(gauthCmd)
}

// ── gauth status ────────────────────────────────────────────────────────────

var gauthStatusCmd = &cobra.Command{
	Use:   "status [--json] [--account <id>]",
	Short: "Show token expiry and cache status for all accounts",
	Long: `Show token expiry and cache status for all provisioned Google OAuth accounts.

Without --account, lists all accounts. With --account, filters to one account.
Use --json for machine-readable output.

Example:
  nself gauth status
  nself gauth status --json
  nself gauth status --account my-service-account`,
	RunE: runGauthStatus,
}

var (
	gauthStatusJSON    bool
	gauthStatusAccount string
)

func init() {
	gauthStatusCmd.Flags().BoolVar(&gauthStatusJSON, "json", false, "Emit JSON output")
	gauthStatusCmd.Flags().StringVar(&gauthStatusAccount, "account", "", "Filter to one account ID")
}

type gauthStatusResponse struct {
	Accounts []gauthAccountStatus `json:"accounts"`
	Error    string               `json:"error,omitempty"`
}

type gauthAccountStatus struct {
	AccountID string    `json:"account_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Cached    bool      `json:"cached"`
}

func runGauthStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// GET /status
	req, err := http.NewRequestWithContext(ctx, "GET",
		pluginGauthURL()+"/status", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("plugin-gauth unavailable: %w (ensure 'nself plugin install plugin-gauth' and service is running)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("plugin-gauth error %d: %s", resp.StatusCode, string(b))
	}

	var statusResp gauthStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if statusResp.Error != "" {
		return fmt.Errorf("plugin-gauth error: %s", statusResp.Error)
	}

	// Filter by account if specified
	accounts := statusResp.Accounts
	if gauthStatusAccount != "" {
		var filtered []gauthAccountStatus
		for _, a := range accounts {
			if a.AccountID == gauthStatusAccount {
				filtered = append(filtered, a)
			}
		}
		accounts = filtered
		if len(accounts) == 0 {
			return fmt.Errorf("account %q not found", gauthStatusAccount)
		}
	}

	// Output
	if gauthStatusJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"accounts": accounts,
		})
	}

	// Table output
	if len(accounts) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No accounts provisioned")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%-30s  %-25s  %-10s\n", "ACCOUNT ID", "EXPIRES AT", "CACHED")
	fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 70))
	for _, a := range accounts {
		cached := "no"
		if a.Cached {
			cached = "yes"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-30s  %-25s  %-10s\n", a.AccountID, a.ExpiresAt.Format(time.RFC3339), cached)
	}

	return nil
}

// ── gauth refresh ───────────────────────────────────────────────────────────

var gauthRefreshCmd = &cobra.Command{
	Use:   "refresh --account <id> [--force]",
	Short: "Force-refresh a specific account's access token",
	Long: `Refresh the access token for a specific Google OAuth account.

By default, returns a cached token if still valid. Use --force to bypass the cache
and fetch a fresh token from Google immediately.

Example:
  nself gauth refresh --account my-service-account
  nself gauth refresh --account my-service-account --force`,
	RunE: runGauthRefresh,
}

var (
	gauthRefreshAccount string
	gauthRefreshForce   bool
)

func init() {
	gauthRefreshCmd.Flags().StringVar(&gauthRefreshAccount, "account", "", "Account ID to refresh (required)")
	gauthRefreshCmd.Flags().BoolVar(&gauthRefreshForce, "force", false, "Bypass cache and fetch fresh token from Google")
	gauthRefreshCmd.MarkFlagRequired("account")
}

type gauthRefreshResponse struct {
	AccessToken string    `json:"access_token,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
	Error       string    `json:"error,omitempty"`
}

func runGauthRefresh(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// POST /refresh?account_id=X[&force=true]
	url := pluginGauthURL() + "/refresh?account_id=" + gauthRefreshAccount
	if gauthRefreshForce {
		url += "&force=true"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("plugin-gauth unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("plugin-gauth error %d: %s", resp.StatusCode, string(b))
	}

	var refreshResp gauthRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if refreshResp.Error != "" {
		return fmt.Errorf("refresh failed for account %q: %s", gauthRefreshAccount, refreshResp.Error)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Account %q refreshed. Expires at: %s\n", gauthRefreshAccount, refreshResp.ExpiresAt.Format(time.RFC3339))
	return nil
}

// ── gauth revoke ────────────────────────────────────────────────────────────

var gauthRevokeCmd = &cobra.Command{
	Use:   "revoke --account <id>",
	Short: "Revoke and remove a stored refresh token",
	Long: `Revoke a Google OAuth refresh token for a specific account and remove it from storage.

This prevents the stored refresh token from being used again and clears it from
the encrypted token store.

Example:
  nself gauth revoke --account my-service-account`,
	RunE: runGauthRevoke,
}

var gauthRevokeAccount string

func init() {
	gauthRevokeCmd.Flags().StringVar(&gauthRevokeAccount, "account", "", "Account ID to revoke (required)")
	gauthRevokeCmd.MarkFlagRequired("account")
}

type gauthRevokeResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func runGauthRevoke(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// DELETE /token?account_id=X
	url := pluginGauthURL() + "/token?account_id=" + gauthRevokeAccount

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("plugin-gauth unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("plugin-gauth error %d: %s", resp.StatusCode, string(b))
	}

	var revokeResp gauthRevokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&revokeResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if revokeResp.Error != "" {
		return fmt.Errorf("revoke failed for account %q: %s", gauthRevokeAccount, revokeResp.Error)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Token revoked and removed for account %q\n", gauthRevokeAccount)
	return nil
}

// ── Helper ──────────────────────────────────────────────────────────────────

// pluginGauthURL returns the base URL for the plugin-gauth service.
// Defaults to http://localhost:5009 (standard nSelf plugin port).
// Can be overridden via NSELF_PLUGIN_GAUTH_URL env var.
func pluginGauthURL() string {
	if url := os.Getenv("NSELF_PLUGIN_GAUTH_URL"); url != "" {
		return strings.TrimSuffix(url, "/")
	}
	return "http://localhost:5009"
}
