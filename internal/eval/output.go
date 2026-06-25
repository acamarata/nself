package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// Purpose: Output formatters for nself ci eval command results.
//   Three modes: JSON (artifact file), table (terminal table), summary (one-liner).
//   JSON artifact is always written to {repo}/.nself-ci/artifacts/eval-results.json.
// Inputs:  EvalRunResult; output format string; repo root path.
// Outputs: Formatted text to writer; artifact file on disk for JSON/table modes.
// Constraints: Artifact dir is created if absent. Never panic on zero-task results.
// SPORT: CLI-CMD-EVAL-001

const (
	// FormatJSON emits full JSON artifact + path message.
	FormatJSON = "json"
	// FormatTable emits a per-task terminal table.
	FormatTable = "table"
	// FormatSummary emits one line per suite with final pass/fail.
	FormatSummary = "summary"

	// artifactsDir is the relative path under repo root for CI artifacts.
	artifactsDir = ".nself-ci/artifacts"
	// artifactFile is the eval results artifact filename.
	artifactFile = "eval-results.json"
)

// WriteResult writes the eval run result in the requested format.
// Purpose: Dispatch to the correct formatter and write artifact file.
// Inputs:  w — output writer; result — run result; format — one of json/table/summary;
//   repoRoot — path where {repoRoot}/.nself-ci/artifacts/ is written.
// Outputs: formatted text to w; eval-results.json on disk.
// Constraints: Always writes JSON artifact regardless of format flag.
func WriteResult(w io.Writer, result EvalRunResult, format, repoRoot string) error {
	// Always write the JSON artifact.
	if err := writeArtifact(result, repoRoot); err != nil {
		// Non-fatal: print warning and continue.
		fmt.Fprintf(w, "warning: could not write eval artifact: %v\n", err)
	}

	switch format {
	case FormatJSON:
		return writeJSON(w, result)
	case FormatTable:
		return writeTable(w, result)
	default: // FormatSummary
		return writeSummary(w, result)
	}
}

// writeArtifact serialises result to {repoRoot}/.nself-ci/artifacts/eval-results.json.
func writeArtifact(result EvalRunResult, repoRoot string) error {
	dir := filepath.Join(repoRoot, artifactsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create artifacts dir: %w", err)
	}
	path := filepath.Join(dir, artifactFile)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal eval result: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// writeJSON writes pretty-printed JSON to w.
func writeJSON(w io.Writer, result EvalRunResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal eval result: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// writeTable writes a per-task tabular report to w.
func writeTable(w io.Writer, result EvalRunResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	status := "PASSED"
	if !result.Passed {
		status = "FAILED"
	}

	fmt.Fprintf(tw, "Suite:\t%s\n", result.SuiteSlug)
	fmt.Fprintf(tw, "Run ID:\t%s\n", result.ID)
	fmt.Fprintf(tw, "Status:\t%s\n", status)
	fmt.Fprintf(tw, "Pass Rate:\t%.1f%%\n", result.PassRate*100)
	fmt.Fprintf(tw, "Suite Score:\t%.4f\n", result.SuiteScore)
	fmt.Fprintln(tw)

	if len(result.Tasks) == 0 {
		fmt.Fprintln(tw, "(no task results)")
		return nil
	}

	// Header.
	fmt.Fprintf(tw, "TASK\tMODE\tSCORE\tPASS\tRATIONALE\n")
	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 24), strings.Repeat("-", 8),
		strings.Repeat("-", 6), strings.Repeat("-", 4),
		strings.Repeat("-", 30))

	for _, t := range result.Tasks {
		passStr := "no"
		if t.Passed {
			passStr = "yes"
		}
		rationale := t.Rationale
		if len(rationale) > 50 {
			rationale = rationale[:47] + "..."
		}
		fmt.Fprintf(tw, "%s\t%s\t%.4f\t%s\t%s\n",
			truncate(t.TaskID, 24), t.ScoringMode, t.Score, passStr, rationale)
	}
	return nil
}

// writeSummary writes a one-line suite result to w.
func writeSummary(w io.Writer, result EvalRunResult) error {
	icon := "✓"
	if !result.Passed {
		icon = "✗"
	}
	_, err := fmt.Fprintf(w, "%s  %s  pass_rate=%.1f%%  score=%.4f  [%s]\n",
		icon, result.SuiteSlug,
		result.PassRate*100,
		result.SuiteScore,
		result.Status,
	)
	return err
}

// WriteGateStatus writes the gate check result to w.
// Purpose: Format the autonomy tier gate check for CLI output.
// Inputs:  w — writer; gs — gate status; verbose — print blocking suites.
// Outputs: formatted text to w.
func WriteGateStatus(w io.Writer, gs GateStatus, verbose bool) {
	icon := "✓"
	label := "CLEARED"
	if !gs.Cleared {
		icon = "✗"
		label = "BLOCKED"
	}
	fmt.Fprintf(w, "%s  tier=%s  %s\n", icon, gs.Tier, label)
	if !gs.Cleared && len(gs.BlockingSuites) > 0 {
		fmt.Fprintln(w, "Blocking suites:")
		for _, slug := range gs.BlockingSuites {
			fmt.Fprintf(w, "  - %s\n", slug)
		}
	}
	if !gs.Enforced {
		fmt.Fprintln(w, "  (note: tier enforcement is disabled — gate result is advisory)")
	}
}

// WriteValidationErrors prints validation errors to w.
func WriteValidationErrors(w io.Writer, errs []ValidationError) {
	fmt.Fprintf(w, "Validation failed (%d error(s)):\n", len(errs))
	for _, e := range errs {
		fmt.Fprintf(w, "  [%s] %s\n", e.Field, e.Message)
	}
}

// truncate trims s to max runes, appending "..." if trimmed.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
