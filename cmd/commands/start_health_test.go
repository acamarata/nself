package commands

// Purpose: Regression coverage for issue #268 — one HealthResult.Status field
// judged by two opposite predicates. The aggregate in
// internal/health.RunAllChecks accepted "healthy" OR "running", while the
// per-service printers here compared only against "healthy", so a report
// could print "4/4 healthy (100%)" and "✗ nginx: running" on consecutive
// lines. Fixed by routing every printer through health.HealthResult.OK(),
// the same predicate the aggregate uses.
// Inputs: none (pure in-memory HealthReport fixtures).
// Outputs: pass/fail via testing.T; captures os.Stderr to observe what
// printServiceDetails actually writes.
// Constraints: must exercise the real printServiceDetails call path, not just
// the shared predicate, so a regression in the printer's own comparison is
// caught even if HealthResult.OK() itself stays correct.

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/health"
)

// captureStderr is defined in deprecation_wiring_test.go and reused here:
// ui.Warn writes to the real os.Stderr, which is what printServiceDetails's
// "✗" lines go through.

// TestPrintServiceDetails_AgreesWithAggregate_RunningService is the exact
// contradiction filed in issue #268: a service with no Docker healthcheck
// reports Status == "running", which the aggregate counts as healthy. This
// test builds a report the same way the aggregate would (every result that
// is .OK() increments Healthy, matching "4/4 healthy (100%)"), then asserts
// printServiceDetails — in both non-verbose and verbose mode — does NOT mark
// that same "running" service with a "✗" in stderr. Aggregate and
// per-service verdicts must agree for every result, not just for "healthy".
func TestPrintServiceDetails_AgreesWithAggregate_RunningService(t *testing.T) {
	results := []health.HealthResult{
		{Service: "postgres", Status: "healthy"},
		{Service: "hasura", Status: "healthy"},
		{Service: "auth", Status: "healthy"},
		{Service: "nginx", Status: "running"}, // no healthcheck declared
	}

	report := &health.HealthReport{
		Results: results,
		Total:   len(results),
	}
	for _, r := range results {
		if r.OK() {
			report.Healthy++
		} else {
			report.Unhealthy++
		}
	}

	// Sanity check on the fixture itself: this must reproduce the issue's
	// "4/4 healthy (100%)" aggregate before we even get to the printer.
	if report.Healthy != 4 || report.Unhealthy != 0 {
		t.Fatalf("fixture setup: got %d/%d healthy, want 4/4", report.Healthy, report.Total)
	}

	for _, verbose := range []bool{false, true} {
		out := captureStderr(t, func() {
			printServiceDetails(report, verbose)
		})
		if strings.Contains(out, "nginx") {
			t.Errorf("verbose=%v: printServiceDetails wrote a warning for nginx (Status=running), "+
				"contradicting the aggregate which counts it healthy; stderr was:\n%s", verbose, out)
		}
	}
}
