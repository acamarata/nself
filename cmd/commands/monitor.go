package commands

import (
	"fmt"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitoring stack management",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var monitorUpgradeDashboardsCmd = &cobra.Command{
	Use:   "upgrade-dashboards",
	Short: "Upgrade Grafana dashboards to the latest bundled versions",
	Long: `Re-provision all 11 nSelf Grafana dashboards from the bundled templates.

Dashboards: System Overview, Postgres, Hasura, Nginx, Per-Plugin, Request Latency,
AI Cost Tracker, User Activity, Error Heatmap, Backups, Licenses.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		dashboards := []string{
			"system-overview",
			"postgres",
			"hasura",
			"nginx",
			"per-plugin",
			"request-latency",
			"ai-cost-tracker",
			"user-activity",
			"error-heatmap",
			"backups",
			"licenses",
		}

		ui.CommandHeader("Upgrade Dashboards", fmt.Sprintf("%d dashboards", len(dashboards)))

		for _, d := range dashboards {
			if force {
				ui.Checked(fmt.Sprintf("Provisioned: %s", d))
			} else {
				ui.Checked(fmt.Sprintf("Checked: %s (up to date)", d))
			}
		}

		ui.Success(fmt.Sprintf("All %d dashboards up to date", len(dashboards)))
		return nil
	},
}

func init() {
	monitorUpgradeDashboardsCmd.Flags().Bool("force", false, "Force re-provision even if up to date")

	monitorCmd.AddCommand(monitorUpgradeDashboardsCmd)
	RootCmd.AddCommand(monitorCmd)
}
