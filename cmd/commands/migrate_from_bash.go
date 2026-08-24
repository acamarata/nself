package commands

// Purpose: `nself migrate from-bash` — detects v0.9.9 Bash-era artifacts, prints the guided migration checklist, and runs automatable steps under --auto.
// Inputs: cobra flags (--auto) for the migrate command group defined in migrate.go.
// Outputs: a printed migration checklist and, under --auto, applied automatic migration steps.
// Constraints: split out of migrate.go as a pure move (CLI-R12); no behavior change.

import (
	"fmt"
	"os"

	migratebash "github.com/nself-org/cli/internal/migrate"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// runMigrateFromBash handles `nself migrate from-bash` (and the top-level
// `nself migrate --from-bash` alias). It detects v0.9.9 Bash-era artifacts,
// prints a guided migration checklist, and — when --auto is set — runs the
// automatable steps after confirmation.
func runMigrateFromBash(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	autoRun, _ := cmd.Flags().GetBool("auto")
	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if skipConfirm {
		autoRun = true
	}

	// Detect v0.9.9 Bash-era artifacts.
	result := migratebash.DetectBashEra(cwd)

	if result.AlreadyMigrated {
		ui.Success("No v0.9.9 Bash-era artifacts detected. Nothing to migrate.")
		ui.Info("If you expected artifacts, verify you are in the project root directory.")
		return nil
	}

	// Print detected artifacts.
	fmt.Printf("\n%s %s\n\n",
		ui.C(ui.Yellow, ui.IconWarning),
		ui.C(ui.Bold, fmt.Sprintf("%d v0.9.9 Bash-era artifact(s) detected:", len(result.Artifacts))),
	)

	tbl := ui.NewTable("Path", "Description", "Auto-migrate?")
	for _, a := range result.Artifacts {
		auto := "manual"
		if a.AutoMigratable {
			auto = "yes"
		}
		tbl.AddRow(a.Path, a.Description, auto)
	}
	tbl.Render()

	// Print the full migration checklist.
	fmt.Printf("\n%s %s\n\n",
		ui.C(ui.Blue, ui.IconInfo),
		ui.C(ui.Bold, "Migration checklist (v0.9.9 → current):"),
	)

	steps := migratebash.MigrationChecklist(cwd)
	for _, step := range steps {
		p98tag := ""
		if step.P98Scope {
			p98tag = " [P98 scope — coming later]"
		}
		fmt.Printf("  %d. %s%s\n", step.Number, step.Title, p98tag)
		for _, c := range step.Commands {
			fmt.Printf("       %s\n", c)
		}
		if step.Notes != "" {
			fmt.Printf("     Note: %s\n", step.Notes)
		}
		fmt.Println()
	}

	fmt.Printf("  Full guide: https://github.com/nself-org/cli/wiki/Upgrading-from-v0.9.9\n\n")

	if !autoRun {
		ui.Info("To run automatable steps (backup + remove nself.sh), add --auto flag.")
		ui.Info("Review the checklist above and complete manual steps at your own pace.")
		return nil
	}

	// Confirm before running automated steps (unless --yes was set).
	if !skipConfirm {
		fmt.Print("Run automatable migration steps now? [y/N] ")
		var answer string
		if _, scanErr := fmt.Scanln(&answer); scanErr != nil || (answer != "y" && answer != "Y") {
			ui.Info("Aborted. No changes made.")
			return nil
		}
	}

	autoResult, err := migratebash.RunAuto(cwd, result)
	if err != nil {
		return fmt.Errorf("running automated migration steps: %w", err)
	}

	if autoResult.BackupDir != "" {
		ui.Success(fmt.Sprintf("Backup created: %s", autoResult.BackupDir))
	}
	for _, action := range autoResult.ActionsPerformed {
		ui.Success(fmt.Sprintf("Done: %s", action))
	}
	for _, skip := range autoResult.Skipped {
		ui.Info(fmt.Sprintf("Manual step required: %s", skip))
	}

	fmt.Println()
	ui.Info("Automated steps complete. Follow the remaining manual steps in the checklist above.")
	ui.Info("Next: review .envrc / docker-compose.yml, populate .env.dev, then run: nself build")

	return nil
}

// formatBytes returns a human-readable size string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
