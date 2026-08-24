package commands

// Purpose: `nself mail send/broadcast/status` split out of mail.go
// (CLI-R12 Batch B mechanical file-size split). The three subcommands that
// send or query a single transactional/broadcast message.
// Inputs: cobra command flags as registered in mail.go's init() (--to,
// --subject, --body, --body-file, --body-type, --list, --template,
// --message-id, --json).
// Outputs: JSON or human-readable send/broadcast/status results; errors
// via mapMailError (mail.go) or requireLicense (mail.go) when no license
// key is configured.
// Constraints: pure move, no behavior change. mailCmd (parent),
// resolveMailClient, requireLicense, printResult, and mapMailError remain
// in mail.go.

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/mail"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

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
