// S9.T03 — `nself backup drill` cobra wiring.
//
// This file adds a single subcommand under the existing `backupCmd` parent
// (defined in backup_ops.go). The command is intentionally thin — all
// orchestration lives in internal/backup.Drill so the same logic powers the
// scripts/dr-drill.sh weekly cron + future Admin UI buttons.
package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nself-org/cli/internal/database"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var backupDrillCmd = &cobra.Command{
	Use:   "drill",
	Short: "Run a DR drill: restore latest backup into scratch DB and measure RTO",
	Long: `Run a disaster-recovery drill.

The drill restores the most recent backup (or the file passed via --file) into
a temporary scratch database, runs smoke checks against the critical np_*
tables (np_users, np_licenses, np_audit_log, np_plugins, np_billing), measures
total wall-clock RTO, and writes the structured result to
.nself/drill-log.json. Production data is never touched.

Default RTO target is 4 hours per STRAT-11; failures past that flip
result.RTOTargetMet to false but do not abort the drill. Pass --rto-hours 0 to
disable the gate entirely (the drill still records its duration).

The doctor check OPS-DRILL-01 reads the drill log to enforce "drill within
last 7 days"; pair this command with a weekly cron via scripts/dr-drill.sh.`,
	RunE: runBackupDrill,
}

func runBackupDrill(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	file, _ := cmd.Flags().GetString("file")
	rto, _ := cmd.Flags().GetFloat64("rto-hours")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	jsonOut, _ := cmd.Flags().GetBool("json")

	opts := database.DrillOptions{
		BackupFile:     file,
		RTOTargetHours: rto,
		DryRun:         dryRun,
		ProjectDir:     ".",
	}

	result, err := database.Drill(cmd.Context(), cfg, opts)
	if jsonOut {
		// Always emit JSON when --json is set, even on error, so cron parsers
		// see the structured outcome (the err itself is the same string as
		// result.ErrorMessage in failure paths).
		data, mErr := json.MarshalIndent(result, "", "  ")
		if mErr == nil {
			fmt.Println(string(data))
		}
		return err
	}
	if err != nil {
		return fmt.Errorf("backup drill: %w", err)
	}

	verdict := "PASS"
	if !result.RTOTargetMet {
		verdict = "PASS (over RTO target)"
	}
	ui.Info(fmt.Sprintf("Backup drill: %s", verdict))
	ui.Dimmed(fmt.Sprintf("  Backup file:    %s", result.BackupFile))
	ui.Dimmed(fmt.Sprintf("  Duration:       %.1fs (restore=%.1fs verify=%.1fs smoke=%.1fs)",
		result.TotalDuration.Seconds(), result.RestoreSeconds, result.VerifySeconds, result.SmokeSeconds))
	ui.Dimmed(fmt.Sprintf("  Tables checked: %d", result.TablesChecked))
	ui.Dimmed(fmt.Sprintf("  Rows observed:  %d", result.RowsObserved))
	ui.Dimmed(fmt.Sprintf("  RTO target met: %v", result.RTOTargetMet))
	return nil
}

func init() {
	backupDrillCmd.Flags().String("file", "", "Backup file to drill against (default: most recent)")
	backupDrillCmd.Flags().Float64("rto-hours", 4.0, "RTO target in hours; 0 disables the gate")
	backupDrillCmd.Flags().Bool("dry-run", false, "Validate inputs and exit without restoring")
	backupDrillCmd.Flags().Bool("json", false, "Emit DrillResult as JSON to stdout")

	backupCmd.AddCommand(backupDrillCmd)
}
