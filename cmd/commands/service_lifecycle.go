package commands

// Purpose: Thin delegators for `nself service {start,stop,restart,ps,
// update,scale}` — each forwards straight into the internal/service
// package, with runServicePs additionally rendering the running-containers
// table. Split out of service.go (CLI-R12) to separate these small
// lifecycle-command handlers from the cobra command definitions and the
// add/upgrade/list/enable/disable/configure handlers in the other
// service_*.go files.
// Inputs: the cobra.Command + args for each subcommand (service name, and
// for scale, a replica count).
// Outputs: delegates to internal/service.{Start,Stop,Restart,PS,Update,
// Scale}; runServicePs prints a tabwriter-aligned table of running services.
// Constraints: pure move — no behavior changes.

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/service"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// service start / stop / restart / ps / update / scale
// ---------------------------------------------------------------------------

func runServiceStart(cmd *cobra.Command, args []string) error {
	return service.Start(cmd.Context(), args[0])
}

func runServiceStop(cmd *cobra.Command, args []string) error {
	return service.Stop(cmd.Context(), args[0])
}

func runServiceRestart(cmd *cobra.Command, args []string) error {
	return service.Restart(cmd.Context(), args[0])
}

func runServicePs(cmd *cobra.Command, args []string) error {
	entries, err := service.PS(cmd.Context())
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No services running. Run `nself start` to boot the stack.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSERVICE\tSTATUS\tHEALTH\tID")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Name, e.Service, e.Status, e.Health, e.ID)
	}
	return w.Flush()
}

func runServiceUpdate(cmd *cobra.Command, args []string) error {
	return service.Update(cmd.Context(), args[0])
}

func runServiceScale(cmd *cobra.Command, args []string) error {
	var replicas int
	if _, err := fmt.Sscanf(args[1], "%d", &replicas); err != nil {
		return fmt.Errorf("replicas must be an integer, got %q", args[1])
	}
	return service.Scale(cmd.Context(), args[0], replicas)
}

// catalogRow is the JSON shape of one `nself service list --core` row.
