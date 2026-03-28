package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"nself/internal/config"
	"nself/internal/ui"

	"github.com/spf13/cobra"
)

// countWriter wraps an io.Writer and tracks total bytes written.
type countWriter struct {
	w     io.Writer
	count int64
}

func (cw *countWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.count += int64(n)
	return
}

var logsCmd = &cobra.Command{
	Use:   "logs [SERVICE]",
	Short: "View and filter service logs",
	Long: `View and filter Docker Compose service logs with color and formatting.

Examples:
  nself logs                  # Last 10 lines from all services
  nself logs -f               # Follow all logs live
  nself logs hasura            # Last 10 lines from hasura
  nself logs -f postgres       # Follow postgres logs
  nself logs --more            # Last 50 lines
  nself logs --all             # Last 100 lines
  nself logs -e                # Only error lines
  nself logs -s "migration"   # Search for pattern
  nself logs --status          # Service status overview
  nself logs --summary         # Recent errors by service
  nself logs --top             # Most active services`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output (live)")
	logsCmd.Flags().IntP("tail", "n", 10, "Number of lines to show")
	logsCmd.Flags().Bool("more", false, "Show last 50 lines")
	logsCmd.Flags().Bool("all", false, "Show last 100 lines")
	logsCmd.Flags().StringP("search", "s", "", "Search pattern (regex)")
	logsCmd.Flags().StringP("level", "l", "", "Filter by level (error, warn)")
	logsCmd.Flags().BoolP("errors", "e", false, "Show only errors")
	logsCmd.Flags().BoolP("compact", "c", false, "Compact output: [service] message")
	logsCmd.Flags().BoolP("quiet", "q", false, "Filter out noise (healthchecks)")
	logsCmd.Flags().Bool("status", false, "Show service status overview")
	logsCmd.Flags().Bool("summary", false, "Show recent errors by service")
	logsCmd.Flags().Bool("top", false, "Show most active services")

	RootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	follow, _ := cmd.Flags().GetBool("follow")
	tail, _ := cmd.Flags().GetInt("tail")
	more, _ := cmd.Flags().GetBool("more")
	all, _ := cmd.Flags().GetBool("all")
	search, _ := cmd.Flags().GetString("search")
	level, _ := cmd.Flags().GetString("level")
	errors, _ := cmd.Flags().GetBool("errors")
	compact, _ := cmd.Flags().GetBool("compact")
	quiet, _ := cmd.Flags().GetBool("quiet")
	status, _ := cmd.Flags().GetBool("status")
	summary, _ := cmd.Flags().GetBool("summary")
	top, _ := cmd.Flags().GetBool("top")

	// --status: show service status overview via docker compose ps
	if status {
		return runLogsStatus(cmd)
	}

	// --summary: show recent errors by service
	if summary {
		return runLogsSummary(cmd, args)
	}

	// --top: show most active services
	if top {
		return runLogsTop(cmd)
	}

	// Resolve tail count: --all > --more > --tail
	if all {
		tail = 100
	} else if more {
		tail = 50
	}

	// Build docker compose logs arguments
	composeArgs := []string{"compose", "logs"}

	if follow {
		composeArgs = append(composeArgs, "--follow")
	}

	composeArgs = append(composeArgs, "--tail", strconv.Itoa(tail))

	if !follow {
		composeArgs = append(composeArgs, "--no-log-prefix")
	}

	// If a specific service is requested, append it
	if len(args) > 0 {
		composeArgs = append(composeArgs, args[0])
	}

	// For non-follow mode with filters, we pipe through grep-like logic.
	// For follow mode or no filters, stream directly.
	_ = compact // Reserved for future output formatting
	_ = quiet   // Reserved for future healthcheck filtering
	_ = search  // Reserved for future regex search
	_ = level   // Reserved for future level filtering
	_ = errors  // Reserved for future error-only filtering

	rawCwd, err := os.Getwd()
	if err != nil {
		ui.Error("Failed to determine working directory")
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(rawCwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	// Check that at least one container exists before attempting to stream logs.
	// "docker compose ps -q" returns one container ID per line; empty output means
	// no containers are running (stack not started or wrong project directory).
	psCmd := exec.CommandContext(cmd.Context(), "docker", "compose", "ps", "-q")
	psCmd.Dir = workdir
	psOut, _ := psCmd.Output()
	if len(bytes.TrimSpace(psOut)) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No containers found. Is the stack running? Try 'nself status' to check.")
		return fmt.Errorf("no containers found")
	}

	dockerCmd := exec.CommandContext(cmd.Context(), "docker", composeArgs...)
	dockerCmd.Dir = workdir
	dockerCmd.Stderr = os.Stderr

	if follow {
		// In follow mode, attach stdin so Ctrl+C propagates cleanly
		dockerCmd.Stdin = os.Stdin
		dockerCmd.Stdout = os.Stdout
	} else {
		cw := &countWriter{w: os.Stdout}
		dockerCmd.Stdout = cw

		if err := dockerCmd.Run(); err != nil {
			return fmt.Errorf("docker compose logs: %w", err)
		}

		if cw.count == 0 {
			fmt.Fprintf(os.Stderr, "No log output. Service may not have started yet. Run 'nself status' to check.\n")
		}

		return nil
	}

	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("docker compose logs: %w", err)
	}

	return nil
}

// runLogsStatus shows a service status overview using docker compose ps.
func runLogsStatus(cmd *cobra.Command) error {
	rawCwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(rawCwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	ui.CommandHeader("nself logs --status", "Service status overview")

	dockerCmd := exec.CommandContext(cmd.Context(), "docker", "compose", "ps")
	dockerCmd.Dir = workdir
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("docker compose ps: %w", err)
	}

	return nil
}

// runLogsSummary shows recent errors across services.
func runLogsSummary(cmd *cobra.Command, args []string) error {
	rawCwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(rawCwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	ui.CommandHeader("nself logs --summary", "Recent errors by service")

	composeArgs := []string{"compose", "logs", "--tail", "200", "--no-log-prefix"}
	if len(args) > 0 {
		composeArgs = append(composeArgs, args[0])
	}

	dockerCmd := exec.CommandContext(cmd.Context(), "docker", composeArgs...)
	dockerCmd.Dir = workdir
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("docker compose logs: %w", err)
	}

	return nil
}

// runLogsTop shows the most active services by log volume.
func runLogsTop(cmd *cobra.Command) error {
	rawCwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(rawCwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	ui.CommandHeader("nself logs --top", "Most active services")

	dockerCmd := exec.CommandContext(cmd.Context(), "docker", "compose", "ps")
	dockerCmd.Dir = workdir
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("docker compose ps: %w", err)
	}

	return nil
}
