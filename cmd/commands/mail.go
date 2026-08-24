package commands

// mail.go — `nself mail` top-level subcommand. Wraps the mux + Postmark
// plugins via ping_api /mail/* endpoints. License-gated: requires the
// ɳSelf+ or nClaw bundle (which ships the Postmark plugin).
//
// Subcommands:
//   send                 transactional email through mux pipeline
//   broadcast            list-template broadcast send
//   status               delivery status by message id
//   templates list       list registered Postmark templates
//   dkim verify          verify DKIM record for a domain
//
// All subcommands honour --json for machine-readable output.
//
// Subcommand RunE bodies now live in mail_transactional.go (send/broadcast/
// status) and mail_admin.go (templates/dkim) — CLI-R12 Batch B mechanical
// file-size split. This file keeps the shared client/license/output
// helpers, the root command, error mapping, and cobra registration.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/mail"
	"github.com/nself-org/cli/internal/plugin"

	"github.com/spf13/cobra"
)

// mailExitNoLicense is the process exit code when no license key is configured.
const mailExitNoLicense = 2

// resolveMailClient picks the ping_api base URL from env (NSELF_PING_API_URL,
// default https://ping.nself.org) and the first configured license key from
// internal/license.CollectLicenseKeys (which already reads NSELF_LICENSE_KEY,
// NSELF_PLUGIN_LICENSE_KEY, and NSELF_LICENSE_KEY_1..10).
//
// Returns (nil, nil) when no license is configured so callers can emit the
// canonical "requires nSelf+ or nClaw bundle" message and exit with code 2.
func resolveMailClient() (*mail.Client, error) {
	pingURL := os.Getenv("NSELF_PING_API_URL")
	if pingURL == "" {
		pingURL = mail.DefaultPingURL
	}
	keys := license.CollectLicenseKeys()
	if len(keys) == 0 {
		return nil, nil
	}
	return mail.New(pingURL, keys[0]), nil
}

// requireLicense prints the canonical "no license" message to stderr and
// returns a *plugin.ExitCodeError so main() exits with code 2. main()
// short-circuits on ExitCodeError without printing, so we print here.
func requireLicense(cmd *cobra.Command) error {
	fmt.Fprintln(cmd.ErrOrStderr(), "Error: nself mail requires nSelf+ or nClaw bundle (Postmark plugin) — run 'nself license add <key>'")
	return &plugin.ExitCodeError{Code: mailExitNoLicense}
}

// printResult emits either JSON or a human-readable rendering. The renderer
// is invoked only when --json is false. If renderer is nil, the value is
// printed as a key:value block via reflection-free JSON marshal indent.
func printResult(jsonMode bool, v interface{}, render func()) error {
	if jsonMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	if render != nil {
		render()
	}
	return nil
}

// ── Root command ──────────────────────────────────────────────────────────

var mailCmd = &cobra.Command{
	Use:   "mail",
	Short: "Send transactional and broadcast email through the nSelf stack",
	Long: `Send transactional and broadcast email through the nSelf stack.

The 'nself mail' command wraps the mux + Postmark plugins. ping_api proxies
each call to the running stack, so the Postmark plugin must be installed
and a valid license key must be configured.

Subcommands:
  send         Send a single transactional email
  broadcast    Send a broadcast to a list using a saved template
  status       Query delivery status for a message
  templates    Manage Postmark templates (list)
  dkim         Manage DKIM (verify)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── error mapping ─────────────────────────────────────────────────────────

// mapMailError converts mail-package sentinel errors into user-readable
// cobra errors. 4xx → short message, 5xx → flagged as ping_api issue, network
// → guidance to check connectivity.
func mapMailError(err error) error {
	switch {
	case errors.Is(err, mail.ErrUnauthorized):
		return errors.New("license rejected by ping.nself.org — run 'nself license validate'")
	case errors.Is(err, mail.ErrUnreachable):
		return errors.New("ping.nself.org unreachable — check connectivity")
	case errors.Is(err, mail.ErrServer):
		return fmt.Errorf("ping_api server error: %w", err)
	default:
		return err
	}
}

// ── registration ──────────────────────────────────────────────────────────

func init() {
	// send flags
	mailSendCmd.Flags().String("to", "", "Recipient email address")
	mailSendCmd.Flags().String("subject", "", "Email subject")
	mailSendCmd.Flags().String("body", "", "Email body (inline)")
	mailSendCmd.Flags().String("body-file", "", "Read body from file ('-' for stdin)")
	mailSendCmd.Flags().String("body-type", "text", "Body type: text or html")
	mailSendCmd.Flags().Bool("json", false, "Output as JSON")

	// broadcast flags
	mailBroadcastCmd.Flags().String("list", "", "Mailing list ID")
	mailBroadcastCmd.Flags().String("template", "", "Postmark template ID")
	mailBroadcastCmd.Flags().Bool("json", false, "Output as JSON")

	// status flags
	mailStatusCmd.Flags().String("message-id", "", "Message ID returned by 'nself mail send'")
	mailStatusCmd.Flags().Bool("json", false, "Output as JSON")

	// templates list flags
	mailTemplatesListCmd.Flags().Bool("json", false, "Output as JSON")

	// dkim verify flags
	mailDKIMVerifyCmd.Flags().String("domain", "", "Domain to verify (e.g. example.com)")
	mailDKIMVerifyCmd.Flags().Bool("json", false, "Output as JSON")

	mailTemplatesCmd.AddCommand(mailTemplatesListCmd)
	mailDKIMCmd.AddCommand(mailDKIMVerifyCmd)

	mailCmd.AddCommand(mailSendCmd)
	mailCmd.AddCommand(mailBroadcastCmd)
	mailCmd.AddCommand(mailStatusCmd)
	mailCmd.AddCommand(mailTemplatesCmd)
	mailCmd.AddCommand(mailDKIMCmd)

	RootCmd.AddCommand(mailCmd)
}
