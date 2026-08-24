package commands

// Purpose: `nself mail templates/dkim` split out of mail.go (CLI-R12
// Batch B mechanical file-size split). Read-only/admin subcommands: list
// registered Postmark templates and verify a domain's DKIM record.
// Inputs: cobra command flags as registered in mail.go's init() (--domain,
// --json).
// Outputs: a table or JSON listing of templates, or a DKIM verification
// report; errors via mapMailError/requireLicense (mail.go).
// Constraints: pure move, no behavior change. mailCmd (parent),
// resolveMailClient, requireLicense, printResult, and mapMailError remain
// in mail.go.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

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
