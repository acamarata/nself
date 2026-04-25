package commands

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/maintenance"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// ── Parent command ────────────────────────────────────────────────────────────

var maintenanceCmd = &cobra.Command{
	Use:   "maintenance",
	Short: "Maintenance utilities: disk cleanup, scheduler",
	Long: `Maintenance utilities for nSelf deployments.

Subcommands:
  disk-cleanup   Free Docker images, old logs, and journal entries
  schedule       Install or remove the daily cleanup timer
  status         Show scheduler status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SilenceUsage: true,
}

// ── disk-cleanup ──────────────────────────────────────────────────────────────

var maintenanceDiskCleanupCmd = &cobra.Command{
	Use:   "disk-cleanup",
	Short: "Free disk space: Docker prune, log rotation, journal vacuum",
	Long: `Free disk space by running three cleanup steps:

  1. docker system prune -af --volumes=false
     Removes stopped containers and unused images. Volumes are kept.

  2. find /var/log -name '*.gz' -mtime +14 -delete
     Removes compressed log files older than 14 days.

  3. journalctl --vacuum-time=7d
     Removes journal entries older than 7 days (Linux only; no-op on macOS).

Disk usage is reported before and after the cleanup.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.CommandHeader("nSelf Maintenance", "Disk cleanup")

		ui.Section("Before cleanup")
		before, err := maintenance.GetDiskUsage()
		if err != nil {
			ui.Warn(fmt.Sprintf("Could not read disk usage: %v", err))
		} else {
			printDiskUsage(before)
		}

		ui.Section("Running cleanup steps")

		result := maintenance.DiskCleanup()

		if result.DockerPruneOut != "" {
			ui.Info("Docker prune:")
			for _, line := range splitLines(result.DockerPruneOut) {
				fmt.Printf("  %s\n", line)
			}
		}
		if result.LogRotationOut != "" {
			ui.Info("Log rotation:")
			for _, line := range splitLines(result.LogRotationOut) {
				fmt.Printf("  %s\n", line)
			}
		}
		if result.JournalVacuumOut != "" {
			ui.Info("Journal vacuum:")
			for _, line := range splitLines(result.JournalVacuumOut) {
				fmt.Printf("  %s\n", line)
			}
		}

		if len(result.Errors) > 0 {
			ui.Warn("Some steps completed with warnings:")
			for _, e := range result.Errors {
				ui.Warn(fmt.Sprintf("  %v", e))
			}
		}

		ui.Section("After cleanup")
		after := result.After
		printDiskUsage(after)

		freed := result.Before.UsedGB - after.UsedGB
		if freed > 0.01 {
			ui.Success(fmt.Sprintf("Freed %.2f GB (disk %d%% → %d%% used)",
				freed, result.Before.UsedPercent, after.UsedPercent))
		} else {
			ui.Info(fmt.Sprintf("Disk usage unchanged at %d%%", after.UsedPercent))
		}

		if after.UsedPercent > 70 {
			ui.Warn(fmt.Sprintf(
				"Disk is still %d%% full. Run `nself maintenance schedule --daily` to enable automatic daily cleanup.",
				after.UsedPercent,
			))
		}

		return nil
	},
	SilenceUsage: true,
}

// ── schedule ──────────────────────────────────────────────────────────────────

var maintenanceScheduleDaily bool
var maintenanceScheduleOff bool

var maintenanceScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Install or remove the daily disk-cleanup timer",
	Long: `Install or remove a system-level timer that runs disk-cleanup every day at 03:00 local time.

On Linux, installs a systemd service + timer unit.
On macOS, installs a LaunchDaemon plist.

Flags:
  --daily   Install the daily timer (requires root/sudo)
  --off     Remove the daily timer (requires root/sudo)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !maintenanceScheduleDaily && !maintenanceScheduleOff {
			return cmd.Help()
		}

		if maintenanceScheduleDaily && maintenanceScheduleOff {
			return fmt.Errorf("specify either --daily or --off, not both")
		}

		if maintenanceScheduleDaily {
			ui.CommandHeader("nSelf Maintenance", "Installing daily timer")
			if err := maintenance.InstallDailyTimer(); err != nil {
				return fmt.Errorf("install daily timer: %w", err)
			}
			ui.Success("Daily disk-cleanup timer installed (runs at 03:00 local time).")
			ui.Info("Use `nself maintenance status` to verify.")
			return nil
		}

		// --off
		ui.CommandHeader("nSelf Maintenance", "Removing daily timer")
		if err := maintenance.RemoveDailyTimer(); err != nil {
			return fmt.Errorf("remove daily timer: %w", err)
		}
		ui.Success("Daily disk-cleanup timer removed.")
		return nil
	},
	SilenceUsage: true,
}

// ── status ────────────────────────────────────────────────────────────────────

var maintenanceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daily cleanup scheduler status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.CommandHeader("nSelf Maintenance", "Scheduler status")

		status := maintenance.GetScheduleStatus()
		fmt.Printf("  Platform : %s\n", status.Platform)
		if status.Enabled {
			ui.Success("Daily cleanup timer is enabled")
		} else {
			ui.Warn("Daily cleanup timer is NOT enabled")
			ui.Info("Run `nself maintenance schedule --daily` to enable it.")
		}
		if status.Detail != "" {
			fmt.Printf("  Detail   : %s\n", status.Detail)
		}

		fmt.Println()
		usage, err := maintenance.GetDiskUsage()
		if err == nil {
			ui.Section("Current disk usage")
			printDiskUsage(usage)
			if usage.UsedPercent > 70 {
				ui.Warn(fmt.Sprintf(
					"Disk is %d%% full. Run `nself maintenance schedule --daily` to enable automatic daily cleanup.",
					usage.UsedPercent,
				))
			}
		}
		return nil
	},
	SilenceUsage: true,
}

// ── helpers ───────────────────────────────────────────────────────────────────

func printDiskUsage(u maintenance.DiskUsage) {
	bar := diskBar(u.UsedPercent, 30)
	fmt.Printf("  Used: %5.1f GB / %.1f GB total  [%s] %d%%\n",
		u.UsedGB, u.TotalGB, bar, u.UsedPercent)
	fmt.Printf("  Free: %.1f GB\n", u.FreeGB)
}

func diskBar(pct, width int) string {
	filled := (pct * width) / 100
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// ── registration ──────────────────────────────────────────────────────────────

func init() {
	maintenanceScheduleCmd.Flags().BoolVar(&maintenanceScheduleDaily, "daily", false, "Install daily cleanup timer at 03:00 local time")
	maintenanceScheduleCmd.Flags().BoolVar(&maintenanceScheduleOff, "off", false, "Remove the daily cleanup timer")

	maintenanceCmd.AddCommand(maintenanceDiskCleanupCmd)
	maintenanceCmd.AddCommand(maintenanceScheduleCmd)
	maintenanceCmd.AddCommand(maintenanceStatusCmd)
	RootCmd.AddCommand(maintenanceCmd)
}
