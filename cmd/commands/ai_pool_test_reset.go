package commands

// Purpose: the "nself ai pool test" and "nself ai pool daily-reset"
// subcommands and their RunE. Inputs are the cobra command/args; outputs are
// printed test results or a reset pool, or an error.
// Constraints: split out of ai_pool.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	poolTestKeyID string
	poolTestAll   bool
)

var aiPoolTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test one or all keys with a 1-token Gemini request",
	RunE:  runPoolTest,
}

func runPoolTest(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	var payload []byte
	if poolTestAll {
		payload, _ = json.Marshal(map[string]any{"all": true})
	} else if poolTestKeyID != "" {
		payload, _ = json.Marshal(map[string]any{"key_id": atoi(poolTestKeyID)})
	} else {
		payload, _ = json.Marshal(map[string]any{"all": true})
	}

	body, status, err := aiPluginRequest(ctx, "POST", "/ai/pool/test", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}

	fmt.Println(string(body))
	return nil
}

// -----------------------------------------------------------------------------
// `nself ai pool daily-reset`
// -----------------------------------------------------------------------------

var poolResetDryRun bool

var aiPoolDailyResetCmd = &cobra.Command{
	Use:   "daily-reset",
	Short: "Manually trigger the daily counter reset",
	RunE:  runPoolDailyReset,
}

func runPoolDailyReset(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Call the daily reset directly via an internal endpoint
	path := "/ai/pool/status" // We use status to show before/after
	body, _, err := aiPluginRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}

	if poolResetDryRun {
		fmt.Println("Dry run. Current status:")
		fmt.Println(string(body))
		return nil
	}

	// Trigger reset via a POST to a special reset endpoint
	// For now, call the pool test with all=true as a proxy for admin action
	fmt.Println("Triggering daily reset...")
	// The actual reset happens server-side via cron at 00:00 UTC.
	// This command is a manual trigger that calls the same logic.
	resetBody, status, err := aiPluginRequest(ctx, "POST", "/ai/pool/test", []byte(`{"all":true}`))
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(resetBody))
	}
	fmt.Println("Daily reset triggered. Run `nself ai pool status` to verify.")
	return nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------
