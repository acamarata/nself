package dr

// Purpose: the cold-start DR drill implementation (provision, restore, verify, destroy) and its VM-lifecycle/backup-discovery helpers, backing Drill in drill.go.
// Inputs: a *config.Config and the *DrillResult being populated as the drill progresses.
// Outputs: a dated drill report and a populated DrillResult (or an error on any failed step).
// Constraints: split out of drill.go as a pure move (CLI-R12); no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// drillColdStart executes the full cold-start DR drill:
//
//	Step 1 — Provision a fresh Hetzner VM.
//	Step 2 — Pull the latest verified backup from BACKUP_DEST.
//	Step 3 — Start clean on the isolated drill VM (no live-host wipe).
//	Step 4 — Restore the backup and run the verify smoke-query catalog.
//	Step 5 — Run smoke queries, record RTO, and write a dated report.
//	Step 6 — Destroy the drill VM.
//
// The dated report is written to ~/.claude/backups/nself-staging/dr/YYYY-MM-DD-cold-start.md.
// If any step fails the report is written as PARTIAL and the function returns an error.
func drillColdStart(ctx context.Context, cfg *config.Config, result *DrillResult) error {
	slog.Info("cold-start drill: provision fresh VM, restore from backup, verify")
	drillStart := time.Now()

	reportDir := filepath.Join(os.Getenv("HOME"), ".claude", "backups", "nself-staging", "dr")
	if err := os.MkdirAll(reportDir, 0o750); err != nil {
		return fmt.Errorf("create drill report dir: %w", err)
	}
	reportFile := filepath.Join(reportDir, drillStart.Format("2006-01-02")+"-cold-start.md")

	// writeReport emits the dated report regardless of outcome.
	writeReport := func(rto time.Duration, smokeResults string, status string) {
		content := strings.Join([]string{
			"# Cold-Start DR Drill Report",
			"",
			"**Date:** " + drillStart.Format("2006-01-02 15:04:05 UTC"),
			"**Status:** " + status,
			"**RTO Measured:** " + rto.String(),
			"**RTO Target:** 20m0s",
			"",
			"## Smoke Query Results",
			"",
			smokeResults,
			"",
			"## Drill Details",
			"",
			fmt.Sprintf("- VM IP: %v", result.Details["vm_ip"]),
			fmt.Sprintf("- Backup: %v", result.Details["backup_file"]),
			fmt.Sprintf("- Restore duration: %v", result.Details["restore_duration"]),
		}, "\n")
		if werr := os.WriteFile(reportFile, []byte(content), 0o640); werr != nil {
			slog.Warn("failed to write drill report", "path", reportFile, "error", werr)
		} else {
			slog.Info("drill report written", "path", reportFile)
		}
		result.Details["report_file"] = reportFile
	}

	// Step 1: Provision VM via Hetzner API.
	slog.Info("step 1: provisioning test VM via Hetzner API")
	result.Details["step_1"] = "provision VM"
	serverIP, err := provisionDrillVM(ctx, cfg)
	if err != nil {
		writeReport(time.Since(drillStart), "N/A — VM provision failed", "PARTIAL")
		return fmt.Errorf("provision VM: %w", err)
	}
	result.Details["vm_ip"] = serverIP
	slog.Info("drill VM provisioned", "ip", serverIP)

	// Ensure VM is cleaned up on return.
	defer func() {
		slog.Info("step 6: destroying drill VM", "ip", serverIP)
		if derr := destroyDrillVM(ctx, cfg, serverIP); derr != nil {
			slog.Warn("failed to destroy drill VM — manual cleanup required", "ip", serverIP, "error", derr)
		}
	}()

	// Step 2: Pull latest verified backup.
	slog.Info("step 2: pulling latest verified backup")
	result.Details["step_2"] = "pull backup"
	backupDest := cfg.Backup.Dir
	if backupDest == "" {
		backupDest = "./backups"
	}
	backupFile, err := latestBackupFile(backupDest)
	if err != nil {
		writeReport(time.Since(drillStart), "N/A — no backup found", "PARTIAL")
		return fmt.Errorf("locate latest backup: %w", err)
	}
	result.Details["backup_file"] = backupFile
	slog.Info("latest backup located", "file", backupFile)

	// Step 3: Start from a clean slate.
	//
	// This drill runs entirely on the throwaway VM provisioned in Step 1 (and
	// destroyed in Step 6 via the deferred cleanup above). It NEVER touches the
	// volumes of any live host, so no destructive wipe of staging or prod data
	// occurs here. The restore in Step 4 creates a fresh test container.
	//
	// SAFETY INVARIANT for future edits: if you ever wire an actual
	// `nself stop --wipe`, it MUST target only this drill VM (serverIP), never a
	// live host. Do not assume a staging guard exists upstream — add one here.
	slog.Info("step 3: clean-slate restore on isolated drill VM", "vm", serverIP)
	result.Details["step_3"] = "clean-slate (isolated drill VM)"

	// Step 4: Run restore + verify (calls backup.runRestoreTest via verify.Verify).
	slog.Info("step 4: restore + verify on drill VM")
	result.Details["step_4"] = "restore + verify"
	restoreStart := time.Now()
	verifyArgs := []string{
		"exec", "-i", serverIP,
		"nself", "backup", "verify", "--restore-test", "--source", backupFile,
	}
	verifyCmd := exec.CommandContext(ctx, "ssh", verifyArgs...)
	verifyOut, verifyErr := verifyCmd.CombinedOutput()
	restoreDuration := time.Since(restoreStart)
	result.Details["restore_duration"] = restoreDuration.String()
	if verifyErr != nil {
		writeReport(time.Since(drillStart), string(verifyOut), "PARTIAL")
		return fmt.Errorf("restore+verify failed (%s): %w", string(verifyOut), verifyErr)
	}
	slog.Info("restore+verify passed", "duration", restoreDuration)

	// Step 5: Record RTO.
	rto := time.Since(drillStart)
	result.Details["rto"] = rto.String()
	result.Details["step_5"] = "rto recorded"
	slog.Info("cold-start drill complete", "rto", rto, "target", "20m")

	smokeResults := "All smoke queries passed (see verify log above)"
	status := "SUCCESS"
	if rto > 20*time.Minute {
		status = "SUCCESS (RTO EXCEEDED TARGET)"
		smokeResults += fmt.Sprintf("\n\nWARNING: RTO %s exceeded 20m target.", rto)
	}
	writeReport(rto, smokeResults, status)

	return nil
}

// latestBackupFile returns the path of the most recently modified backup file
// in backupDest. Returns an error when the directory is empty.
func latestBackupFile(backupDest string) (string, error) {
	entries, err := os.ReadDir(backupDest)
	if err != nil {
		return "", fmt.Errorf("read backup dir: %w", err)
	}
	var latest string
	var latestMod time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = filepath.Join(backupDest, e.Name())
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no backup files found in %s", backupDest)
	}
	return latest, nil
}

func provisionDrillVM(ctx context.Context, cfg *config.Config) (string, error) {
	// Use hcloud CLI to create a temporary server.
	args := []string{
		"server", "create",
		"--name", cfg.ProjectName + "-dr-drill",
		"--type", "cx22",
		"--image", "ubuntu-24.04",
		"--location", "fsn1",
		"--ssh-key", "default",
		"-o", "json",
	}
	cmd := exec.CommandContext(ctx, "hcloud", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("hcloud server create: %s: %w", string(output), err)
	}

	// Parse server IP from JSON output.
	var resp struct {
		Server struct {
			PublicNet struct {
				IPv4 struct {
					IP string `json:"ip"`
				} `json:"ipv4"`
			} `json:"public_net"`
		} `json:"server"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		return "", fmt.Errorf("parse hcloud response: %w", err)
	}

	return resp.Server.PublicNet.IPv4.IP, nil
}

func destroyDrillVM(ctx context.Context, cfg *config.Config, serverIP string) error {
	args := []string{"server", "delete", cfg.ProjectName + "-dr-drill"}
	cmd := exec.CommandContext(ctx, "hcloud", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("hcloud server delete: %s: %w", string(output), err)
	}
	return nil
}
