package commands

// sentry_pages_alerts.go — 'nself sentry status-pages list|create' and
// 'nself sentry alerts channels|test'.
//
// Purpose: Status-page and alert-channel management against the ɳSentry API.
// Inputs:  cloud flag trio; create takes --name/--slug; test takes <channel-id>.
// Outputs: ui.Table (human) or --json.
// Constraints: tier limits (free 1 page, bundle 3, nself-plus 10) enforced
//   server-side; 402/429 surfaces the upgrade hint.
// SPORT: CLI-CMD-SENTRY-007

import (
	"fmt"

	"github.com/nself-org/cli/internal/sentryapi"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// ── status-pages ─────────────────────────────────────────────────────────────

var sentryStatusPagesCmd = &cobra.Command{
	Use:   "status-pages",
	Short: "Manage ɳSentry status pages",
	Long: `List and create public status pages.

Examples:
  nself sentry status-pages list
  nself sentry status-pages create --name "Public Status" --slug public`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sentryStatusPagesListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List status pages",
	SilenceUsage: true,
	RunE:         runSentryStatusPagesList,
}

var sentryStatusPagesCreateCmd = &cobra.Command{
	Use:          "create",
	Short:        "Create a status page",
	SilenceUsage: true,
	RunE:         runSentryStatusPagesCreate,
}

// ── alerts ───────────────────────────────────────────────────────────────────

var sentryAlertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage ɳSentry alert channels",
	Long: `List alert channels and send test notifications.

Examples:
  nself sentry alerts channels
  nself sentry alerts test <channel-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sentryAlertsChannelsCmd = &cobra.Command{
	Use:          "channels",
	Short:        "List alert channels",
	SilenceUsage: true,
	RunE:         runSentryAlertsChannels,
}

var sentryAlertsTestCmd = &cobra.Command{
	Use:          "test <channel-id>",
	Short:        "Send a test notification through a channel",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runSentryAlertsTest,
}

func init() {
	for _, c := range []*cobra.Command{sentryStatusPagesListCmd, sentryStatusPagesCreateCmd} {
		addSentryCloudFlags(c)
		sentryStatusPagesCmd.AddCommand(c)
	}
	sentryStatusPagesCreateCmd.Flags().String("name", "", "Status page display name (required)")
	sentryStatusPagesCreateCmd.Flags().String("slug", "", "URL slug — page served at sentry.nself.org/s/<slug> (required)")
	_ = sentryStatusPagesCreateCmd.MarkFlagRequired("name")
	_ = sentryStatusPagesCreateCmd.MarkFlagRequired("slug")

	for _, c := range []*cobra.Command{sentryAlertsChannelsCmd, sentryAlertsTestCmd} {
		addSentryCloudFlags(c)
		sentryAlertsCmd.AddCommand(c)
	}

	sentryCmd.AddCommand(sentryStatusPagesCmd, sentryAlertsCmd)
}

// runSentryStatusPagesList implements 'nself sentry status-pages list'.
func runSentryStatusPagesList(cmd *cobra.Command, _ []string) error {
	client := sentryClientFromCmd(cmd)
	pages, err := client.ListStatusPages(cmd.Context())
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		if pages == nil {
			pages = []sentryapi.StatusPage{}
		}
		return printSentryJSON(cmd, pages)
	}

	if len(pages) == 0 {
		ui.Info("No status pages. Create one: nself sentry status-pages create --name <name> --slug <slug>")
		return nil
	}

	tbl := ui.NewTable("ID", "Name", "Slug", "URL", "Public")
	for _, p := range pages {
		public := "no"
		if p.Public {
			public = "yes"
		}
		tbl.AddRow(p.ID, dashIfEmpty(p.Name), dashIfEmpty(p.Slug), dashIfEmpty(p.URL), public)
	}
	tbl.Render()
	fmt.Printf("\n%d status pages\n", len(pages))
	return nil
}

// runSentryStatusPagesCreate implements 'nself sentry status-pages create'.
func runSentryStatusPagesCreate(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	slug, _ := cmd.Flags().GetString("slug")

	client := sentryClientFromCmd(cmd)
	page, err := client.CreateStatusPage(cmd.Context(), sentryapi.CreateStatusPageRequest{Name: name, Slug: slug})
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		return printSentryJSON(cmd, page)
	}
	ui.Success(fmt.Sprintf("Status page created: %s — %s", page.Name, dashIfEmpty(page.URL)))
	return nil
}

// runSentryAlertsChannels implements 'nself sentry alerts channels'.
func runSentryAlertsChannels(cmd *cobra.Command, _ []string) error {
	client := sentryClientFromCmd(cmd)
	channels, err := client.ListAlertChannels(cmd.Context())
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		if channels == nil {
			channels = []sentryapi.AlertChannel{}
		}
		return printSentryJSON(cmd, channels)
	}

	if len(channels) == 0 {
		ui.Info("No alert channels configured. Add channels in the ɳSentry dashboard.")
		return nil
	}

	tbl := ui.NewTable("ID", "Kind", "Target", "Enabled")
	for _, ch := range channels {
		enabled := "no"
		if ch.Enabled {
			enabled = "yes"
		}
		tbl.AddRow(ch.ID, dashIfEmpty(ch.Kind), dashIfEmpty(ch.Target), enabled)
	}
	tbl.Render()
	fmt.Printf("\n%d channels\n", len(channels))
	return nil
}

// runSentryAlertsTest implements 'nself sentry alerts test <channel-id>'.
func runSentryAlertsTest(cmd *cobra.Command, args []string) error {
	client := sentryClientFromCmd(cmd)
	res, err := client.TestAlertChannel(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		return printSentryJSON(cmd, res)
	}
	if res.Delivered {
		ui.Success(fmt.Sprintf("Test notification delivered: %s", dashIfEmpty(res.Detail)))
		return nil
	}
	return fmt.Errorf("test notification NOT delivered: %s", dashIfEmpty(res.Detail))
}
