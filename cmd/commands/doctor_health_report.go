package commands

// Purpose: the container-health check plus the report-building and printing
// helpers (buildDoctorReport, printDoctorJSON, printDoctorSummary, printCheck).
// Inputs are collected doctorCheckResult values; outputs are a *doctorReport or
// printed text/JSON.
// Constraints: split out of doctor.go (CLI-R12) as a pure move, no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/ui"
)

// checkContainerHealth inspects running containers and reports their health.
func checkContainerHealth(ctx context.Context, projectDir string, verbose bool) []doctorCheckResult {
	var results []doctorCheckResult
	compose := docker.NewCompose()

	containers, err := compose.ComposePs(ctx, projectDir)
	if err != nil {
		name := "Container health"
		// No compose project here — not necessarily an error
		msg := "no running containers found"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}

	if len(containers) == 0 {
		name := "Container health"
		msg := "no running containers"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}

	for _, c := range containers {
		name := fmt.Sprintf("Container: %s", c.Name)
		status := "pass"
		msg := c.State

		if c.Health != "" && c.Health != "none" {
			msg = fmt.Sprintf("%s (%s)", c.State, c.Health)
		}

		switch {
		case c.State != "running":
			status = "fail"
			msg = fmt.Sprintf("not running (state: %s)", c.State)
		case c.Health == "unhealthy":
			status = "fail"
			msg = "unhealthy"
			// Fetch recent logs for unhealthy containers when verbose
			if verbose {
				logs, logErr := docker.GetContainerLogs(ctx, c.Name, 5)
				if logErr == nil && logs != "" {
					msg += "\n    Recent logs:\n    " + strings.ReplaceAll(strings.TrimSpace(logs), "\n", "\n    ")
				}
			}
		case c.Health == "starting":
			status = "warn"
			msg = "still starting"
		}

		printCheck(status, name, msg, verbose)
		results = append(results, doctorCheckResult{Name: name, Status: status, Message: msg})
	}

	return results
}

// buildDoctorReport aggregates check results into a report with summary counts.
func buildDoctorReport(checks []doctorCheckResult) *doctorReport {
	report := &doctorReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Checks:    checks,
	}
	for _, c := range checks {
		report.Summary.Total++
		switch c.Status {
		case "pass":
			report.Summary.Passed++
		case "warn":
			report.Summary.Warnings++
		case "fail":
			report.Summary.Failed++
		}
	}
	return report
}

// printDoctorJSON renders the full report as JSON to stdout.
func printDoctorJSON(report *doctorReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

// printDoctorSummary prints a final summary box.
func printDoctorSummary(report *doctorReport) {
	fmt.Println()
	ui.Separator()

	items := []string{
		fmt.Sprintf("Total checks: %d", report.Summary.Total),
		fmt.Sprintf("Passed: %d", report.Summary.Passed),
	}
	if report.Summary.Warnings > 0 {
		items = append(items, fmt.Sprintf("Warnings: %d", report.Summary.Warnings))
	}
	if report.Summary.Failed > 0 {
		items = append(items, fmt.Sprintf("Failed: %d", report.Summary.Failed))
	}

	if report.Summary.Failed > 0 {
		ui.Error(fmt.Sprintf("Doctor found %d issue(s) that need attention", report.Summary.Failed))
	} else if report.Summary.Warnings > 0 {
		ui.Warn(fmt.Sprintf("Doctor found %d warning(s)", report.Summary.Warnings))
	} else {
		ui.Success("All checks passed")
	}

	for _, item := range items {
		ui.Bullet(item)
	}
	fmt.Println()
}

// printCheck renders a single check result to the terminal using ui.Checked / ui.Unchecked.
func printCheck(status, name, message string, verbose bool) {
	line := fmt.Sprintf("%s: %s", name, message)
	switch status {
	case "pass":
		ui.Checked(line)
	case "warn":
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.C(ui.Yellow, ui.IconWarning), line)
	case "fail":
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.C(ui.Red, ui.IconFailure), line)
	}
}

// ── New checks: T06 ──────────────────────────────────────────────────────────
