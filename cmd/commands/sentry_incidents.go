package commands

// sentry_incidents.go — 'nself sentry incidents list|ack|resolve'.
//
// Purpose: Incident lifecycle against the ɳSentry API (open → ack → resolved).
// Inputs:  cloud flag trio; list supports --status filter.
// Outputs: ui.Table (human) or --json.
// Constraints: ack/resolve are idempotent server-side; non-2xx surfaces the
//   server error message.
// SPORT: CLI-CMD-SENTRY-006

import (
	"fmt"

	"github.com/nself-org/cli/internal/sentryapi"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var sentryIncidentsCmd = &cobra.Command{
	Use:   "incidents",
	Short: "Manage ɳSentry incidents",
	Long: `List, acknowledge, and resolve ɳSentry incidents.

Examples:
  nself sentry incidents list
  nself sentry incidents list --status open
  nself sentry incidents ack <id>
  nself sentry incidents resolve <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sentryIncidentsListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List incidents",
	SilenceUsage: true,
	RunE:         runSentryIncidentsList,
}

var sentryIncidentsAckCmd = &cobra.Command{
	Use:          "ack <id>",
	Short:        "Acknowledge an incident",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSentryIncidentAction(cmd, args[0], "ack")
	},
}

var sentryIncidentsResolveCmd = &cobra.Command{
	Use:          "resolve <id>",
	Short:        "Resolve an incident",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSentryIncidentAction(cmd, args[0], "resolve")
	},
}

func init() {
	for _, c := range []*cobra.Command{sentryIncidentsListCmd, sentryIncidentsAckCmd, sentryIncidentsResolveCmd} {
		addSentryCloudFlags(c)
		sentryIncidentsCmd.AddCommand(c)
	}
	sentryIncidentsListCmd.Flags().String("status", "", "Filter by status: open, acknowledged, or resolved")
	sentryCmd.AddCommand(sentryIncidentsCmd)
}

// runSentryIncidentsList implements 'nself sentry incidents list'.
func runSentryIncidentsList(cmd *cobra.Command, _ []string) error {
	status, _ := cmd.Flags().GetString("status")
	client := sentryClientFromCmd(cmd)
	incidents, err := client.ListIncidents(cmd.Context(), status)
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		if incidents == nil {
			incidents = []sentryapi.Incident{}
		}
		return printSentryJSON(cmd, incidents)
	}

	if len(incidents) == 0 {
		ui.Info("No incidents. All quiet.")
		return nil
	}

	tbl := ui.NewTable("ID", "Title", "Status", "Severity", "Started", "Monitor")
	for _, inc := range incidents {
		tbl.AddRow(inc.ID, dashIfEmpty(inc.Title), dashIfEmpty(inc.Status),
			dashIfEmpty(inc.Severity), dashIfEmpty(inc.StartedAt), dashIfEmpty(inc.MonitorID))
	}
	tbl.Render()
	fmt.Printf("\n%d incidents\n", len(incidents))
	return nil
}

// runSentryIncidentAction implements ack/resolve.
func runSentryIncidentAction(cmd *cobra.Command, id, action string) error {
	client := sentryClientFromCmd(cmd)
	var (
		inc *sentryapi.Incident
		err error
	)
	if action == "ack" {
		inc, err = client.AckIncident(cmd.Context(), id)
	} else {
		inc, err = client.ResolveIncident(cmd.Context(), id)
	}
	if err != nil {
		return err
	}
	if sentryJSONFlag(cmd) {
		return printSentryJSON(cmd, inc)
	}
	ui.Success(fmt.Sprintf("Incident %s is now %s.", inc.ID, inc.Status))
	return nil
}
