package commands

// Purpose: the "nself ai pool provision" and "nself ai pool add" subcommands
// and their RunE. Inputs are the cobra command/args; outputs are a
// provisioned or added pool key, or an error.
// Constraints: split out of ai_pool.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var poolProvisionAccount string

var aiPoolProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Non-interactive provision using stored refresh token",
	RunE:  runPoolProvision,
}

func runPoolProvision(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if poolProvisionAccount == "" {
		return fmt.Errorf("--account is required")
	}

	payload, _ := json.Marshal(map[string]any{
		"google_account": poolProvisionAccount,
	})
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/pool/provision", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}

	var result struct {
		Email       string `json:"email"`
		ProjectID   string `json:"gcp_project_id"`
		Fingerprint string `json:"fingerprint"`
	}
	json.Unmarshal(body, &result)
	fmt.Printf("Provisioned key for %s\n", result.Email)
	fmt.Printf("  GCP Project: %s\n", result.ProjectID)
	fmt.Printf("  Fingerprint: %s\n", result.Fingerprint)
	return nil
}

// -----------------------------------------------------------------------------
// `nself ai pool add`
// -----------------------------------------------------------------------------

var poolAddAccount string

var aiPoolAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a Google account via OAuth (opens browser)",
	RunE:  runPoolAdd,
}

func runPoolAdd(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Check current pool size before starting OAuth to surface cap violations
	// before the user completes an OAuth flow that would be rejected anyway.
	statusBody, statusCode, err := aiPluginRequest(ctx, "GET", "/ai/pool/status", nil)
	if err == nil && statusCode < 400 {
		var ps struct {
			TotalKeys int `json:"total_keys"`
		}
		if jsonErr := json.Unmarshal(statusBody, &ps); jsonErr == nil {
			if ps.TotalKeys >= aiPoolMaxKeys {
				return fmt.Errorf("pool is at capacity (%d/%d keys); remove a key before adding another",
					ps.TotalKeys, aiPoolMaxKeys)
			}
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"account_hint": poolAddAccount,
	})
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/pool/oauth/start", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}

	var resp struct {
		AuthURL string `json:"auth_url"`
	}
	json.Unmarshal(body, &resp)

	if resp.AuthURL == "" {
		return fmt.Errorf("no auth_url returned")
	}

	fmt.Printf("Opening browser for OAuth...\n")
	openBrowser(resp.AuthURL)
	fmt.Println("After authorization, run: nself ai pool status")
	return nil
}

// -----------------------------------------------------------------------------
// `nself ai pool remove`
// -----------------------------------------------------------------------------
