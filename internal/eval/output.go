package eval

// output.go — output formatters for eval run results.
//
// Purpose: Three output formatters for `nself ci eval` results per spec §6:
//   - JSON: write eval-results.json artifact to {repo}/.nself-ci/artifacts/
//   - table: terminal table per spec §12 UX contract (per-task progress lines)
//   - summary: one-line per suite + final PASSED/FAILED status line
//
// Inputs: EvalRunResult, output format string, artifact directory path.
// Outputs: formatted text to writer (table/summary) or JSON file (json mode).
// Constraints: JSON artifact must match EvalResultsArtifact schema from types.go;
// table format must follow UX contract steps 1-5 from spec §12.
// SPORT: F02-COMMAND-INVENTORY.md (nself ci eval output flag).

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// OutputFormat enumerates the supported output formats for `nself ci eval`.
type OutputFormat string

const (
	OutputFormatJSON    OutputFormat = "json"
	OutputFormatTable   OutputFormat = "table"
	OutputFormatSummary OutputFormat = "summary"
)

// WriteOutput dispatches to the appropriate formatter based on format.
//
// Purpose: Single entry point for all output formatting called by eval.go cobra command.
// Inputs: w — writer for terminal output; result — completed eval run; format — from --output flag;
// artifactsDir — path to {repo}/.nself-ci/artifacts/ for JSON artifact writing.
// Outputs: formatted output to w; eval-results.json written when format=="json".
// Constraints: JSON format writes artifact file AND prints the artifact path to w.
func WriteOutput(w io.Writer, result EvalRunResult, format OutputFormat, artifactsDir string) error {
	switch format {
	case OutputFormatJSON:
		return writeJSONOutput(w, result, artifactsDir)
	case OutputFormatTable:
		return writeTableOutput(w, result)
	default:
		return writeSummaryOutput(w, result)
	}
}

// writeJSONOutput writes eval-results.json to artifactsDir and prints its path.
//
// Purpose: Produce the machine-readable artifact for CI consumers downstream of
// `nself ci eval`. Artifact schema matches EvalResultsArtifact (types.go).
// Inputs: result — completed run; artifactsDir — directory path; w — for status line.
// Outputs: {artifactsDir}/eval-results.json; path confirmation line to w.
// Constraints: creates artifactsDir if not present (0755); artifact is indented JSON.
func writeJSONOutput(w io.Writer, result EvalRunResult, artifactsDir string) error {
	artifact := EvalResultsArtifact{
		RunID:      result.RunID,
		Suite:      result.Suite,
		Commit:     result.Commit,
		PassRate:   result.PassRate,
		SuiteScore: result.SuiteScore,
		Passed:     result.Passed,
		Tasks:      result.Tasks,
		DurationMS: result.DurationMS,
	}

	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("output: marshaling eval results: %w", err)
	}

	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("output: creating artifacts dir %s: %w", artifactsDir, err)
	}

	outPath := filepath.Join(artifactsDir, "eval-results.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("output: writing eval-results.json: %w", err)
	}

	fmt.Fprintf(w, "eval-results.json written to %s\n", outPath)
	return nil
}

// writeTableOutput prints per-task progress lines and a suite summary.
//
// Purpose: Implement the spec §12 UX contract for `nself ci eval --output table`:
//  1. Header: "[EVAL] Running suite: {slug} ({n} tasks)..."  (printed by caller)
//  2. Per-task line: "  {id} [{metric}={v} ...] PASS|FAIL"
//  3. Suite summary: "Suite score: {score} | Pass rate: {n}/{total} ({pct}%) | PASSED|FAILED"
//  4. On failure: appends "(threshold: {thr})"
//
// Inputs: w — terminal writer; result — completed EvalRunResult.
// Outputs: formatted table lines to w.
// Constraints: metric keys printed in iteration order from map (non-deterministic but acceptable);
// pass indicator uses PASS/FAIL for terminal readability.
func writeTableOutput(w io.Writer, result EvalRunResult) error {
	total := len(result.Tasks)
	passed := 0

	for _, task := range result.Tasks {
		if task.Passed {
			passed++
		}
		passFail := "FAIL"
		if task.Passed {
			passFail = "PASS"
		}

		// Build metric pairs: "precision_at_3=0.89 recall_at_3=0.93"
		var metricParts []string
		for k, v := range task.Metrics {
			metricParts = append(metricParts, fmt.Sprintf("%s=%.2f", k, v))
		}
		metricsStr := strings.Join(metricParts, " ")
		if metricsStr != "" {
			fmt.Fprintf(w, "  %s [%s] %s\n", task.ID, metricsStr, passFail)
		} else {
			fmt.Fprintf(w, "  %s [score=%.2f] %s\n", task.ID, task.Score, passFail)
		}
	}

	// Suite summary line.
	passedLabel := "PASSED"
	if !result.Passed {
		passedLabel = "FAILED"
	}
	pct := 0.0
	if total > 0 {
		pct = float64(passed) / float64(total) * 100
	}
	fmt.Fprintf(w, "Suite score: %.2f | Pass rate: %d/%d (%.1f%%) | %s\n",
		result.SuiteScore, passed, total, pct, passedLabel)

	return nil
}

// writeSummaryOutput prints a one-line summary per suite.
//
// Purpose: Minimal output for scripted use and CI log readability.
// Spec §12 UX contract step 3-4: single line showing score, pass rate, and status.
// Inputs: w — writer; result — completed run.
// Outputs: single summary line to w.
func writeSummaryOutput(w io.Writer, result EvalRunResult) error {
	total := len(result.Tasks)
	passed := 0
	for _, t := range result.Tasks {
		if t.Passed {
			passed++
		}
	}
	pct := 0.0
	if total > 0 {
		pct = float64(passed) / float64(total) * 100
	}
	status := "PASSED"
	if !result.Passed {
		status = "FAILED"
	}
	fmt.Fprintf(w, "[%s] suite=%s score=%.2f pass_rate=%d/%d(%.1f%%) %s\n",
		status, result.Suite, result.SuiteScore, passed, total, pct, result.RunID)
	return nil
}
