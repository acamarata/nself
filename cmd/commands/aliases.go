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

	// TRAP-10: guard against duplicate registration if trust is ever added as
	// a first-class command.
	for _, c := range RootCmd.Commands() {
		if c.Name() == "trust" {
			RootCmd.AddCommand(upCmd, downCmd)
			return
		}
	}

	trustCmd := &cobra.Command{
		Use:   "trust",
		Short: "Alias for: nself dns-setup",
		Long:  "trust is a convenience alias for dns-setup. See 'nself dns-setup --help' for full usage.",
		RunE:  dnsSetupCmd.RunE,
	}
	trustCmd.Flags().BoolP("dry-run", "n", false, "Print entries that would be added without writing")

	RootCmd.AddCommand(upCmd, downCmd, trustCmd)
}
