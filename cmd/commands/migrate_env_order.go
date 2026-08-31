package commands

// migrate_env_order.go — `nself migrate` output for the CLI-R18 env cascade
// migration shim.
//
// Purpose: Render an internal/migrate.EnvOrderReport (produced by
//          runMigrate in migrate.go) as a human-readable summary: what was
//          auto-fixed, what needs manual review, and a plain "no change
//          needed" line when the project already resolves identically under
//          both cascade orders. Split out of migrate.go to keep that file
//          under the 300-line cap.
// Inputs:  the cobra command (for stderr) and the report.
// Outputs: stdout/stderr only — no files written here (that's Apply, in
//          internal/migrate/env_order_apply.go).
// Constraints: Pure presentation — must not re-derive or second-guess the
//              report's Action/FixedFile decisions.
// SPORT:   cli/cmd/commands — CLI-R18.

import (
	"fmt"

	migratebash "github.com/nself-org/cli/internal/migrate"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// printEnvOrderSummary prints the result of the CLI-R18 env cascade
// migration shim: what was auto-fixed, what needs manual review, and a
// "no change needed" line when the project already resolves identically
// under both cascade orders.
func printEnvOrderSummary(cmd *cobra.Command, report *migratebash.EnvOrderReport) {
	fmt.Println()
	if report.NoChangeNeeded {
		ui.Success("env cascade order: no change needed — this project resolves identically under the old and new order.")
		return
	}

	ui.Info(fmt.Sprintf("env cascade order: %d variable(s) affected by the CLI-R18 reorder.", len(report.Changes)))
	fmt.Println()

	if fixed := report.FixedCount(); fixed > 0 {
		tbl := ui.NewTable("Env", "Variable", "Before (legacy order)", "After (fixed)")
		for _, c := range report.Changes {
			if c.Action != migratebash.ActionFixed {
				continue
			}
			tbl.AddRow(c.EnvName, c.Var, fmt.Sprintf("%s=%s", c.OldWinner, c.OldValue), fmt.Sprintf("%s=%s", c.FixedFile, c.OldValue))
		}
		tbl.Render()
		ui.Success(fmt.Sprintf("%d variable(s) auto-fixed: the pre-migration effective value was written into .env.secrets so it keeps winning under the new cascade order.", fixed))
		fmt.Println()
	}

	if manual := report.ManualReviewCount(); manual > 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n\n", ui.C(ui.Yellow, ui.IconWarning), ui.C(ui.Bold, fmt.Sprintf("%d variable(s) need your review:", manual)))
		for _, c := range report.Changes {
			if c.Action != migratebash.ActionManualReview {
				continue
			}
			fmt.Printf("  %s [%s]: %s\n", ui.C(ui.Bold, c.Var), c.EnvName, c.Reason)
		}
		fmt.Println()
	}

	if report.AIArchived {
		ui.Info(".env.ai content folded into .env.secrets; the old file was renamed to .env.ai.migrated.")
	}

	ui.Info("Run 'nself env explain [VAR]' to see the effective cascade for any variable.")
}
