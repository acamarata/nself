package commands

import (
	"github.com/spf13/cobra"
)

// Purpose: Parent command for eval subcommands (gate, run, validate).
// Inputs: None directly; delegates to subcommands.
// Outputs: None; calls Help if invoked without subcommand.
// Constraints: Must be a no-op when invoked without subcommand.
// SPORT: F02-COMMAND-INVENTORY.md (command registry)

var ciEvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluation suite operations",
	Long: `Commands for managing and running evaluation suites, including tier gates.

Subcommands:
  gate    Check autonomy tier clearance via evaluation gates`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

func init() {
	ciEvalCmd.AddCommand(ciEvalGateCmd)
}
