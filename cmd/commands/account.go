package commands

import (
	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────────────

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage your nSelf account, sessions, licenses, team, and devices",
	Long: `Manage your nSelf account.

With no subcommand, shows the current account status (same as 'nself account status').

Subcommands:
  login      Log in via device-code OAuth flow
  logout     Revoke current session and clear local credentials
  status     Show current account summary (default)
  team       List team members; --invite, --remove, --role
  licenses   List active licenses; --activate <id>
  devices    List registered devices; --revoke <id>
  transfer   Move a license to another account`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show status.
		return runAccountStatus(cmd, args)
	},
}

func init() {
	accountCmd.AddCommand(accountLoginCmd)
	accountCmd.AddCommand(accountLogoutCmd)
	accountCmd.AddCommand(accountStatusCmd)
	accountCmd.AddCommand(accountTeamCmd)
	accountCmd.AddCommand(accountLicensesCmd)
	accountCmd.AddCommand(accountDevicesCmd)
	accountCmd.AddCommand(accountTransferCmd)
	RootCmd.AddCommand(accountCmd)
}

// ── account login ────────────────────────────────────────────────────────────
