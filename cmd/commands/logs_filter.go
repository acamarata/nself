package commands

// Purpose: Log line filtering/formatting primitives split out of logs.go
// (CLI-R12 Batch B mechanical file-size split). Holds LogsOptions, the
// level-detection regexes/helpers, and the per-line filter/format/JSON
// transforms used while streaming `docker compose logs` output.
// Inputs: raw log lines (strings) and a LogsOptions describing active
// filters (--search, --grep, --level, --errors, --compact, --quiet, --json).
// Outputs: filtered/formatted line strings, or a JSON-encoded line.
// Constraints: pure move, no behavior change. Consumed by
// streamFilteredLogs/CollectLogSummary/SampleLogVolume in the sibling
// logs_stream.go and logs_summary.go files.

import (
	"encoding/json"
	"regexp"
	"strings"
)

// LogsOptions holds parsed flag values for log filtering and formatting.
type LogsOptions struct {
	Search  string
	Grep    string // regex pattern for --grep
	Errors  bool
	Level   string // debug|info|warn|error
	Compact bool
	Quiet   bool
	Tail    int
	Follow  bool
	Since   string // relative (1h, 30m) or absolute RFC3339
	Until   string // relative or absolute RFC3339
	JSON    bool   // structured JSON output per line
	NoColor bool
	Plain   bool // disable highlighting for piping
	Service []string
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

	if opts.Grep != "" {
		re, err := regexp.Compile(opts.Grep)
		if err == nil && !re.MatchString(line) {
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

// logLineToJSON converts a log line into a JSON object with extracted fields.
func logLineToJSON(line string) string {
	entry := map[string]string{
		"message": line,
	}
	if level := detectLineLevel(line); level != "" {
		entry["level"] = level
	}
	// Extract timestamp if present
	if loc := timestampPrefixRE.FindString(line); loc != "" {
		entry["timestamp"] = strings.TrimSpace(loc)
		entry["message"] = strings.TrimSpace(timestampPrefixRE.ReplaceAllString(line, ""))
	}
	// Extract service prefix if present
	if loc := servicePrefixRE.FindString(line); loc != "" {
		svc := strings.TrimSpace(strings.Trim(loc, "[]|"))
		if svc != "" {
			entry["service"] = svc
		}
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return line
	}
	return string(data)
}
