package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nself-org/cli/internal/alerts"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage Prometheus alert rules and silences",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured alert rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		severity, _ := cmd.Flags().GetString("severity")
		jsonOut, _ := cmd.Flags().GetBool("json")

		rules := alerts.List(severity)

		if jsonOut {
			data, err := json.MarshalIndent(rules, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		ui.CommandHeader("Alert Rules", fmt.Sprintf("%d rules", len(rules)))
		for _, r := range rules {
			fmt.Printf("  %s [%s] for=%s\n    %s\n    expr: %s\n\n",
				r.Name, r.Severity, r.For, r.Summary, r.Expr)
		}
		return nil
	},
}

var alertsSilenceCmd = &cobra.Command{
	Use:   "silence <alert-name>",
	Short: "Silence an alert for a specified duration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		durationStr, _ := cmd.Flags().GetString("duration")
		reason, _ := cmd.Flags().GetString("reason")

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", durationStr, err)
		}

		cwd, _ := os.Getwd()
		silence, err := alerts.CreateSilence(cmd.Context(), args[0], reason, duration, cwd)
		if err != nil {
			return err
		}

		ui.Success(fmt.Sprintf("Silenced %s until %s (id: %s)", args[0],
			silence.ExpiresAt.Format(time.RFC3339), silence.ID))
		return nil
	},
}

var alertsTestCmd = &cobra.Command{
	Use:   "test <alert-name>",
	Short: "Fire a synthetic test alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("alertmanager-url")
		if err := alerts.TestAlert(cmd.Context(), args[0], url); err != nil {
			return fmt.Errorf("test alert failed: %w", err)
		}
		ui.Success(fmt.Sprintf("Test alert %q sent to Alertmanager", args[0]))
		return nil
	},
}

func init() {
	alertsListCmd.Flags().String("severity", "", "Filter by severity (P1, P2, P3)")
	alertsListCmd.Flags().Bool("json", false, "JSON output")

	alertsSilenceCmd.Flags().String("duration", "2h", "Silence duration (e.g. 2h, 30m)")
	alertsSilenceCmd.Flags().String("reason", "", "Reason for silencing")
	_ = alertsSilenceCmd.MarkFlagRequired("reason")

	alertsTestCmd.Flags().String("alertmanager-url", "", "Alertmanager URL (default: http://127.0.0.1:9093)")

	alertsCmd.AddCommand(alertsListCmd, alertsSilenceCmd, alertsTestCmd)
	RootCmd.AddCommand(alertsCmd)
}
