package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/health"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// statusJSONOutput is the JSON schema for --json output.
type statusJSONOutput struct {
	Timestamp string              `json:"timestamp"`
	Services  []statusJSONService `json:"services"`
	Summary   statusJSONSummary   `json:"summary"`
}

type statusJSONService struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Details  string `json:"details"`
}

type statusJSONSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
}

var statusCmd = &cobra.Command{
	Use:   "status [SERVICE]",
	Short: "Show health status of all services",
	Long: `Show health status of all nSelf services, or a single service if specified.

Exit codes:
  0  All services healthy
  1  Error running checks
  2  One or more services unhealthy`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		verbose, _ := cmd.Flags().GetBool("verbose")
		healthOnly, _ := cmd.Flags().GetBool("health-only")
		metrics, _ := cmd.Flags().GetBool("metrics")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Single service mode.
		if len(args) == 1 {
			exitCode, err := runSingleServiceStatus(ctx, args[0], jsonOut, verbose)
			if err != nil {
				return err
			}
			if exitCode != 0 {
				cmd.Root().SetContext(context.WithValue(cmd.Root().Context(), exitCodeKey, exitCode))
			}
			return nil
		}

		// Full status: load config to discover enabled services.
		rawCwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		cwd, err := config.FindNSelfRoot(rawCwd)
		if err != nil {
			return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
		}

		cfg, err := config.Load(cwd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		report, err := health.RunAllChecks(ctx, cfg, cwd)
		if err != nil {
			return fmt.Errorf("failed to run health checks: %w", err)
		}

		if jsonOut {
			return printStatusJSON(report)
		}

		printStatusTable(report, verbose, healthOnly, metrics)

		// Exit code: 2 if any unhealthy, 1 if some in intermediate state.
		if report.Unhealthy > 0 {
			cmd.Root().SetContext(context.WithValue(cmd.Root().Context(), exitCodeKey, 2))
		} else if report.Healthy < report.Total {
			// Some services in intermediate state (starting, etc.) — warning
			cmd.Root().SetContext(context.WithValue(cmd.Root().Context(), exitCodeKey, 1))
		}
		return nil
	},
}

// runSingleServiceStatus checks a single service by name.
// Returns an exit code (0 = healthy, 2 = unhealthy) and any error.
func runSingleServiceStatus(ctx context.Context, service string, jsonOut, verbose bool) (int, error) {
	result, err := health.CheckService(ctx, service)
	if err != nil {
		return 0, wrapDockerError(err, service)
	}

	if jsonOut {
		report := &health.HealthReport{
			Timestamp: time.Now(),
			Results:   []health.HealthResult{*result},
			Total:     1,
		}
		if result.Status == "healthy" {
			report.Healthy = 1
		} else {
			report.Unhealthy = 1
		}
		return 0, printStatusJSON(report)
	}

	// Simple single-service output.
	icon := ui.C(ui.Green, ui.IconSuccess)
	statusText := ui.C(ui.Green, "healthy")
	if result.Status != "healthy" {
		icon = ui.C(ui.Red, ui.IconFailure)
		statusText = ui.C(ui.Red, result.Status)
	}

	fmt.Printf("%s %s  %s", icon, statusText, result.Service)
	if verbose {
		fmt.Printf("  (%s, %s)", result.Duration.Round(time.Millisecond), result.Details)
	} else {
		fmt.Printf("  (%s)", result.Details)
	}
	fmt.Println()

	if result.Status != "healthy" {
		return 2, nil
	}
	return 0, nil
}

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

	for _, r := range report.Results {
		var icon, statusText string
		if r.Status == "healthy" {
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
	if strings.Contains(msg, "Docker is not running") ||
		strings.Contains(msg, "Cannot connect to the Docker daemon") {
		return fmt.Errorf("Docker is not running — start Docker Desktop or run: systemctl start docker")
	}
	if strings.Contains(msg, "No such container") || strings.Contains(msg, "not found") {
		return fmt.Errorf("container %s is not running — try: nself start", containerName)
	}
	return fmt.Errorf("failed to inspect container %s: %w", containerName, err)
}

func init() {
	statusCmd.Flags().BoolP("json", "j", false, "JSON output")
	statusCmd.Flags().Bool("verbose", false, "Show resource usage, uptime")
	statusCmd.Flags().Bool("health-only", false, "Show only health status")
	statusCmd.Flags().Bool("metrics", false, "Show performance metrics")
	RootCmd.AddCommand(statusCmd)
}
