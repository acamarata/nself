package onboarding

// Purpose: the six individual stage-check implementations plus telemetry emission and next-action guidance, backing RunFunnel in funnel.go.
// Inputs: skip flags and prior-stage StageResult values for stages that depend on earlier state.
// Outputs: a StageResult per stage, and an emitted telemetry event on pass.
// Constraints: split out of funnel.go as a pure move (CLI-R12); no behavior change.

import (
	"fmt"
	"runtime"
	"time"

	"github.com/nself-org/cli/internal/telemetry"
	nselfversion "github.com/nself-org/cli/internal/version"
)

// ── Stage checks ────────────────────────────────────────────────────────────

func checkStage1Install() StageResult {
	ver := nselfversion.GetVersion()
	meta := map[string]interface{}{
		"version": ver,
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	}
	return StageResult{
		ID:       1,
		Name:     "Install",
		Status:   StatusPass,
		Message:  fmt.Sprintf("v%s (%s/%s)", ver, runtime.GOOS, runtime.GOARCH),
		Metadata: meta,
	}
}

func checkStage2Activation(skip bool) StageResult {
	if skip {
		return skipped(2, "Activation")
	}
	count := ProjectsCount()
	if count == 0 {
		return StageResult{
			ID:          2,
			Name:        "Activation",
			Status:      StatusFail,
			Message:     "no projects initialized",
			Remediation: "Run: nself init",
		}
	}
	noun := "project"
	if count != 1 {
		noun = "projects"
	}
	return StageResult{
		ID:       2,
		Name:     "Activation",
		Status:   StatusPass,
		Message:  fmt.Sprintf("%d %s initialized", count, noun),
		Metadata: map[string]interface{}{"projects_count": count},
	}
}

func checkStage3FirstUse(skip bool, s1 StageResult) StageResult {
	if skip {
		return skipped(3, "First-use")
	}
	t := LastStartTime()
	if t.IsZero() {
		return StageResult{
			ID:          3,
			Name:        "First-use",
			Status:      StatusFail,
			Message:     "nself start has not been run yet",
			Remediation: "Run: nself start  (from an initialized project directory)",
		}
	}
	daysAgo := int(time.Since(t).Hours() / 24)
	var durationStr string
	switch {
	case daysAgo == 0:
		durationStr = "today"
	case daysAgo == 1:
		durationStr = "1 day ago"
	default:
		durationStr = fmt.Sprintf("%d days ago", daysAgo)
	}

	// Compute time-to-start delta if install-id was recorded with a timestamp.
	// For now emit 0 as a safe default (Q07 may not be present).
	ttStartSec := 0

	return StageResult{
		ID:      3,
		Name:    "First-use",
		Status:  StatusPass,
		Message: fmt.Sprintf("first start %s", durationStr),
		Metadata: map[string]interface{}{
			"time_to_start_seconds": ttStartSec,
		},
	}
}

func checkStage4FirstPlugin(skip bool) StageResult {
	if skip {
		return skipped(4, "First-plugin")
	}
	name, tier := FirstInstalledPlugin()
	if name == "" {
		return StageResult{
			ID:          4,
			Name:        "First-plugin",
			Status:      StatusFail,
			Message:     "no plugins installed",
			Remediation: "Run: nself plugin install ai  (or any plugin)",
		}
	}
	return StageResult{
		ID:      4,
		Name:    "First-plugin",
		Status:  StatusPass,
		Message: fmt.Sprintf("first plugin: %s (%s)", name, tier),
		Metadata: map[string]interface{}{
			"plugin_name": name,
			"tier":        tier,
		},
	}
}

func checkStage5FirstValue(skip bool) StageResult {
	if skip {
		return skipped(5, "First-value")
	}
	count := QueryCount()
	if count < 0 {
		// File absent — Hasura hook not wired (self-hosted without telemetry).
		return StageResult{
			ID:      5,
			Name:    "First-value",
			Status:  StatusUnknown,
			Message: "query-count file not found (Hasura telemetry hook not wired)",
		}
	}
	if count == 0 {
		return StageResult{
			ID:          5,
			Name:        "First-value",
			Status:      StatusFail,
			Message:     "no Hasura queries logged yet",
			Remediation: "Make a GraphQL query via your app or: nself status",
		}
	}
	return StageResult{
		ID:      5,
		Name:    "First-value",
		Status:  StatusPass,
		Message: fmt.Sprintf("%d query/queries logged", count),
		Metadata: map[string]interface{}{
			"query_count": count,
		},
	}
}

func checkStage6Habit(skip bool) StageResult {
	if skip {
		return skipped(6, "Habit")
	}
	starts := RecentStartCount()
	if starts < 3 {
		return StageResult{
			ID:          6,
			Name:        "Habit",
			Status:      StatusFail,
			Message:     fmt.Sprintf("%d start(s) in last 7 days (need 3+)", starts),
			Remediation: "Keep using nself start — habit forms after regular use",
		}
	}
	return StageResult{
		ID:      6,
		Name:    "Habit",
		Status:  StatusPass,
		Message: fmt.Sprintf("%d starts in last 7 days", starts),
		Metadata: map[string]interface{}{
			"starts_last_7d": starts,
		},
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func skipped(id int, name string) StageResult {
	return StageResult{
		ID:      id,
		Name:    name,
		Status:  StatusSkipped,
		Message: "skipped, prior stage failed",
	}
}

func emitTelemetry(stage int, event string, meta map[string]interface{}) {
	if !telemetry.IsOptedIn() {
		return
	}
	props := map[string]interface{}{
		"stage":      stage,
		"install_id": InstallID(),
	}
	for k, v := range meta {
		props[k] = v
	}
	telemetry.Send(event, props)
}

// nextActionFromPosition returns the human-readable next step message.
func nextActionFromPosition(pos int, stages []StageResult) string {
	if pos == 6 {
		return "You have completed the full onboarding funnel."
	}
	// Find the first non-pass, non-skipped stage with a remediation hint
	for _, s := range stages {
		if s.Status == StatusFail && s.Remediation != "" {
			return s.Remediation
		}
	}
	if pos == 0 {
		return "Install nself and run nself init to get started."
	}
	return fmt.Sprintf("Complete stage %d/%d to continue.", pos+1, 6)
}
