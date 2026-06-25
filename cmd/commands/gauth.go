package commands

// Purpose: nself gauth command group — status, refresh, revoke.
//   Wires to plugin-gauth (default port 3762) for headless Google OAuth token management.
//   Operator-facing commands for provisioning the headless OAuth service.
//   No OAuth logic here; the CLI delegates to plugin-gauth HTTP endpoints.
// Inputs: account_id flag, optional --json flag for status, --force flag for refresh.
// Outputs: Formatted tables (status) or confirmation lines; never displays token values.
// Constraints:
//   - Revoke never prints refresh tokens in error messages.
//   - --json output does not include token values, only metadata.
// SPORT: F02 — 3 new nself gauth commands.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// gauthBaseURL returns the plugin-gauth base URL from env or default.
func gauthBaseURL() string {
	if u := os.Getenv("GAUTH_URL"); u != "" {
		return u
	}
	port := os.Getenv("GAUTH_PORT")
	if port == "" {
		port = "3762"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// gauthClient returns a short-timeout HTTP client for gauth calls.
func gauthClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

// --- gauth root command ---

var gauthCmd = &cobra.Command{
	Use:   "gauth",
	Short: "Manage Google OAuth tokens for nSelf AI services",
	Long: `Manage headless Google OAuth tokens used by plugin-gauth (:3762).

Subcommands:
  status   Show token expiry for all provisioned accounts
  refresh  Force-refresh a specific account's access token
  revoke   Revoke and remove a stored refresh token`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// --- gauth status ---

var gauthStatusJSON bool
var gauthStatusAccount string

var gauthStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show token expiry for all provisioned gauth accounts",
	Long: `List all provisioned Google OAuth accounts with their token expiry and cache state.
Use --account to filter to a single account. Use --json for machine-readable output.`,
	RunE: runGauthStatus,
}

func runGauthStatus(cmd *cobra.Command, args []string) error {
	url := gauthBaseURL() + "/status"
	if gauthStatusAccount != "" {
		url += "?account_id=" + gauthStatusAccount
	}

	resp, err := gauthClient().Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("gauth status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gauth status: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gauth status: server returned %d: %s", resp.StatusCode, body)
	}

	if gauthStatusJSON {
		fmt.Println(string(body))
		return nil
	}

	var envelope struct {
		Accounts []struct {
			AccountID   string  `json:"account_id"`
			Status      string  `json:"status"`
			ExpiresHint *string `json:"expires_hint"`
			Cached      bool    `json:"cached"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("gauth status: parse response: %w", err)
	}

	if len(envelope.Accounts) == 0 {
		fmt.Println("No gauth accounts provisioned.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ACCOUNT\tSTATUS\tEXPIRES\tCACHED")
	for _, a := range envelope.Accounts {
		exp := "unknown"
		if a.ExpiresHint != nil {
			exp = *a.ExpiresHint
		}
		cached := "no"
		if a.Cached {
			cached = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.AccountID, a.Status, exp, cached)
	}
	return w.Flush()
}

// --- gauth refresh ---

var gauthRefreshAccount string
var gauthRefreshForce bool

var gauthRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Force-refresh a Google OAuth access token for an account",
	Long: `Refresh the access token for a specific Google account via plugin-gauth.
By default uses the cached token if still valid. Use --force to bypass the cache.`,
	RunE: runGauthRefresh,
}

func runGauthRefresh(cmd *cobra.Command, args []string) error {
	if gauthRefreshAccount == "" {
		return fmt.Errorf("--account is required")
	}

	url := gauthBaseURL() + "/refresh?account_id=" + gauthRefreshAccount
	if gauthRefreshForce {
		url += "&force=true"
	}

	resp, err := gauthClient().Post(url, "application/json", http.NoBody) //nolint:noctx
	if err != nil {
		return fmt.Errorf("gauth refresh: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gauth refresh: read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var result struct {
			ExpiresAt string `json:"expires_at"`
			AccountID string `json:"account_id"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("gauth refresh: parse response: %w", err)
		}
		fmt.Printf("Refreshed: account=%s expires_at=%s\n", result.AccountID, result.ExpiresAt)
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("gauth refresh: token revoked for account %s — re-provision via `nself secret set`", gauthRefreshAccount)
	case http.StatusNotFound:
		return fmt.Errorf("gauth refresh: account not found: %s", gauthRefreshAccount)
	default:
		return fmt.Errorf("gauth refresh: server returned %d", resp.StatusCode)
	}
}

// --- gauth revoke ---

var gauthRevokeAccount string

var gauthRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke and remove a stored Google OAuth refresh token",
	Long: `Mark a stored refresh token as revoked and clear it from the in-memory cache.
After revocation, the account must be re-provisioned to obtain new tokens.`,
	RunE: runGauthRevoke,
}

func runGauthRevoke(cmd *cobra.Command, args []string) error {
	if gauthRevokeAccount == "" {
		return fmt.Errorf("--account is required")
	}

	url := gauthBaseURL() + "/token?account_id=" + gauthRevokeAccount
	req, err := http.NewRequest(http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("gauth revoke: build request: %w", err)
	}

	resp, err := gauthClient().Do(req)
	if err != nil {
		return fmt.Errorf("gauth revoke: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gauth revoke: read response: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Revoked: account=%s\n", gauthRevokeAccount)
		return nil
	}
	return fmt.Errorf("gauth revoke: server returned %d: %s", resp.StatusCode, body)
}

func init() {
	// Status flags
	gauthStatusCmd.Flags().BoolVar(&gauthStatusJSON, "json", false, "Output as raw JSON")
	gauthStatusCmd.Flags().StringVar(&gauthStatusAccount, "account", "", "Filter to a single account ID")

	// Refresh flags
	gauthRefreshCmd.Flags().StringVar(&gauthRefreshAccount, "account", "", "Account ID to refresh (required)")
	gauthRefreshCmd.Flags().BoolVar(&gauthRefreshForce, "force", false, "Bypass cache and force a new token from Google")
	_ = gauthRefreshCmd.MarkFlagRequired("account")

	// Revoke flags
	gauthRevokeCmd.Flags().StringVar(&gauthRevokeAccount, "account", "", "Account ID to revoke (required)")
	_ = gauthRevokeCmd.MarkFlagRequired("account")

	gauthCmd.AddCommand(gauthStatusCmd)
	gauthCmd.AddCommand(gauthRefreshCmd)
	gauthCmd.AddCommand(gauthRevokeCmd)

	RootCmd.AddCommand(gauthCmd)
}
