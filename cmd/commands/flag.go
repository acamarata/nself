package commands

import (
	"log"

	"github.com/spf13/cobra"
)

// CLI-R09: `flag` (application feature flags, served by the feature-flags
// plugin) and `feature` (CLI build-time capability gates) were an ambiguous
// pair — two similarly named top-level commands for unrelated concepts.
// `feature` moved to `nself config features`; this one is now plural to match
// what it manages. `flag` stays as a cobra alias, so the old spelling keeps
// working and its deprecation entry fires through CalledAs (see CLI-R03).
//
// Subcommand RunE bodies now live in flag_query.go (list/get/history) and
// flag_mutate.go (set/enable/disable/kill/prune) — CLI-R12 Batch B
// mechanical file-size split.
var flagCmd = &cobra.Command{
	Use:     "flags",
	Aliases: []string{"flag"},
	Short:   "Manage application feature flags",
	Long: `Manage feature flags via the nself feature-flags plugin.

Feature flags let you toggle functionality, run canary rollouts, and kill-switch
bad code paths without a redeploy. All operations route through nginx (port 3305 is
never accessed directly).

Flag types:
  release      New feature rollout (percentage-based)
  ops          Operational toggle (rate limits, cache tuning, etc.)
  experiment   A/B test variant
  kill_switch  Emergency disable — never auto-enables

Rule types (for evaluation):
  percentage   Random bucketing by user ID hash (0-100)
  user_id      Exact UID allowlist
  group        Named segment membership
  attribute    Arbitrary context attribute match
  datetime     Time-window gate (starts_at / ends_at)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	// Shared plugin-url override (for tests / non-default deployments)
	for _, sub := range []*cobra.Command{
		flagListCmd, flagGetCmd, flagSetCmd, flagEnableCmd,
		flagDisableCmd, flagKillCmd, flagHistoryCmd, flagPruneCmd,
	} {
		sub.Flags().String("plugin-url", "", "Override feature-flags plugin URL (default: http://127.0.0.1:3305/v1)")
	}

	// list flags
	flagListCmd.Flags().String("type", "", "Filter by flag type: release, ops, experiment, kill_switch")
	flagListCmd.Flags().Bool("json", false, "Output as JSON")

	// get flags
	flagGetCmd.Flags().Bool("json", false, "Output as JSON")

	// set flags
	flagSetCmd.Flags().Bool("enabled", false, "Set enabled state (use --enabled or --no-enabled)")
	flagSetCmd.Flags().String("rollout-pct", "", "Set rollout percentage (0-100)")

	// kill flags
	flagKillCmd.Flags().String("reason", "", "Reason for kill-switch (required)")
	if err := flagKillCmd.MarkFlagRequired("reason"); err != nil {
		// Programming error: MarkFlagRequired only returns an error when the named
		// flag does not exist. Since "reason" is registered on the line above,
		// this fires only if this code is misedited. Bug-in-our-code guard.
		log.Fatalf("flag kill: mark required: %v — this is a code bug, not a config error", err)
	}

	// history flags
	flagHistoryCmd.Flags().Bool("json", false, "Output as JSON")

	// prune flags
	flagPruneCmd.Flags().Bool("stale", false, "List/delete flags past their stale_after_days threshold")
	flagPruneCmd.Flags().Bool("dry-run", false, "Preview stale flags without deleting")

	flagCmd.AddCommand(
		flagListCmd,
		flagGetCmd,
		flagSetCmd,
		flagEnableCmd,
		flagDisableCmd,
		flagKillCmd,
		flagHistoryCmd,
		flagPruneCmd,
	)
	RootCmd.AddCommand(flagCmd)
}
