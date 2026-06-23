package commands

import (
	"github.com/spf13/cobra"
)

// Purpose: Root command for CI-related operations (evaluation, gates, testing).
// Inputs: None directly; subcommands receive tier, suite, repo flags.
// Outputs: None; delegates to subcommands.
// Constraints: Must be a no-op when invoked without subcommand (call Help).
// SPORT: F02-COMMAND-INVENTORY.md (command registry)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "CI and evaluation operations",
	Long: `Commands for CI integration, including eval gates and tier verification.

Subcommands:
  eval gate    Check autonomy tier clearance via evaluation gates`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

func init() {
	ciCmd.AddCommand(ciEvalCmd)
	RootCmd.AddCommand(ciCmd)
}
