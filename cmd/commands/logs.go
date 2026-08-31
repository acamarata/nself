package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [SERVICE]",
	Short: "View and filter service logs",
	Long: `View and filter Docker Compose service logs with color and formatting.

Examples:
  nself logs                           # Last 10 lines from all services
  nself logs -f                        # Follow all logs live
  nself logs hasura                    # Last 10 lines from hasura
  nself logs -f postgres               # Follow postgres logs
  nself logs --more                    # Last 50 lines
  nself logs --all                     # Last 100 lines
  nself logs -e                        # Only error lines
  nself logs -s "migration"            # Search for pattern
  nself logs --grep "error.*timeout"   # Regex filter
  nself logs --since 1h                # Logs from last hour
  nself logs --since 1h --until 30m    # Between 1h and 30m ago
  nself logs -S hasura -S postgres     # Multiple services
  nself logs --json                    # Structured JSON output
  nself logs --plain | grep error      # Plain output for piping
  nself logs --status                  # Service status overview
  nself logs --summary                 # Recent errors by service
  nself logs --top                     # Most active services`,
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
	logsCmd.Flags().String("grep", "", "Regex pattern filter")
	logsCmd.Flags().String("since", "", "Show logs since (e.g. 1h, 30m, or RFC3339)")
	logsCmd.Flags().String("until", "", "Show logs until (e.g. 1h or RFC3339)")
	logsCmd.Flags().Bool("json", false, "Structured JSON output per line")
	logsCmd.Flags().Bool("no-color", false, "Disable colored output")
	logsCmd.Flags().Bool("plain", false, "Plain output (no highlighting, for piping)")
	logsCmd.Flags().StringSliceP("service", "S", nil, "Filter by service name (repeatable)")
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
	grepPattern, _ := cmd.Flags().GetString("grep")
	level, _ := cmd.Flags().GetString("level")
	errorsOnly, _ := cmd.Flags().GetBool("errors")
	compact, _ := cmd.Flags().GetBool("compact")
	quiet, _ := cmd.Flags().GetBool("quiet")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	jsonOut, _ := cmd.Flags().GetBool("json")
	noColor, _ := cmd.Flags().GetBool("no-color")
	plain, _ := cmd.Flags().GetBool("plain")
	services, _ := cmd.Flags().GetStringSlice("service")
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

	opts := LogsOptions{
		Search:  search,
		Grep:    grepPattern,
		Errors:  errorsOnly,
		Level:   level,
		Compact: compact,
		Quiet:   quiet,
		Tail:    tail,
		Follow:  follow,
		Since:   since,
		Until:   until,
		JSON:    jsonOut,
		NoColor: noColor,
		Plain:   plain,
		Service: services,
	}

	// Build docker compose logs arguments
	composeArgs := []string{"compose", "logs"}

	if follow {
		composeArgs = append(composeArgs, "--follow")
	}

	composeArgs = append(composeArgs, "--tail", strconv.Itoa(tail))

	if since != "" {
		composeArgs = append(composeArgs, "--since", since)
	}
	if until != "" {
		composeArgs = append(composeArgs, "--until", until)
	}

	if !follow {
		composeArgs = append(composeArgs, "--no-log-prefix")
	}

	// If --no-color or --plain, pass --no-color to docker compose
	if noColor || plain {
		composeArgs = append(composeArgs, "--no-color")
	}

	// Collect target services: --service flags + positional arg
	var targetServices []string
	targetServices = append(targetServices, services...)
	if len(args) > 0 {
		targetServices = append(targetServices, args[0])
	}
	composeArgs = append(composeArgs, targetServices...)

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
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No containers found. Is the stack running? Try 'nself status' to check.")
		return fmt.Errorf("no containers found")
	}

	if err := streamFilteredLogs(cmd.Context(), workdir, composeArgs, opts, os.Stdout); err != nil {
		return fmt.Errorf("docker compose logs: %w", err)
	}

	return nil
}
