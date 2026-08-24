package commands

// Purpose: Read-only feature-flag commands split out of flag.go (CLI-R12
// Batch B mechanical file-size split). Holds `nself flag list/get/history`
// — the three subcommands that only read state from the feature-flags
// plugin.
// Inputs: cobra command flags (--type, --json, --plugin-url) and the
// positional flag key for get/history.
// Outputs: a formatted table or JSON document; errors wrap
// internal/flags client failures.
// Constraints: pure move, no behavior change. flagCmd (parent) and the
// shared --plugin-url flag registration remain in flag.go's init().

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nself-org/cli/internal/flags"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

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
