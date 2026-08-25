package commands

// Purpose: the "nself backup resume" and "nself backup schedule" subcommands,
// their RunE, and the validateCronExpression helper. Inputs are the cobra
// command/args; outputs are a resumed backup or a scheduled cron entry, or an
// error.
// Constraints: split out of backup_ops.go (CLI-R12) as a pure move, no behavior change.

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/backup"
	"github.com/spf13/cobra"
)

var backupResumeCmd = &cobra.Command{
	Use:   "resume <backup-id>",
	Short: "Resume an interrupted streaming backup",
	Long: `Resume a previously interrupted streaming backup.

Since rclone rcat uploads are not resumable at the protocol level, resume
re-streams the full backup from pg_dump and overwrites the partial remote
object. Resume state is stored in ~/.nself/backup-state/.`,
	Args: cobra.ExactArgs(1),
	RunE: runBackupResume,
}

func runBackupResume(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	result, err := backup.Resume(cmd.Context(), cfg, backup.ResumeOptions{
		BackupID: args[0],
	})
	if err != nil {
		return fmt.Errorf("backup resume: %w", err)
	}

	fmt.Printf("Resume complete.\n")
	fmt.Printf("  Destination: %s\n", result.Destination)
	fmt.Printf("  Duration:    %s\n", result.Duration)
	return nil
}

// ── backup schedule ────────────────────────────────────────────────

var backupScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Schedule recurring streaming backups via systemd timers",
	Long: `Install a systemd timer to run 'nself backup stream' on a cron schedule.

Examples:

  nself backup schedule --cron "0 2 * * *" --to s3:bucket/path
  nself backup schedule --cron "0 2 * * *" --to r2:bucket --recipient age1xxx --dry-run`,
	RunE: runBackupSchedule,
}

// validateCronExpression checks that a cron expression has exactly 5 whitespace-separated
// fields, each field being a non-empty token.  This is a structural pre-check; the
// systemd OnCalendar translation in internal/backup/stream.go handles semantic
// validation.  An empty or malformed expression always returns a non-nil error so
// callers can surface exit code 1 before spawning any systemd units.
func validateCronExpression(expr string) error {
	if expr == "" {
		return fmt.Errorf("--cron expression is required (e.g. '0 2 * * *')")
	}
	// Count fields by splitting on whitespace.
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("invalid cron expression %q: expected 5 fields (minute hour dom month dow), got %d", expr, len(fields))
	}
	return nil
}

func runBackupSchedule(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	cron, _ := cmd.Flags().GetString("cron")
	to, _ := cmd.Flags().GetString("to")
	recipient, _ := cmd.Flags().GetString("recipient")
	unitDir, _ := cmd.Flags().GetString("unit-dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateCronExpression(cron); err != nil {
		return err
	}

	if err := backup.ScheduleStream(cfg, cron, to, recipient, unitDir, dryRun); err != nil {
		return fmt.Errorf("backup schedule: %w", err)
	}

	if !dryRun {
		fmt.Printf("Streaming backup scheduled (cron: %s, destination: %s).\n", cron, to)
		fmt.Println("Run 'nself backup status' to see next scheduled run.")
	}
	return nil
}

// ── backup restore-remote ──────────────────────────────────────────
