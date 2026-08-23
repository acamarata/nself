package commands

// Purpose: The `nself secrets schedule`, `list-schedules`, and `verify`
// subcommands — creating/showing named rotation schedules and checking
// that a named secret is present in the store. Split out of secrets.go
// (CLI-R12); see secrets_core.go for the file-splitting rationale shared
// by every secrets_*.go file in this split.
// Inputs: cobra.Command args/flags per subcommand (secret name, rotation
// interval).
// Outputs: an updated schedule record, or a printed schedule/verify report.
// Constraints: pure move — no behavior changes.

import (
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/secrets"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var secretsScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Add a rotation schedule or show all schedule statuses",
	Long: `Add a named rotation schedule, or show the status of all tracked schedules.

If --secret and --every are provided, the schedule is created or updated.
Otherwise, the current schedule table is printed.

Examples:
  nself secrets schedule --secret JWT_SIGNING_KEY --every 90d
  nself secrets schedule`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		secretName, _ := cmd.Flags().GetString("secret")
		everyFlag, _ := cmd.Flags().GetString("every")

		// If both --secret and --every are supplied, create/update the schedule.
		if secretName != "" && everyFlag != "" {
			var cadenceDays int
			if _, err := fmt.Sscanf(everyFlag, "%dd", &cadenceDays); err != nil {
				return fmt.Errorf("--every must be in format <N>d (e.g. 90d): %w", err)
			}
			if err := secrets.AddSchedule(cwd, secretName, cadenceDays, 7); err != nil {
				return err
			}
			fmt.Printf("Rotation schedule set: %s every %dd.\n", secretName, cadenceDays)
			return nil
		}

		// Default: show schedule table.
		checks, err := secrets.CheckSchedule(cwd)
		if err != nil {
			return err
		}
		if len(checks) == 0 {
			fmt.Println("No rotation schedules configured.")
			return nil
		}
		tbl := ui.NewTable("Secret", "Cadence", "Window", "Next Due", "Due In", "Status")
		for _, c := range checks {
			dueIn := "-"
			if c.DueInDays >= 0 {
				dueIn = fmt.Sprintf("%dd", c.DueInDays)
			}
			nextDue := c.NextDue
			if len(nextDue) > 10 {
				nextDue = nextDue[:10]
			}
			if nextDue == "" {
				nextDue = "-"
			}
			tbl.AddRow(c.SecretName, fmt.Sprintf("%dd", c.CadenceDays),
				fmt.Sprintf("%dd", c.WindowDays), nextDue, dueIn, c.Status)
		}
		tbl.Render()
		return nil
	},
}

// secretsListSchedulesCmd is an alias for schedule (show only) per the F08 spec.
var secretsListSchedulesCmd = &cobra.Command{
	Use:   "list-schedules",
	Short: "List all rotation schedule statuses (alias for: secrets schedule)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		checks, err := secrets.CheckSchedule(cwd)
		if err != nil {
			return err
		}
		if len(checks) == 0 {
			fmt.Println("No rotation schedules configured.")
			return nil
		}
		tbl := ui.NewTable("Secret", "Cadence", "Window", "Next Due", "Due In", "Status")
		for _, c := range checks {
			dueIn := "-"
			if c.DueInDays >= 0 {
				dueIn = fmt.Sprintf("%dd", c.DueInDays)
			}
			nextDue := c.NextDue
			if len(nextDue) > 10 {
				nextDue = nextDue[:10]
			}
			if nextDue == "" {
				nextDue = "-"
			}
			tbl.AddRow(c.SecretName, fmt.Sprintf("%dd", c.CadenceDays),
				fmt.Sprintf("%dd", c.WindowDays), nextDue, dueIn, c.Status)
		}
		tbl.Render()
		return nil
	},
}

// secretsVerifyCmd checks that a named secret is present in the store.
var secretsVerifyCmd = &cobra.Command{
	Use:   "verify <KEY>",
	Short: "Verify that a named secret exists in the store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if err := secrets.VerifySecretExists(cwd, secretsEnvFlag, args[0]); err != nil {
			return err
		}
		fmt.Printf("Secret %s: PRESENT in %s environment.\n", args[0], secretsEnvFlag)
		return nil
	},
}

// secretsRotationLogCmd prints the rotation event log.
