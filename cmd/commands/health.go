package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/health"

	"github.com/spf13/cobra"
)

// Flags shared across health subcommands.
var (
	healthTimeout  int
	healthInterval int
	healthRetries  int
	healthEnv      string
	healthJSON     bool
	healthQuiet    bool
)

var healthCmd = &cobra.Command{
	Use:   "health [subcommand]",
	Short: "Health check management with continuous monitoring",
	Long: `Run health checks against all services, individual services, or HTTP
endpoints. Supports continuous watch mode and history viewing.`,
	// Default action when no subcommand is given: run all checks.
	RunE: healthCheckRunE,
}

var healthCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Run all health checks",
	RunE:  healthCheckRunE,
}

var healthServiceCmd = &cobra.Command{
	Use:   "service <name>",
	Short: "Check a single service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(healthTimeout)*time.Second)
		defer cancel()

		var lastErr error
		var result *health.HealthResult
		for attempt := 0; attempt < healthRetries; attempt++ {
			result, lastErr = health.CheckService(ctx, args[0])
			if lastErr == nil && result.Status == "healthy" {
				break
			}
		}
		if lastErr != nil {
			return fmt.Errorf("checking service %s: %w", args[0], lastErr)
		}

		if healthJSON {
			return printJSON(result)
		}
		if healthQuiet && result.Status == "healthy" {
			return nil
		}
		printServiceResult(result)
		return nil
	},
}

var healthEndpointCmd = &cobra.Command{
	Use:   "endpoint <url>",
	Short: "Check an HTTP endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(healthTimeout)*time.Second)
		defer cancel()

		var lastErr error
		var result *health.HealthResult
		for attempt := 0; attempt < healthRetries; attempt++ {
			result, lastErr = health.CheckEndpoint(ctx, args[0])
			if lastErr == nil && result.Status == "healthy" {
				break
			}
		}
		if lastErr != nil {
			return fmt.Errorf("checking endpoint %s: %w", args[0], lastErr)
		}

		if healthJSON {
			return printJSON(result)
		}
		if healthQuiet && result.Status == "healthy" {
			return nil
		}
		printServiceResult(result)
		return nil
	},
}

var healthWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuous health monitoring",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, workdir, err := loadHealthConfig()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		interval := time.Duration(healthInterval) * time.Second
		ch, err := health.WatchHealth(ctx, cfg, workdir, interval)
		if err != nil {
			return fmt.Errorf("starting watch: %w", err)
		}

		for report := range ch {
			if healthQuiet && report.Unhealthy == 0 {
				continue
			}
			if healthJSON {
				if err := printJSON(report); err != nil {
					return err
				}
			} else {
				printReport(report)
			}
		}
		return nil
	},
}

var healthHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show last 20 health checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := healthHistoryDir()
		if err != nil {
			return err
		}

		entries, err := health.GetHistory(dir, 20)
		if err != nil {
			return fmt.Errorf("reading history: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No health check history found.")
			return nil
		}

		if healthJSON {
			return printJSON(entries)
		}

		for _, report := range entries {
			fmt.Printf("[%s] %d/%d healthy\n",
				report.Timestamp.Format(time.RFC3339),
				report.Healthy,
				report.Total,
			)
			for _, r := range report.Results {
				printServiceResult(&r)
			}
			fmt.Println()
		}
		return nil
	},
}

var healthConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show health check settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := map[string]interface{}{
			"timeout_seconds":  healthTimeout,
			"interval_seconds": healthInterval,
			"retries":          healthRetries,
			"env":              healthEnv,
			"json_output":      healthJSON,
			"quiet":            healthQuiet,
		}
		if healthJSON {
			return printJSON(settings)
		}
		fmt.Printf("%-20s %v\n", "Timeout (seconds):", healthTimeout)
		fmt.Printf("%-20s %v\n", "Interval (seconds):", healthInterval)
		fmt.Printf("%-20s %v\n", "Retries:", healthRetries)
		fmt.Printf("%-20s %s\n", "Environment:", healthEnv)
		fmt.Printf("%-20s %v\n", "JSON output:", healthJSON)
		fmt.Printf("%-20s %v\n", "Quiet:", healthQuiet)
		return nil
	},
}

// healthCheckRunE is the default action: run all service health checks.
func healthCheckRunE(cmd *cobra.Command, args []string) error {
	cfg, workdir, err := loadHealthConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(healthTimeout)*time.Second)
	defer cancel()

	var report *health.HealthReport
	var lastErr error
	for attempt := 0; attempt < healthRetries; attempt++ {
		report, lastErr = health.RunAllChecks(ctx, cfg, workdir)
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return fmt.Errorf("running health checks: %w", lastErr)
	}

	if healthJSON {
		return printJSON(report)
	}
	if healthQuiet && report.Unhealthy == 0 {
		return nil
	}
	printReport(report)
	return nil
}

// loadHealthConfig loads the project config, optionally setting ENV from --env.
// Returns the config and the project working directory.
func loadHealthConfig() (*config.Config, string, error) {
	if healthEnv != "" {
		os.Setenv("ENV", healthEnv)
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("getting working directory: %w", err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}
	return cfg, dir, nil
}

// healthHistoryDir returns the directory where health history is stored.
func healthHistoryDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return filepath.Join(dir, ".nself", "health"), nil
}

// printReport prints a HealthReport as a formatted table.
func printReport(report *health.HealthReport) {
	fmt.Printf("%-20s %-12s %-8s %s\n", "Service", "Status", "Time", "Details")
	for _, r := range report.Results {
		printServiceResult(&r)
	}
	fmt.Printf("\n%d/%d services healthy\n", report.Healthy, report.Total)
}

// printServiceResult prints a single health result line.
func printServiceResult(r *health.HealthResult) {
	marker := "\u2717"
	if r.Status == "healthy" {
		marker = "\u2713"
	}
	fmt.Printf("%-20s %s %-10s %-8s %s\n", r.Service, marker, r.Status, r.Duration.Truncate(time.Millisecond), r.Details)
}

// printJSON marshals v to indented JSON and writes to stdout.
func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func init() {
	// Register persistent flags on the parent command so all subcommands inherit them.
	healthCmd.PersistentFlags().IntVar(&healthTimeout, "timeout", 30, "Check timeout in seconds")
	healthCmd.PersistentFlags().IntVar(&healthInterval, "interval", 10, "Watch interval in seconds")
	healthCmd.PersistentFlags().IntVar(&healthRetries, "retries", 3, "Retry count on failure")
	healthCmd.PersistentFlags().StringVar(&healthEnv, "env", "", "Environment to load config for")
	healthCmd.PersistentFlags().BoolVar(&healthJSON, "json", false, "Output in JSON format")
	healthCmd.PersistentFlags().BoolVar(&healthQuiet, "quiet", false, "Only output on failure")

	// Register subcommands.
	healthCmd.AddCommand(healthCheckCmd)
	healthCmd.AddCommand(healthServiceCmd)
	healthCmd.AddCommand(healthEndpointCmd)
	healthCmd.AddCommand(healthWatchCmd)
	healthCmd.AddCommand(healthHistoryCmd)
	healthCmd.AddCommand(healthConfigCmd)

	RootCmd.AddCommand(healthCmd)
}
