package commands

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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

// LogsOptions holds parsed flag values for log filtering and formatting.
type LogsOptions struct {
	Search  string
	Errors  bool
	Level   string // debug|info|warn|error
	Compact bool
	Quiet   bool
	Tail    int
	Follow  bool
}

var timestampPrefixRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?\s*`)
var servicePrefixRE = regexp.MustCompile(`^(\[[^\]]+\]\s*|\S+\s*\|\s*)`)

// logLevelValue returns a numeric weight for a log level string.
// Returns -1 for unknown levels.
func logLevelValue(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return 0
	case "info":
		return 1
	case "warn", "warning":
		return 2
	case "error", "err", "fatal", "crit", "critical":
		return 3
	}
	return -1
}

// detectLineLevel scans a log line for level indicators and returns the level string.
// Returns "" if no level can be detected.
func detectLineLevel(line string) string {
	lower := strings.ToLower(line)

	// Structured: level=error or "level":"error"
	for _, kv := range []string{`level=error`, `"level":"error"`, `level="error"`} {
		if strings.Contains(lower, kv) {
			return "error"
		}
	}
	for _, kv := range []string{`level=warn`, `"level":"warn"`, `level="warn"`, `level=warning`, `"level":"warning"`} {
		if strings.Contains(lower, kv) {
			return "warn"
		}
	}
	for _, kv := range []string{`level=info`, `"level":"info"`, `level="info"`} {
		if strings.Contains(lower, kv) {
			return "info"
		}
	}
	for _, kv := range []string{`level=debug`, `"level":"debug"`, `level="debug"`} {
		if strings.Contains(lower, kv) {
			return "debug"
		}
	}

	// Bracketed: [error], [warn], [info], [debug]
	for _, pat := range []string{"[error]", "[err]", "[fatal]", "[crit]"} {
		if strings.Contains(lower, pat) {
			return "error"
		}
	}
	for _, pat := range []string{"[warn]", "[warning]"} {
		if strings.Contains(lower, pat) {
			return "warn"
		}
	}
	if strings.Contains(lower, "[info]") {
		return "info"
	}
	if strings.Contains(lower, "[debug]") {
		return "debug"
	}

	// Bare words surrounded by spaces or at word boundary
	if strings.Contains(lower, " error ") || strings.Contains(lower, "\terror\t") ||
		strings.HasSuffix(lower, " error") || strings.HasPrefix(lower, "error ") {
		return "error"
	}
	if strings.Contains(lower, " warn ") || strings.Contains(lower, " warning ") {
		return "warn"
	}

	return ""
}

// filterLogLine applies active filters to a log line.
// Returns (line, true) if the line passes all filters, ("", false) otherwise.
func filterLogLine(line string, opts LogsOptions) (string, bool) {
	if opts.Search != "" {
		if !strings.Contains(strings.ToLower(line), strings.ToLower(opts.Search)) {
			return "", false
		}
	}

	if opts.Errors {
		detected := detectLineLevel(line)
		if logLevelValue(detected) < logLevelValue("error") {
			return "", false
		}
	} else if opts.Level != "" {
		detected := detectLineLevel(line)
		// Lines with no detectable level pass through
		if detected != "" && logLevelValue(detected) < logLevelValue(opts.Level) {
			return "", false
		}
	}

	return line, true
}

// formatLogLine applies compact and quiet formatting transforms to a log line.
func formatLogLine(line, containerPrefix string, opts LogsOptions) string {
	if opts.Compact {
		line = timestampPrefixRE.ReplaceAllString(line, "")
	}
	if opts.Quiet {
		line = servicePrefixRE.ReplaceAllString(line, "")
	}
	return line
}

// streamFilteredLogs runs docker with composeArgs and streams filtered/formatted output to w.
func streamFilteredLogs(ctx context.Context, workdir string, composeArgs []string, opts LogsOptions, w io.Writer) error {
	dockerCmd := exec.CommandContext(ctx, "docker", composeArgs...)
	dockerCmd.Dir = workdir
	dockerCmd.Stderr = os.Stderr

	if opts.Follow {
		dockerCmd.Stdin = os.Stdin
	}

	stdout, err := dockerCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := dockerCmd.Start(); err != nil {
		return fmt.Errorf("starting docker: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		filtered, ok := filterLogLine(line, opts)
		if !ok {
			continue
		}
		formatted := formatLogLine(filtered, "", opts)
		fmt.Fprintln(w, formatted)
	}

	waitErr := dockerCmd.Wait()
	if ctx.Err() != nil {
		return nil
	}
	return waitErr
}

// getRunningServices returns the list of service names from docker compose ps --services.
func getRunningServices(ctx context.Context, workdir string) ([]string, error) {
	out, err := exec.CommandContext(ctx, "docker", "compose", "ps", "--services").Output()
	if err != nil {
		return nil, fmt.Errorf("docker compose ps --services: %w", err)
	}
	var services []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			services = append(services, line)
		}
	}
	return services, nil
}

// LogSummary holds per-service log analysis results.
type LogSummary struct {
	Service    string
	TotalLines int
	ErrorCount int
	WarnCount  int
	TopErrors  []string // up to 5 most frequent error messages (deduped)
	LastLine   string
}

// CollectLogSummary fetches and analyses recent logs for a single service.
func CollectLogSummary(ctx context.Context, workdir string, service string, maxLines int) (*LogSummary, error) {
	summary := &LogSummary{Service: service}

	cmd := exec.CommandContext(ctx, "docker", "compose", "logs",
		"--tail", strconv.Itoa(maxLines),
		"--no-log-prefix",
		service,
	)
	cmd.Dir = workdir
	out, err := cmd.Output()
	if err != nil {
		// Service may not be running — return empty summary, no error.
		return summary, nil
	}

	errorFreq := make(map[string]int)
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		summary.TotalLines++
		summary.LastLine = line

		level := detectLineLevel(line)
		switch level {
		case "error":
			summary.ErrorCount++
			// Strip timestamp for dedup key
			key := timestampPrefixRE.ReplaceAllString(line, "")
			errorFreq[key]++
		case "warn":
			summary.WarnCount++
		}
	}

	// Build top 5 errors sorted by frequency descending.
	type errEntry struct {
		msg   string
		count int
	}
	entries := make([]errEntry, 0, len(errorFreq))
	for msg, cnt := range errorFreq {
		entries = append(entries, errEntry{msg, cnt})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	max := 5
	if len(entries) < max {
		max = len(entries)
	}
	for i := 0; i < max; i++ {
		summary.TopErrors = append(summary.TopErrors, entries[i].msg)
	}

	return summary, nil
}

// LogVolume holds per-service log volume sample data.
type LogVolume struct {
	Service     string
	LinesPerMin float64
	BytesPerMin float64
	ErrorRate   float64 // errors / total lines in sample window
}

// SampleLogVolume samples docker logs for a service over sampleDuration and computes volume metrics.
func SampleLogVolume(ctx context.Context, workdir string, service string, sampleDuration time.Duration) (*LogVolume, error) {
	vol := &LogVolume{Service: service}

	sinceSeconds := int(sampleDuration.Seconds())
	cmd := exec.CommandContext(ctx, "docker", "compose", "logs",
		"--since", strconv.Itoa(sinceSeconds)+"s",
		"--no-log-prefix",
		service,
	)
	cmd.Dir = workdir
	out, err := cmd.Output()
	if err != nil {
		// Service may not be running — return empty volume, no error.
		return vol, nil
	}

	totalLines := 0
	totalBytes := len(out)
	errorLines := 0
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		totalLines++
		if detectLineLevel(line) == "error" {
			errorLines++
		}
	}

	minutes := sampleDuration.Minutes()
	if minutes > 0 && totalLines > 0 {
		vol.LinesPerMin = float64(totalLines) / minutes
		vol.BytesPerMin = float64(totalBytes) / minutes
	}
	if totalLines > 0 {
		vol.ErrorRate = float64(errorLines) / float64(totalLines)
	}

	return vol, nil
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
	errorsOnly, _ := cmd.Flags().GetBool("errors")
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

	opts := LogsOptions{
		Search:  search,
		Errors:  errorsOnly,
		Level:   level,
		Compact: compact,
		Quiet:   quiet,
		Tail:    tail,
		Follow:  follow,
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

	if err := streamFilteredLogs(cmd.Context(), workdir, composeArgs, opts, os.Stdout); err != nil {
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

	var services []string
	if len(args) > 0 {
		services = []string{args[0]}
	} else {
		var err error
		services, err = getRunningServices(cmd.Context(), workdir)
		if err != nil {
			return fmt.Errorf("listing services: %w", err)
		}
	}

	tbl := ui.NewTable("Service", "Lines", "Errors", "Warnings", "Last Error")
	for _, svc := range services {
		summary, err := CollectLogSummary(cmd.Context(), workdir, svc, 500)
		if err != nil {
			return fmt.Errorf("collecting log summary for %s: %w", svc, err)
		}

		lastError := "—"
		if len(summary.TopErrors) > 0 {
			msg := summary.TopErrors[0]
			if len(msg) > 50 {
				msg = msg[:50]
			}
			lastError = msg
		}

		tbl.AddRow(
			summary.Service,
			strconv.Itoa(summary.TotalLines),
			strconv.Itoa(summary.ErrorCount),
			strconv.Itoa(summary.WarnCount),
			lastError,
		)
	}
	tbl.Render()

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

	services, err := getRunningServices(cmd.Context(), workdir)
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	volumes := make([]*LogVolume, 0, len(services))
	for _, svc := range services {
		vol, err := SampleLogVolume(cmd.Context(), workdir, svc, 30*time.Second)
		if err != nil {
			return fmt.Errorf("sampling log volume for %s: %w", svc, err)
		}
		volumes = append(volumes, vol)
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].LinesPerMin > volumes[j].LinesPerMin
	})

	tbl := ui.NewTable("Service", "Lines/min", "Bytes/min", "Error Rate")
	for _, vol := range volumes {
		tbl.AddRow(
			vol.Service,
			fmt.Sprintf("%.1f", vol.LinesPerMin),
			fmt.Sprintf("%.1f", vol.BytesPerMin),
			fmt.Sprintf("%.1f%%", vol.ErrorRate*100),
		)
	}
	tbl.Render()

	return nil
}
