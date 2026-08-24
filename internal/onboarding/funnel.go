// Package onboarding implements the 6-stage onboarding funnel check for
// nself doctor --install-check.
//
// Each stage is checked in order; a failed stage marks subsequent stages as
// skipped. Telemetry events are fired for each PASS via the telemetry package
// (fire-and-forget, opt-in only).
package onboarding

// StageStatus is the result of a single funnel stage check.
type StageStatus string

const (
	StatusPass    StageStatus = "pass"
	StatusFail    StageStatus = "fail"
	StatusUnknown StageStatus = "unknown" // file absent, cannot determine
	StatusSkipped StageStatus = "skipped" // prior stage failed
)

// StageResult is the outcome of one funnel stage.
type StageResult struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Status      StageStatus            `json:"status"`
	Message     string                 `json:"message"`
	Remediation string                 `json:"remediation,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FunnelReport is the full 6-stage funnel result.
type FunnelReport struct {
	Stages   []StageResult `json:"stages"`
	Position int           `json:"funnel_position"` // highest passing stage (1-6), 0 if none
	Next     string        `json:"next_action"`
}

// RunFunnel executes all 6 stages and returns the report.
// Telemetry events are fired for each passing stage (no-op when opt-out).
func RunFunnel() FunnelReport {
	report := FunnelReport{}
	skipping := false

	// Stage 1 — Install
	s1 := checkStage1Install()
	report.Stages = append(report.Stages, s1)
	if s1.Status == StatusPass {
		report.Position = 1
		emitTelemetry(1, "onboard.install", s1.Metadata)
	} else {
		skipping = true
	}

	// Stage 2 — Activation
	s2 := checkStage2Activation(skipping)
	report.Stages = append(report.Stages, s2)
	if s2.Status == StatusPass {
		report.Position = 2
		emitTelemetry(2, "onboard.activation", s2.Metadata)
	} else if s2.Status == StatusFail {
		skipping = true
	}

	// Stage 3 — First-use
	s3 := checkStage3FirstUse(skipping, s1)
	report.Stages = append(report.Stages, s3)
	if s3.Status == StatusPass {
		report.Position = 3
		emitTelemetry(3, "onboard.first_use", s3.Metadata)
	} else if s3.Status == StatusFail {
		skipping = true
	}

	// Stage 4 — First-plugin
	s4 := checkStage4FirstPlugin(skipping)
	report.Stages = append(report.Stages, s4)
	if s4.Status == StatusPass {
		report.Position = 4
		emitTelemetry(4, "onboard.first_plugin", s4.Metadata)
	} else if s4.Status == StatusFail {
		skipping = true
	}

	// Stage 5 — First-value
	s5 := checkStage5FirstValue(skipping)
	report.Stages = append(report.Stages, s5)
	if s5.Status == StatusPass {
		report.Position = 5
		emitTelemetry(5, "onboard.first_value", s5.Metadata)
	} else if s5.Status == StatusFail {
		skipping = true
	}

	// Stage 6 — Habit
	s6 := checkStage6Habit(skipping)
	report.Stages = append(report.Stages, s6)
	if s6.Status == StatusPass {
		report.Position = 6
		emitTelemetry(6, "onboard.habit", s6.Metadata)
	}

	report.Next = nextActionFromPosition(report.Position, report.Stages)
	return report
}
