package commands

// Purpose: State-changing feature-flag commands split out of flag.go
// (CLI-R12 Batch B mechanical file-size split). Holds `nself flag
// set/enable/disable/kill/prune` — every subcommand that writes to the
// feature-flags plugin.
// Inputs: cobra command flags (--enabled, --rollout-pct, --reason,
// --stale, --dry-run, --plugin-url) and the positional flag key.
// Outputs: stdout confirmation messages or a stale-flags table; errors
// wrap internal/flags client failures.
// Constraints: pure move, no behavior change. flagCmd (parent) and the
// shared --plugin-url flag registration remain in flag.go's init().

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/flags"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var flagSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Update a feature flag's enabled state and/or rollout percentage",
	Long: `Update a feature flag. At least one of --enabled or --rollout-pct is required.

Examples:
  nself flags set ai.safety.jailbreak_filter --enabled
  nself flags set ai.safety.jailbreak_filter --rollout-pct 25
  nself flags set ai.safety.jailbreak_filter --enabled --rollout-pct 50`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		enabledFlag := cmd.Flags().Lookup("enabled")
		rolloutStr, _ := cmd.Flags().GetString("rollout-pct")
		baseURL, _ := cmd.Flags().GetString("plugin-url")

		req := flags.SetFlagRequest{}

		if enabledFlag.Changed {
			v, _ := cmd.Flags().GetBool("enabled")
			req.Enabled = &v
		}
		if rolloutStr != "" {
			n, err := strconv.Atoi(rolloutStr)
			if err != nil || n < 0 || n > 100 {
				return fmt.Errorf("flag set: --rollout-pct must be an integer 0-100")
			}
			req.RolloutPct = &n
		}

		if req.Enabled == nil && req.RolloutPct == nil {
			return fmt.Errorf("flag set: provide at least --enabled or --rollout-pct")
		}

		c := flags.NewClient(baseURL)
		f, err := c.Set(cmd.Context(), args[0], req)
		if err != nil {
			return fmt.Errorf("flag set: %w", err)
		}

		enabled := "false"
		if f.Enabled {
			enabled = "true"
		}
		ui.Success(fmt.Sprintf("Updated %s (enabled=%s)", f.Key, enabled))
		return nil
	},
}

var flagEnableCmd = &cobra.Command{
	Use:   "enable <key>",
	Short: "Enable a feature flag",
	Long: `Enable a feature flag (sets enabled=true).

Example:
  nself flags enable ai.safety.jailbreak_filter`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL, _ := cmd.Flags().GetString("plugin-url")
		c := flags.NewClient(baseURL)
		f, err := c.Enable(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("flag enable: %w", err)
		}
		ui.Success(fmt.Sprintf("Enabled flag: %s", f.Key))
		return nil
	},
}

var flagDisableCmd = &cobra.Command{
	Use:   "disable <key>",
	Short: "Disable a feature flag",
	Long: `Disable a feature flag (sets enabled=false, broadcasts pubsub cache invalidation).

Example:
  nself flags disable ai.safety.jailbreak_filter`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL, _ := cmd.Flags().GetString("plugin-url")
		c := flags.NewClient(baseURL)
		f, err := c.Disable(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("flag disable: %w", err)
		}
		ui.Success(fmt.Sprintf("Disabled flag: %s", f.Key))
		return nil
	},
}

var flagKillCmd = &cobra.Command{
	Use:   "kill <key>",
	Short: "Kill-switch a feature flag (emergency disable with required reason)",
	Long: `Immediately kill a feature flag. This sets enabled=false, bypasses cache
TTL via pubsub broadcast to all SDK consumers, and writes an audit row.

Kill is the emergency path — use disable for routine toggling.
--reason is REQUIRED to prevent accidental kills.

Example:
  nself flags kill ai.safety.jailbreak_filter --reason "CVE-2026-1234 mitigation"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reason, _ := cmd.Flags().GetString("reason")
		baseURL, _ := cmd.Flags().GetString("plugin-url")

		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("flag kill: --reason is required (must be non-empty)")
		}

		c := flags.NewClient(baseURL)
		f, err := c.Kill(cmd.Context(), args[0], reason)
		if err != nil {
			return fmt.Errorf("flag kill: %w", err)
		}
		ui.Warn(fmt.Sprintf("Kill-switched flag: %s (reason: %s)", f.Key, reason))
		return nil
	},
}

var flagPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "List (or delete) stale feature flags",
	Long: `List feature flags that have exceeded their stale_after_days threshold
(default 90 days). Use --dry-run to preview without deleting.

Examples:
  nself flags prune --stale --dry-run
  nself flags prune --stale`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stale, _ := cmd.Flags().GetBool("stale")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		baseURL, _ := cmd.Flags().GetString("plugin-url")

		if !stale {
			return fmt.Errorf("flag prune: --stale is required")
		}

		c := flags.NewClient(baseURL)
		staleFlags, err := c.Prune(cmd.Context(), dryRun)
		if err != nil {
			return fmt.Errorf("flag prune: %w", err)
		}

		if len(staleFlags) == 0 {
			ui.Success("No stale flags found.")
			return nil
		}

		mode := "would delete"
		if !dryRun {
			mode = "deleted"
		}
		ui.CommandHeader("Stale Flags", fmt.Sprintf("%d flags %s", len(staleFlags), mode))
		t := ui.NewTable("KEY", "TYPE", "UPDATED")
		for _, f := range staleFlags {
			tp := f.Type
			if tp == "" {
				tp = "—"
			}
			t.AddRow(f.Key, tp, f.UpdatedAt.Format(time.RFC3339))
		}
		t.Render()

		if dryRun {
			ui.Info("Dry run — no flags were deleted. Run without --dry-run to delete.")
		}
		return nil
	},
}
