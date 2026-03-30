package commands

import "github.com/spf13/cobra"

func init() {
	// TRAP-09: guard against duplicate registration if up/down are ever
	// added as first-class commands in the future.
	for _, c := range RootCmd.Commands() {
		if c.Name() == "up" || c.Name() == "down" {
			return
		}
	}

	upCmd := &cobra.Command{
		Use:    "up",
		Short:  "Alias for: nself start",
		Hidden: true,
		RunE:   startCmd.RunE,
	}

	downCmd := &cobra.Command{
		Use:    "down",
		Short:  "Alias for: nself stop",
		Hidden: true,
		RunE:   stopCmd.RunE,
	}

	RootCmd.AddCommand(upCmd, downCmd)
}
