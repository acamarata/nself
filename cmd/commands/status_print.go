package commands

// Purpose: Rendering helpers for `nself status` — JSON output, the
// terminal status table, a Docker-error message wrapper, and the
// suggested-next-command hints printed after an unhealthy report. Split
// out of status.go (CLI-R12) to separate presentation from the
// statusCmd cobra wiring and the per-service status check
// (runSingleServiceStatus) that remain in status.go.
// Inputs: a *health.HealthReport plus verbose/healthOnly/metrics flags,
// or (wrapDockerError) a raw error and container name.
// Outputs: printed JSON or table output, or a wrapped, more actionable
// error.
// Constraints: pure move — no behavior changes.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/health"
	"github.com/nself-org/cli/internal/ui"
)

// printStatusJSON renders the health report as JSON.
func printStatusJSON(report *health.HealthReport) error {
	out := statusJSONOutput{
		Timestamp: report.Timestamp.Format(time.RFC3339),
		Services:  make([]statusJSONService, 0, len(report.Results)),
		Summary: statusJSONSummary{
			Total:     report.Total,
			Healthy:   report.Healthy,
			Unhealthy: report.Unhealthy,
		},
	}

	for _, r := range report.Results {
		out.Services = append(out.Services, statusJSONService{
			Name:     r.Service,
			Status:   r.Status,
			Duration: r.Duration.Round(time.Millisecond).String(),
			Details:  r.Details,
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

// printStatusTable renders the health report as a table.
func printStatusTable(report *health.HealthReport, verbose, healthOnly, metrics bool) {
	headers := []string{"Service", "Status", "Details"}
	if verbose || metrics {
		headers = append(headers, "Duration")
	}

	tbl := ui.NewTable(headers...)

	// Same predicate the report's Healthy/Unhealthy counts use
	// (health.HealthResult.OK) — see issue #268.
	for _, r := range report.Results {
		var icon, statusText string
		if r.OK() {
			icon = ui.IconSuccess
			statusText = icon + " healthy"
		} else {
			icon = ui.IconFailure
			statusText = icon + " " + r.Status
		}

		if healthOnly {
			// health-only: just service + status
			row := []string{r.Service, statusText}
			if verbose || metrics {
				row = append(row, "", r.Duration.Round(time.Millisecond).String())
			} else {
				row = append(row, "")
			}
			tbl.AddRow(row...)
			continue
		}

		row := []string{r.Service, statusText, r.Details}
		if verbose || metrics {
			row = append(row, r.Duration.Round(time.Millisecond).String())
		}
		tbl.AddRow(row...)
	}

	tbl.Render()

	// Summary line.
	fmt.Println()
	fmt.Printf("Summary: %d/%d services healthy\n", report.Healthy, report.Total)
}

// wrapDockerError converts raw Docker errors into user-friendly messages.
// It checks for well-known conditions (daemon not running, container missing)
// and falls back to a generic context-wrapped error otherwise.
func wrapDockerError(err error, containerName string) error {
	if errors.Is(err, errs.ErrServiceNotFound) {
		return fmt.Errorf("container %s is not running — try: nself start", containerName)
	}
	msg := err.Error()
	if strings.Contains(msg, "docker is not running") ||
		strings.Contains(msg, "Cannot connect to the Docker daemon") {
		return fmt.Errorf("docker is not running — start Docker Desktop or run: systemctl start docker")
	}
	if strings.Contains(msg, "No such container") || strings.Contains(msg, "not found") {
		return fmt.Errorf("container %s is not running — try: nself start", containerName)
	}
	return fmt.Errorf("failed to inspect container %s: %w", containerName, err)
}

// printStatusSuggestions prints state-aware next-step hints based on the
// health report. All-healthy gets a positive nudge; partial/unhealthy gets
// targeted fix commands.
func printStatusSuggestions(report *health.HealthReport) {
	if report.Unhealthy == 0 && report.Healthy == report.Total {
		// All healthy — encourage next action.
		fmt.Println()
		fmt.Printf("  %s All services healthy. Try: %s\n",
			ui.C(ui.Green, ui.IconSuccess),
			ui.C(ui.Cyan, "nself plugin list"),
		)
		return
	}

	// Collect names of unhealthy services.
	var unhealthyNames []string
	for _, r := range report.Results {
		if !r.OK() {
			unhealthyNames = append(unhealthyNames, r.Service)
		}
	}
	if len(unhealthyNames) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("  %s %d service(s) not healthy: %s\n",
		ui.C(ui.Yellow, ui.IconWarning),
		len(unhealthyNames),
		strings.Join(unhealthyNames, ", "),
	)
	fmt.Printf("  %s Run %s for diagnostics\n",
		ui.C(ui.Blue, ui.IconArrow),
		ui.C(ui.Cyan, "nself doctor"),
	)
	fmt.Printf("  %s Run %s to see service logs\n",
		ui.C(ui.Blue, ui.IconArrow),
		ui.C(ui.Cyan, "nself logs "+unhealthyNames[0]),
	)
}
