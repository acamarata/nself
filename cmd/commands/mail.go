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

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/mail"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"

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

// ── send ──────────────────────────────────────────────────────────────────

var mailSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a single transactional email",
	Long: `Send a single transactional email through the mux pipeline.

The body can be supplied inline with --body or read from a file with
--body-file (use '-' for stdin). At least one of --body or --body-file
must be set.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		to, _ := cmd.Flags().GetString("to")
		subject, _ := cmd.Flags().GetString("subject")
		body, _ := cmd.Flags().GetString("body")
		bodyFile, _ := cmd.Flags().GetString("body-file")
		bodyType, _ := cmd.Flags().GetString("body-type")
		jsonMode, _ := cmd.Flags().GetBool("json")

		if strings.TrimSpace(to) == "" {
			return fmt.Errorf("--to <addr> is required")
		}
		if strings.TrimSpace(subject) == "" {
			return fmt.Errorf("--subject <s> is required")
		}
		if body == "" && bodyFile == "" {
			return fmt.Errorf("either --body <text> or --body-file <path> is required")
		}
		if body != "" && bodyFile != "" {
			return fmt.Errorf("--body and --body-file are mutually exclusive")
		}
		if bodyFile != "" {
			b, err := readBodyFile(bodyFile)
			if err != nil {
				return fmt.Errorf("reading --body-file: %w", err)
			}
			body = b
		}

		client, err := resolveMailClient()
		if err != nil {
			return err
		}
		if client == nil {
			return requireLicense(cmd)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		resp, err := client.Send(ctx, mail.SendRequest{
			To:       to,
			Subject:  subject,
			Body:     body,
			BodyType: bodyType,
		})
		if err != nil {
			return mapMailError(err)
		}

		return printResult(jsonMode, resp, func() {
			ui.Success(fmt.Sprintf("Email queued for %s", resp.To))
			fmt.Printf("  Message ID: %s\n", resp.MessageID)
			fmt.Printf("  Accepted:   %t\n", resp.Accepted)
		})
	},
}

// readBodyFile reads --body-file. "-" means stdin.
func readBodyFile(path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ── broadcast ─────────────────────────────────────────────────────────────

var mailBroadcastCmd = &cobra.Command{
	Use:   "broadcast",
	Short: "Send a broadcast to a list using a saved template",
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, _ := cmd.Flags().GetString("list")
		tmplID, _ := cmd.Flags().GetString("template")
		jsonMode, _ := cmd.Flags().GetBool("json")

		if strings.TrimSpace(listID) == "" {
			return fmt.Errorf("--list <list-id> is required")
		}
		if strings.TrimSpace(tmplID) == "" {
			return fmt.Errorf("--template <tpl-id> is required")
		}

		client, err := resolveMailClient()
		if err != nil {
			return err
		}
		if client == nil {
			return requireLicense(cmd)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		resp, err := client.Broadcast(ctx, mail.BroadcastRequest{
			ListID:     listID,
			TemplateID: tmplID,
		})
		if err != nil {
			return mapMailError(err)
		}

		return printResult(jsonMode, resp, func() {
			ui.Success(fmt.Sprintf("Broadcast queued: batch %s", resp.BatchID))
			fmt.Printf("  Recipients: %d\n", resp.Recipients)
			fmt.Printf("  Queued:     %t\n", resp.Queued)
		})
	},
}

// ── status ────────────────────────────────────────────────────────────────

var mailStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Query delivery status for a sent message",
	RunE: func(cmd *cobra.Command, args []string) error {
		msgID, _ := cmd.Flags().GetString("message-id")
		jsonMode, _ := cmd.Flags().GetBool("json")

		if strings.TrimSpace(msgID) == "" {
			return fmt.Errorf("--message-id <id> is required")
		}

		client, err := resolveMailClient()
		if err != nil {
			return err
		}
		if client == nil {
			return requireLicense(cmd)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		resp, err := client.Status(ctx, msgID)
		if err != nil {
			return mapMailError(err)
		}

		return printResult(jsonMode, resp, func() {
			fmt.Printf("Message ID: %s\n", resp.MessageID)
			fmt.Printf("Status:     %s\n", resp.Status)
			if resp.To != "" {
				fmt.Printf("To:         %s\n", resp.To)
			}
			if resp.DeliveredAt != "" {
				fmt.Printf("Delivered:  %s\n", resp.DeliveredAt)
			}
			if resp.BouncedAt != "" {
				fmt.Printf("Bounced:    %s\n", resp.BouncedAt)
			}
			if resp.BounceType != "" {
				fmt.Printf("Bounce:     %s\n", resp.BounceType)
			}
			if resp.Detail != "" {
				fmt.Printf("Detail:     %s\n", resp.Detail)
			}
		})
	},
}

// ── templates ─────────────────────────────────────────────────────────────

var mailTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Manage Postmark templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var mailTemplatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Postmark templates registered with nSelf",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonMode, _ := cmd.Flags().GetBool("json")

		client, err := resolveMailClient()
		if err != nil {
			return err
		}
		if client == nil {
			return requireLicense(cmd)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		tmpls, err := client.ListTemplates(ctx)
		if err != nil {
			return mapMailError(err)
		}

		return printResult(jsonMode, map[string]interface{}{"templates": tmpls}, func() {
			if len(tmpls) == 0 {
				fmt.Println("No templates registered.")
				return
			}
			tbl := ui.NewTable("ID", "Name", "Subject", "Provider", "Updated")
			for _, t := range tmpls {
				tbl.AddRow(t.ID, t.Name, t.Subject, t.Provider, t.Updated)
			}
			tbl.Render()
		})
	},
}

// ── dkim ──────────────────────────────────────────────────────────────────

var mailDKIMCmd = &cobra.Command{
	Use:   "dkim",
	Short: "Manage DKIM verification",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var mailDKIMVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify DKIM record present and valid for a domain",
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, _ := cmd.Flags().GetString("domain")
		jsonMode, _ := cmd.Flags().GetBool("json")

		if strings.TrimSpace(domain) == "" {
			return fmt.Errorf("--domain <d> is required")
		}

		client, err := resolveMailClient()
		if err != nil {
			return err
		}
		if client == nil {
			return requireLicense(cmd)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		resp, err := client.VerifyDKIM(ctx, domain)
		if err != nil {
			return mapMailError(err)
		}

		return printResult(jsonMode, resp, func() {
			fmt.Printf("Domain:   %s\n", resp.Domain)
			if resp.Valid {
				ui.Success("DKIM record valid")
			} else {
				ui.Warn("DKIM record invalid or missing")
			}
			if resp.Selector != "" {
				fmt.Printf("Selector: %s\n", resp.Selector)
			}
			if resp.Record != "" {
				fmt.Printf("Record:   %s\n", resp.Record)
			}
			if resp.Detail != "" {
				fmt.Printf("Detail:   %s\n", resp.Detail)
			}
		})
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
