package commands

import (
	"testing"
)

func TestFilterLogLine_Search(t *testing.T) {
	opts := LogsOptions{Search: "migration"}

	line := "2024-01-01T00:00:00Z running migration step 1"
	got, ok := filterLogLine(line, opts)
	if !ok {
		t.Errorf("expected line to pass search filter, but it was rejected")
	}
	if got != line {
		t.Errorf("expected line unchanged, got %q", got)
	}

	line2 := "2024-01-01T00:00:00Z normal startup"
	_, ok2 := filterLogLine(line2, opts)
	if ok2 {
		t.Errorf("expected line to be rejected by search filter")
	}
}

func TestFilterLogLine_SearchCaseInsensitive(t *testing.T) {
	opts := LogsOptions{Search: "Migration"}

	line := "2024-01-01T00:00:00Z running migration step 1"
	_, ok := filterLogLine(line, opts)
	if !ok {
		t.Errorf("search filter should be case-insensitive")
	}
}

func TestFilterLogLine_Errors(t *testing.T) {
	opts := LogsOptions{Errors: true}

	errLine := `{"level":"error","msg":"connection refused"}`
	_, ok := filterLogLine(errLine, opts)
	if !ok {
		t.Errorf("expected error line to pass errors filter")
	}

	infoLine := `{"level":"info","msg":"started"}`
	_, ok2 := filterLogLine(infoLine, opts)
	if ok2 {
		t.Errorf("expected info line to be rejected by errors filter")
	}

	unknownLine := "some arbitrary log line with no level"
	_, ok3 := filterLogLine(unknownLine, opts)
	if ok3 {
		t.Errorf("expected line with no detectable error level to be rejected by errors filter")
	}
}

func TestFilterLogLine_Level(t *testing.T) {
	opts := LogsOptions{Level: "warn"}

	warnLine := "2024-01-01 [warn] disk usage high"
	_, ok := filterLogLine(warnLine, opts)
	if !ok {
		t.Errorf("expected warn line to pass level=warn filter")
	}

	errorLine := "2024-01-01 [error] out of memory"
	_, ok2 := filterLogLine(errorLine, opts)
	if !ok2 {
		t.Errorf("expected error line to pass level=warn filter (error >= warn)")
	}

	infoLine := "2024-01-01 [info] request completed"
	_, ok3 := filterLogLine(infoLine, opts)
	if ok3 {
		t.Errorf("expected info line to be rejected by level=warn filter")
	}

	// Lines with no detectable level pass through
	unknownLine := "some line with no level markers"
	_, ok4 := filterLogLine(unknownLine, opts)
	if !ok4 {
		t.Errorf("expected line with no detectable level to pass through level filter")
	}
}

func TestFilterLogLine_NoFilter(t *testing.T) {
	opts := LogsOptions{}

	line := "any log line"
	got, ok := filterLogLine(line, opts)
	if !ok {
		t.Errorf("expected line to pass with no filters set")
	}
	if got != line {
		t.Errorf("expected line unchanged, got %q", got)
	}
}

func TestFormatLogLine_Compact(t *testing.T) {
	opts := LogsOptions{Compact: true}

	line := "2024-01-01T12:34:56Z some message here"
	got := formatLogLine(line, "", opts)
	if got != "some message here" {
		t.Errorf("compact mode should strip timestamp prefix, got %q", got)
	}

	line2 := "2024-01-01 12:34:56 another message"
	got2 := formatLogLine(line2, "", opts)
	if got2 != "another message" {
		t.Errorf("compact mode should strip space-separated timestamp, got %q", got2)
	}
}

func TestFormatLogLine_Quiet(t *testing.T) {
	opts := LogsOptions{Quiet: true}

	line := "postgres | some log message"
	got := formatLogLine(line, "", opts)
	if got != "some log message" {
		t.Errorf("quiet mode should strip service prefix, got %q", got)
	}

	line2 := "[postgres] some log message"
	got2 := formatLogLine(line2, "", opts)
	if got2 != "some log message" {
		t.Errorf("quiet mode should strip bracketed service prefix, got %q", got2)
	}
}

func TestFormatLogLine_None(t *testing.T) {
	opts := LogsOptions{}

	line := "2024-01-01T12:34:56Z postgres | some message"
	got := formatLogLine(line, "", opts)
	if got != line {
		t.Errorf("no formatting flags: expected line unchanged, got %q", got)
	}
}
