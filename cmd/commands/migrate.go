package commands

import (
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/migration"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Detect and migrate v1 artifacts to nSelf v2",
	Long: `Scan the current project directory for nSelf v1 artifacts and report
what needs to be updated for v2 compatibility.

Examples:
  nself migrate          # Scan current directory for v1 artifacts
  nself migrate detect   # Alias for the default scan
  nself migrate run      # Migrate v1 project to v2
  nself migrate rollback # Restore from the most recent backup`,
	RunE: runMigrate,
}

var migrateDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect v1 artifacts in the current project",
	RunE:  runMigrate,
}

var migrateRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Migrate v1 project to v2",
	Long: `Perform a full v1 → v2 migration.

Steps:
  1. Detect v1 artifacts (idempotent — exits cleanly if already v2)
  2. Stop running containers gracefully
  3. Backup current state to .nself/backup/{timestamp}/
  4. Move nginx configs to v2 layout (nginx/sites/)
  5. Regenerate docker-compose.yml, nginx, and SSL certificates
  6. Print migration summary

On any failure, run 'nself migrate rollback' to restore the previous state.`,
	RunE: runMigrateRun,
}

var migrateRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Restore a v1 backup created by migrate run",
	Long: `Restore the project from the most recent migration backup.

Use --backup to select a specific backup by timestamp.
Use --list to see all available backups.`,
	RunE: runMigrateRollback,
}

func init() {
	migrateRollbackCmd.Flags().String("backup", "", "Specific backup timestamp to restore (e.g. 20260328-143022)")
	migrateRollbackCmd.Flags().Bool("list", false, "List available backups with timestamps and sizes")

	migrateCmd.AddCommand(migrateDetectCmd)
	migrateCmd.AddCommand(migrateRunCmd)
	migrateCmd.AddCommand(migrateRollbackCmd)
	RootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	artifacts := migration.Detect(cwd)

	if len(artifacts) == 0 {
		ui.Success("No v1 artifacts detected. This project is ready for nSelf v2.")
		return nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\n%s %s\n\n", ui.C(ui.Yellow, ui.IconWarning), ui.C(ui.Bold, "Legacy nSelf v1 artifacts detected:"))

	tbl := ui.NewTable("Path", "Description", "Action")
	for _, a := range artifacts {
		tbl.AddRow(a.Path, a.Description, a.Action)
	}
	tbl.Render()

	fmt.Println()
	ui.Warn(fmt.Sprintf("%d v1 artifact(s) found. Review the table above and migrate each item.", len(artifacts)))

	return nil
}

func runMigrateRun(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	return migration.Run(cmd.Context(), cwd)
}

func runMigrateRollback(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	listOnly, _ := cmd.Flags().GetBool("list")
	if listOnly {
		return printBackupList(cwd)
	}

	backupTimestamp, _ := cmd.Flags().GetString("backup")
	return migration.Rollback(cmd.Context(), cwd, backupTimestamp)
}

// printBackupList prints all available backups with timestamps and sizes.
func printBackupList(projectDir string) error {
	backups, err := migration.ListBackups(projectDir)
	if err != nil {
		return fmt.Errorf("listing backups: %w", err)
	}

	if len(backups) == 0 {
		ui.Warn("No backups found in .nself/backup/")
		return nil
	}

	fmt.Printf("\n%s %s\n\n", ui.C(ui.Blue, ui.IconInfo), ui.C(ui.Bold, "Available migration backups:"))

	tbl := ui.NewTable("Timestamp", "Size", "Path")
	for _, b := range backups {
		tbl.AddRow(b.Timestamp, formatBytes(b.Size), b.Dir)
	}
	tbl.Render()

	fmt.Println()
	ui.Info("Restore a backup: nself migrate rollback --backup <timestamp>")
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
