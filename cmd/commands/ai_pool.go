// P88 Sprint 03: `nself ai pool` — 8 subcommands for Gemini pool management.
package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// aiPoolMaxKeys is the maximum number of Gemini API keys the pool accepts.
// Per F-PLUGIN:ai-pool-keys (ADR-006), this is 30 — raised from the original
// 10-key cap in P88 and the 20-key interim cap.  Any nself ai pool add call
// that would exceed this limit is rejected with a clear error before the OAuth
// flow is started.
const aiPoolMaxKeys = 30

// -----------------------------------------------------------------------------
// `nself ai pool` root
// -----------------------------------------------------------------------------

var aiPoolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Manage the Gemini API key pool (auto-provisioned + manual)",
	Long: `Manage the zero-config Gemini API key pool.

Subcommands:
  init         Interactive setup wizard (OAuth + auto-provision)
  status       Show pool status table
  provision    Non-interactive provision using stored refresh token
  add          Add a Google account via OAuth flow
  remove       Remove a key from the pool
  rotate       Rotate a key (new GCP key, revoke old)
  test         Test one or all keys with a 1-token request
  daily-reset  Manual midnight-style counter reset`,
	RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
}

// -----------------------------------------------------------------------------
// `nself ai pool init`
// -----------------------------------------------------------------------------

var aiPoolInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive wizard: add a Google account and auto-provision a Gemini key",
	RunE:  runPoolInit,
}

func runPoolInit(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	fmt.Println("Starting Gemini pool setup wizard...")
	fmt.Println("This will open your browser to authorize a Google account.")
	fmt.Println("A GCP project will be created and a free Gemini API key provisioned automatically.")
	fmt.Println()

	// Start OAuth flow
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/pool/oauth/start", []byte(`{}`))
	if err != nil {
		return fmt.Errorf("start OAuth: %w", err)
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}

	var resp struct {
		AuthURL string `json:"auth_url"`
		State   string `json:"state"`
	}
	json.Unmarshal(body, &resp)

	fmt.Printf("Opening browser: %s\n", resp.AuthURL)
	openBrowser(resp.AuthURL)
	fmt.Println()
	fmt.Println("After authorizing, you will be redirected back. Check pool status with:")
	fmt.Println("  nself ai pool status")

	return nil
}

// -----------------------------------------------------------------------------
// `nself ai pool status`
// -----------------------------------------------------------------------------

func init() {
	aiPoolStatusCmd.Flags().BoolVar(&poolStatusJSON, "json", false, "Emit JSON output")
	aiPoolStatusCmd.Flags().BoolVar(&poolStatusVerbose, "verbose", false, "Show per-key details")

	aiPoolProvisionCmd.Flags().StringVar(&poolProvisionAccount, "account", "", "Google account email")

	aiPoolAddCmd.Flags().StringVar(&poolAddAccount, "account", "", "Google account email hint")

	aiPoolRemoveCmd.Flags().StringVar(&poolRemoveAccount, "account", "", "Remove by Google account email")
	aiPoolRemoveCmd.Flags().StringVar(&poolRemoveKeyID, "key-id", "", "Remove by key index")

	aiPoolRotateCmd.Flags().StringVar(&poolRotateKeyID, "key-id", "", "Key index to rotate")

	aiPoolTestCmd.Flags().StringVar(&poolTestKeyID, "key-id", "", "Test a specific key")
	aiPoolTestCmd.Flags().BoolVar(&poolTestAll, "all", false, "Test all keys")

	aiPoolDailyResetCmd.Flags().BoolVar(&poolResetDryRun, "dry-run", false, "Show what would reset without resetting")

	aiPoolCmd.AddCommand(aiPoolInitCmd)
	aiPoolCmd.AddCommand(aiPoolStatusCmd)
	aiPoolCmd.AddCommand(aiPoolProvisionCmd)
	aiPoolCmd.AddCommand(aiPoolAddCmd)
	aiPoolCmd.AddCommand(aiPoolRemoveCmd)
	aiPoolCmd.AddCommand(aiPoolRotateCmd)
	aiPoolCmd.AddCommand(aiPoolTestCmd)
	aiPoolCmd.AddCommand(aiPoolDailyResetCmd)

	aiCmd.AddCommand(aiPoolCmd)
}
