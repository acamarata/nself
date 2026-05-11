// Package doctor — ops_drill.go: OPS-DRILL-01 check.
//
// S9.T03 + S9.T15 (P100 v1.1.0): Verifies that a successful `nself backup
// drill` ran within the last 7 days. The drill log lives at
// <projectDir>/.nself/drill-log.json (JSON Lines format, written by
// internal/backup.Drill). The check is read-only and never invokes the drill
// itself — that would be a privileged operation gated by SSH.
//
// Status semantics:
//   - PASS: most recent successful drill within 7 days
//   - WARN: most recent successful drill 7–14 days old (operator nudge)
//   - FAIL: no successful drill found OR most recent > 14 days
//
// The fix-cmd suggests `nself backup drill` so operators can run it manually
// or wire the weekly cron via scripts/dr-drill.sh.
package doctor

import (
	"context"
	"fmt"
	"time"

	"github.com/nself-org/cli/internal/database"
)

// CheckOPSDrill returns the OPS-DRILL-01 result for projectDir. The function
// is pure read-only — it never mutates filesystem state.
func CheckOPSDrill(_ context.Context, projectDir string) CheckResult {
	const checkID = "OPS-DRILL-01"
	const warnAge = 7 * 24 * time.Hour
	const failAge = 14 * 24 * time.Hour

	last, ok := database.LastSuccessfulDrill(projectDir)
	if !ok {
		return CheckResult{
			Section: "operations",
			Name:    checkID,
			Status:  "fail",
			Message: "OPS-DRILL-01: no successful backup drill recorded — run weekly drill",
			FixCmd:  "nself backup drill",
		}
	}

	age := time.Since(last.CompletedAt)
	switch {
	case age <= warnAge:
		return CheckResult{
			Section: "operations",
			Name:    checkID,
			Status:  "pass",
			Message: fmt.Sprintf("OPS-DRILL-01: last successful drill %s ago (RTO %.1fh, target met=%v)",
				age.Round(time.Minute), last.TotalDuration.Hours(), last.RTOTargetMet),
		}
	case age <= failAge:
		return CheckResult{
			Section: "operations",
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("OPS-DRILL-01: last successful drill %s ago — schedule a fresh run within 7 days",
				age.Round(time.Hour)),
			FixCmd: "nself backup drill",
		}
	default:
		return CheckResult{
			Section: "operations",
			Name:    checkID,
			Status:  "fail",
			Message: fmt.Sprintf("OPS-DRILL-01: last successful drill %s ago (>14 days) — run drill ASAP",
				age.Round(time.Hour)),
			FixCmd: "nself backup drill",
		}
	}
}
