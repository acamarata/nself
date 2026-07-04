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

Cloud subcommands (login, monitors, incidents, status-pages, alerts, whoami)
target the hosted SaaS at ` + "`https://api.sentry.nself.org`" + ` by default, or a
self-hosted/local sentry bundle via --api-url / NSELF_SENTRY_API_URL.
Auth: API key (nsk_*) via 'nself sentry login', --api-key, or NSELF_SENTRY_API_KEY.

A symlinked binary named 'nsentry' behaves as this command group:
  ln -s $(which nself) /usr/local/bin/nsentry
  nsentry monitors list

Examples:
  nself sentry login --api-key nsk_abc123...
  nself sentry monitors add --name api --url https://api.example.com --interval 60s
  nself sentry incidents list --status open
  nself sentry whoami --json
  nself sentry status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(sentryCmd)
}
