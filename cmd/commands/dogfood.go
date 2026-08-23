package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/dogfood"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"

	"github.com/nself-org/cli/internal/errs"
)

var dogfoodCmd = &cobra.Command{
	Use:   "dogfood",
	Short: "Production dogfood audit and reporting",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var dogfoodAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Run comprehensive dogfood audit (21 checks)",
	Long: `Run the complete dogfood audit checklist against your production environment.

Verifies backups, DR, tenancy, licensing, secrets, migrations, monitoring,
security, watchdog, queue health, and more.

Exit codes:
  0  All checks passed
  1  One or more failures
  2  Warnings only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		save, _ := cmd.Flags().GetBool("save")

		cwd, _ := os.Getwd()
		ctx := cmd.Context()

		report := dogfood.RunAudit(ctx, cwd)

		if save {
			if err := dogfood.SaveReport(cwd, report); err != nil {
				ui.Warn(fmt.Sprintf("Failed to save report: %v", err))
			} else {
				ui.Bullet(fmt.Sprintf("Report saved to .nself/dogfood/"))
			}
		}

		if jsonOut {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
		} else {
			ui.CommandHeader("Dogfood Audit", fmt.Sprintf("%d checks", len(report.Checks)))

			currentSection := ""
			for _, c := range report.Checks {
				if c.Section != currentSection {
					currentSection = c.Section
					ui.Section(currentSection)
				}
				printCheck(c.Status, c.Name, c.Message, false)
			}

			fmt.Println()
			ui.Separator()
			ui.Bullet(fmt.Sprintf("Pass: %d  Fail: %d  Warn: %d  Skip: %d",
				report.PassCount, report.FailCount, report.WarnCount, report.SkipCount))
		}

		// The report above is the output; exit status alone conveys the verdict.
		if report.FailCount > 0 {
			return errs.Exit(1)
		}
		if report.WarnCount > 0 {
			return errs.Exit(2)
		}
		return nil
	},
}

var dogfoodReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show the latest dogfood audit report",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		cwd, _ := os.Getwd()

		report, err := dogfood.GetLatestReport(cwd)
		if err != nil {
			return fmt.Errorf("no reports found: %w", err)
		}

		if jsonOut {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		ui.CommandHeader("Latest Dogfood Report", report.RanAt.Format("2006-01-02 15:04:05"))
		ui.Bullet(fmt.Sprintf("Pass: %d  Fail: %d  Warn: %d  Skip: %d",
			report.PassCount, report.FailCount, report.WarnCount, report.SkipCount))

		for _, c := range report.Checks {
			if c.Status != "pass" {
				printCheck(c.Status, c.Name, c.Message, false)
			}
		}

		return nil
	},
}

func init() {
	dogfoodAuditCmd.Flags().Bool("json", false, "JSON output")
	dogfoodAuditCmd.Flags().Bool("save", true, "Save report to .nself/dogfood/")
	dogfoodReportCmd.Flags().Bool("json", false, "JSON output")

	dogfoodCmd.AddCommand(dogfoodAuditCmd, dogfoodReportCmd)
	RootCmd.AddCommand(dogfoodCmd)
}
