package commands

// sentry.go — 'nself sentry' command group.
//
// Purpose: Parent command for ɳSentry operations (status, alerts, etc.).
//   Prints help when invoked without a subcommand.
// Inputs:  subcommand
// Outputs: help text or delegates to subcommand
// Constraints: no flags at parent level; add flags on subcommands only.
// SPORT: CLI-CMD-SENTRY-001

import (
	"github.com/spf13/cobra"
)

var sentryCmd = &cobra.Command{
	Use:   "sentry",
	Short: "ɳSentry ops: status, alerts, and observability",
	Long: `Manage and inspect ɳSentry observability resources.

Examples:
  nself sentry status
  nself sentry status --json
  nself sentry status --url https://status.nself.org/status.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(sentryCmd)
}
