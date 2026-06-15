package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/flags"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var flagCmd = &cobra.Command{
	Use:   "flag",
	Short: "Manage feature flags",
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

var flagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all feature flags",
	Long: `List all feature flags. Optionally filter by type.

Examples:
  nself flag list
  nself flag list --type release
  nself flag list --type kill_switch --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		flagType, _ := cmd.Flags().GetString("type")
		jsonOut, _ := cmd.Flags().GetBool("json")
		baseURL, _ := cmd.Flags().GetString("plugin-url")

		c := flags.NewClient(baseURL)
		fs, err := c.List(cmd.Context(), flagType)
		if err != nil {
			return fmt.Errorf("flag list: %w", err)
		}

		if jsonOut {
			data, err := json.MarshalIndent(fs, "", "  ")
			if err != nil {
				return fmt.Errorf("flag list: marshal: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if len(fs) == 0 {
			ui.Info("No feature flags found. Create one via the plugin REST API or Admin UI.")
			return nil
		}

		ui.CommandHeader("Feature Flags", fmt.Sprintf("%d flags", len(fs)))
		t := ui.NewTable("KEY", "TYPE", "ENABLED", "ROLLOUT", "UPDATED")
		for _, f := range fs {
			tp := f.Type
			if tp == "" {
				tp = "—"
			}
			enabled := "false"
			if f.Enabled {
				enabled = "true"
			}
			rollout := "—"
			if f.RolloutPct != nil {
				rollout = fmt.Sprintf("%d%%", *f.RolloutPct)
			}
			t.AddRow(f.Key, tp, enabled, rollout, f.UpdatedAt.Format(time.RFC3339))
		}
		t.Render()
		return nil
	},
}

var flagGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a single feature flag",
	Long: `Get a feature flag by key and display its full configuration.

Example:
  nself flag get ai.safety.jailbreak_filter`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		baseURL, _ := cmd.Flags().GetString("plugin-url")

		c := flags.NewClient(baseURL)
		f, err := c.Get(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("flag get: %w", err)
		}

		if jsonOut {
			data, err := json.MarshalIndent(f, "", "  ")
			if err != nil {
				return fmt.Errorf("flag get: marshal: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		ui.CommandHeader("Feature Flag", args[0])
		tp := f.Type
		if tp == "" {
			tp = "—"
		}
		enabled := "false"
		if f.Enabled {
			enabled = "true"
		}
		rollout := "—"
		if f.RolloutPct != nil {
			rollout = fmt.Sprintf("%d%%", *f.RolloutPct)
		}
		name := "—"
		if f.Name != nil {
			name = *f.Name
		}
		desc := "—"
		if f.Description != nil {
			desc = *f.Description
		}

		t := ui.NewTable("FIELD", "VALUE")
		t.AddRow("Key", f.Key)
		t.AddRow("Name", name)
		t.AddRow("Description", desc)
		t.AddRow("Type", tp)
		t.AddRow("Enabled", enabled)
		t.AddRow("Rollout", rollout)
		t.AddRow("Created", f.CreatedAt.Format(time.RFC3339))
		t.AddRow("Updated", f.UpdatedAt.Format(time.RFC3339))
		t.Render()
		return nil
	},
}

var flagSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Update a feature flag's enabled state and/or rollout percentage",
	Long: `Update a feature flag. At least one of --enabled or --rollout-pct is required.

Examples:
  nself flag set ai.safety.jailbreak_filter --enabled
  nself flag set ai.safety.jailbreak_filter --rollout-pct 25
  nself flag set ai.safety.jailbreak_filter --enabled --rollout-pct 50`,
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
  nself flag enable ai.safety.jailbreak_filter`,
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
  nself flag disable ai.safety.jailbreak_filter`,
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
  nself flag kill ai.safety.jailbreak_filter --reason "CVE-2026-1234 mitigation"`,
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

var flagHistoryCmd = &cobra.Command{
	Use:   "history <key>",
	Short: "Show audit log for a feature flag",
	Long: `Display the audit log for a feature flag. Shows actor, action, and
before/after states for all state-changing operations.

Example:
  nself flag history ai.safety.jailbreak_filter`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		baseURL, _ := cmd.Flags().GetString("plugin-url")

		c := flags.NewClient(baseURL)
		entries, err := c.History(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("flag history: %w", err)
		}

		if jsonOut {
			data, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return fmt.Errorf("flag history: marshal: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if len(entries) == 0 {
			ui.Info(fmt.Sprintf("No audit entries for flag: %s", args[0]))
			return nil
		}

		ui.CommandHeader(fmt.Sprintf("Audit Log: %s", args[0]), fmt.Sprintf("%d entries", len(entries)))
		t := ui.NewTable("TIME", "ACTOR", "ACTION", "REASON")
		for _, e := range entries {
			reason := "—"
			if e.Reason != nil {
				reason = *e.Reason
			}
			t.AddRow(e.Ts.Format(time.RFC3339), e.Actor, e.Action, reason)
		}
		t.Render()
		return nil
	},
}

var flagPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "List (or delete) stale feature flags",
	Long: `List feature flags that have exceeded their stale_after_days threshold
(default 90 days). Use --dry-run to preview without deleting.

Examples:
  nself flag prune --stale --dry-run
  nself flag prune --stale`,
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
