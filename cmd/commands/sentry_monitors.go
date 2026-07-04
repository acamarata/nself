package commands

// sentry_monitors.go — 'nself sentry monitors list|add|rm|pause|resume'.
//
// Purpose: Manage uptime monitors on the ɳSentry API (SaaS or self-hosted).
// Inputs:  cloud flag trio + per-subcommand flags (--url, --name, --kind, --interval).
// Outputs: ui.Table (human) or --json.
// Constraints: quota errors (402/429) surface the server message + upgrade hint.
// SPORT: CLI-CMD-SENTRY-005

import (
	"fmt"
	"time"

	"github.com/nself-org/cli/internal/sentryapi"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var sentryMonitorsCmd = &cobra.Command{
	Use:   "monitors",
	Short: "Manage ɳSentry uptime monitors",
	Long: `Manage uptime monitors on ɳSentry.

Examples:
  nself sentry monitors list
  nself sentry monitors add --name api --url https://api.example.com --interval 60s
  nself sentry monitors rm <id>
  nself sentry monitors pause <id>
  nself sentry monitors resume <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sentryMonitorsListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List monitors",
	SilenceUsage: true,
	RunE:         runSentryMonitorsList,
}

var sentryMonitorsAddCmd = &cobra.Command{
	Use:          "add",
	Short:        "Add a monitor",
	SilenceUsage: true,
	RunE:         runSentryMonitorsAdd,
}

var sentryMonitorsRmCmd = &cobra.Command{
	Use:          "rm <id>",
	Short:        "Remove a monitor",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runSentryMonitorsRm,
}

var sentryMonitorsPauseCmd = &cobra.Command{
	Use:          "pause <id>",
	Short:        "Pause a monitor (stops checks, keeps history)",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSentryMonitorsPause(cmd, args[0], true)
	},
}

var sentryMonitorsResumeCmd = &cobra.Command{
	Use:          "resume <id>",
	Short:        "Resume a paused monitor",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSentryMonitorsPause(cmd, args[0], false)
	},
}

func init() {
	for _, c := range []*cobra.Command{
		sentryMonitorsListCmd, sentryMonitorsAddCmd, sentryMonitorsRmCmd,
		sentryMonitorsPauseCmd, sentryMonitorsResumeCmd,
	} {
		addSentryCloudFlags(c)
		sentryMonitorsCmd.AddCommand(c)
	}

	sentryMonitorsAddCmd.Flags().String("name", "", "Monitor display name (defaults to the URL host)")
	sentryMonitorsAddCmd.Flags().String("url", "", "Target URL or host to monitor (required)")
	sentryMonitorsAddCmd.Flags().String("kind", "http", "Monitor kind: http, tcp, or ping")
	sentryMonitorsAddCmd.Flags().Duration("interval", 60*time.Second, "Check interval (tier floors apply: free 5m, bundle 1m, nself-plus 30s)")
	_ = sentryMonitorsAddCmd.MarkFlagRequired("url")

	sentryCmd.AddCommand(sentryMonitorsCmd)
}

// runSentryMonitorsList implements 'nself sentry monitors list'.
func runSentryMonitorsList(cmd *cobra.Command, _ []string) error {
	client := sentryClientFromCmd(cmd)
	monitors, err := client.ListMonitors(cmd.Context())
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		if monitors == nil {
			monitors = []sentryapi.Monitor{}
		}
		return printSentryJSON(cmd, monitors)
	}

	if len(monitors) == 0 {
		ui.Info("No monitors yet. Add one: nself sentry monitors add --url https://example.com")
		return nil
	}

	tbl := ui.NewTable("ID", "Name", "URL", "Kind", "Interval", "Status")
	for _, m := range monitors {
		status := m.Status
		if m.Paused {
			status = "paused"
		}
		tbl.AddRow(m.ID, dashIfEmpty(m.Name), m.URL, dashIfEmpty(m.Kind),
			(time.Duration(m.IntervalSeconds) * time.Second).String(), dashIfEmpty(status))
	}
	tbl.Render()
	fmt.Printf("\n%d monitors\n", len(monitors))
	return nil
}

// runSentryMonitorsAdd implements 'nself sentry monitors add'.
func runSentryMonitorsAdd(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	target, _ := cmd.Flags().GetString("url")
	kind, _ := cmd.Flags().GetString("kind")
	interval, _ := cmd.Flags().GetDuration("interval")

	if kind != "http" && kind != "tcp" && kind != "ping" {
		return fmt.Errorf("invalid --kind %q (use http, tcp, or ping)", kind)
	}
	if name == "" {
		name = target
	}

	client := sentryClientFromCmd(cmd)
	m, err := client.CreateMonitor(cmd.Context(), sentryapi.CreateMonitorRequest{
		Name:            name,
		URL:             target,
		Kind:            kind,
		IntervalSeconds: int(interval.Seconds()),
	})
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		return printSentryJSON(cmd, m)
	}
	ui.Success(fmt.Sprintf("Monitor created: %s (%s every %s) — id %s", m.Name, m.URL,
		(time.Duration(m.IntervalSeconds) * time.Second).String(), m.ID))
	return nil
}

// runSentryMonitorsRm implements 'nself sentry monitors rm <id>'.
func runSentryMonitorsRm(cmd *cobra.Command, args []string) error {
	client := sentryClientFromCmd(cmd)
	if err := client.DeleteMonitor(cmd.Context(), args[0]); err != nil {
		return err
	}
	if sentryJSONFlag(cmd) {
		return printSentryJSON(cmd, map[string]string{"deleted": args[0]})
	}
	ui.Success(fmt.Sprintf("Monitor %s deleted.", args[0]))
	return nil
}

// runSentryMonitorsPause implements pause/resume.
func runSentryMonitorsPause(cmd *cobra.Command, id string, pause bool) error {
	client := sentryClientFromCmd(cmd)
	m, err := client.PauseMonitor(cmd.Context(), id, pause)
	if err != nil {
		return err
	}
	if sentryJSONFlag(cmd) {
		return printSentryJSON(cmd, m)
	}
	verb := "paused"
	if !pause {
		verb = "resumed"
	}
	ui.Success(fmt.Sprintf("Monitor %s %s.", id, verb))
	return nil
}
