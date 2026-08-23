package commands

// Purpose: the "nself ai pool status" subcommand and its RunE. Inputs are
// the cobra command/args; outputs are printed pool status or an error.
// Constraints: split out of ai_pool.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	poolStatusJSON    bool
	poolStatusVerbose bool
)

var aiPoolStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pool status (keys, usage, capacity)",
	RunE:  runPoolStatus,
}

func runPoolStatus(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	body, status, err := aiPluginRequest(ctx, "GET", "/ai/pool/status", nil)
	if err != nil {
		return fmt.Errorf("pool status: %w", err)
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}

	if poolStatusJSON {
		fmt.Println(string(body))
		return nil
	}

	var ps struct {
		TotalKeys     int `json:"total_keys"`
		ActiveKeys    int `json:"active_keys"`
		ExhaustedKeys int `json:"exhausted_keys"`
		RateLimited   int `json:"rate_limited"`
		RevokedKeys   int `json:"revoked_keys"`
		Quarantined   int `json:"quarantined"`
		DailyCapacity int `json:"daily_capacity"`
		DailyUsed     int `json:"daily_used"`
		Keys          []struct {
			KeyIndex        int    `json:"key_index"`
			GoogleAccount   string `json:"google_account"`
			Fingerprint     string `json:"fingerprint"`
			DailyLimit      int    `json:"daily_limit"`
			CurrentUsage    int    `json:"current_usage"`
			Status          string `json:"status"`
			AutoProvisioned bool   `json:"auto_provisioned"`
		} `json:"keys"`
	}
	json.Unmarshal(body, &ps)

	fmt.Printf("Gemini Pool Status\n")
	fmt.Printf("  Total:     %d keys\n", ps.TotalKeys)
	fmt.Printf("  Active:    %d\n", ps.ActiveKeys)
	fmt.Printf("  Exhausted: %d\n", ps.ExhaustedKeys)
	fmt.Printf("  Rate-limited: %d\n", ps.RateLimited)
	fmt.Printf("  Revoked:   %d\n", ps.RevokedKeys)
	fmt.Printf("  Quarantine: %d\n", ps.Quarantined)
	fmt.Printf("  Capacity:  %d/%d RPD used\n", ps.DailyUsed, ps.DailyCapacity)
	fmt.Println()

	if poolStatusVerbose && len(ps.Keys) > 0 {
		fmt.Printf("%-5s %-12s %-25s %-6s %-8s %-10s %-5s\n",
			"IDX", "FINGERPRINT", "ACCOUNT", "LIMIT", "USED", "STATUS", "AUTO")
		for _, k := range ps.Keys {
			auto := "no"
			if k.AutoProvisioned {
				auto = "yes"
			}
			acct := k.GoogleAccount
			if len(acct) > 24 {
				acct = acct[:21] + "..."
			}
			fmt.Printf("%-5d %-12s %-25s %-6d %-8d %-10s %-5s\n",
				k.KeyIndex, k.Fingerprint, acct, k.DailyLimit, k.CurrentUsage, k.Status, auto)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// `nself ai pool provision`
// -----------------------------------------------------------------------------
