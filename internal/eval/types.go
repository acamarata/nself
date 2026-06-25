package eval

// Purpose: Shared types for nself-eval-gate CLI client layer.
//   Mirrors the JSON contract of nself-eval-gate HTTP API responses.
//   schema_ver on EvalRunResult is checked on decode to surface breaking changes early.
// Inputs:  JSON responses from nself-eval-gate (port 3770).
// Outputs: Typed Go structs consumed by eval.go / eval_gate.go commands.
// Constraints: schema_ver must match expectedSchemaVer; mismatch → hard error to caller.
// SPORT: CLI-CMD-EVAL-001

// expectedSchemaVer is the schema version this client understands.
// Bump only when nself-eval-gate increments its schema_ver field.
const expectedSchemaVer = 1

// EvalRunResult is the response from GET /eval/runs/{id} after completion.
type EvalRunResult struct {
	// SchemaVer is the schema version of this response; checked against expectedSchemaVer.
	SchemaVer int `json:"schema_ver"`
	// ID is the unique run identifier.
	ID string `json:"id"`
	// SuiteSlug is the evaluated suite's slug.
	SuiteSlug string `json:"suite_slug"`
	// Status is one of: queued, running, passed, failed.
	Status string `json:"status"`
	// PassRate is the fraction of tasks that passed (0.0–1.0).
	PassRate float64 `json:"pass_rate"`
	// SuiteScore is the weighted mean score across all tasks (0.0–1.0).
	SuiteScore float64 `json:"suite_score"`
	// Passed indicates whether the run met the configured threshold.
	Passed bool `json:"passed"`
	// Tasks holds per-task result details.
	Tasks []EvalTaskResult `json:"tasks"`
	// PreconditionFailed is true when a dependency (BGE-M3, gateway) was unavailable.
	PreconditionFailed bool `json:"precondition_failed"`
	// ErrorMessage is populated when Status is "failed" with a human-readable cause.
	ErrorMessage string `json:"error_message,omitempty"`
}

// EvalTaskResult is the per-task breakdown within an EvalRunResult.
type EvalTaskResult struct {
	// TaskID is the unique task identifier.
	TaskID string `json:"task_id"`
	// Input is the query string used for this task.
	Input string `json:"input"`
	// Output is the system output evaluated.
	Output string `json:"output"`
	// Score is the numeric score (0.0–1.0).
	Score float64 `json:"score"`
	// Passed indicates whether this individual task passed.
	Passed bool `json:"passed"`
	// ScoringMode is one of: exact, semantic, rubric.
	ScoringMode string `json:"scoring_mode"`
	// Rationale is provided for rubric-scored tasks.
	Rationale string `json:"rationale,omitempty"`
}

// GateStatus is the response from GET /eval/gate/{tier}.
type GateStatus struct {
	// Tier is the autonomy tier checked: supervised, semi-auto, full-auto.
	Tier string `json:"tier"`
	// Cleared is true when all required suites pass their thresholds.
	Cleared bool `json:"cleared"`
	// BlockingSuites lists suite slugs that are failing or have no runs.
	BlockingSuites []string `json:"blocking_suites"`
	// Enforced mirrors the threshold's enforced flag.
	Enforced bool `json:"enforced"`
}

// RunRequest is sent to POST /eval/run.
type RunRequest struct {
	// SuiteSlug is the slug of the suite to run.
	SuiteSlug string `json:"suite_slug"`
}

// RunQueued is the 202 response from POST /eval/run.
type RunQueued struct {
	// RunID is the identifier to poll at GET /eval/runs/{id}.
	RunID string `json:"run_id"`
	// Status is always "queued" on the initial response.
	Status string `json:"status"`
}

// ValidationResult is the response from POST /eval/validate.
type ValidationResult struct {
	// Valid is true when the submitted YAML passes schema validation.
	Valid bool `json:"valid"`
	// Errors lists field-level validation errors.
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidationError is a single schema validation failure.
type ValidationError struct {
	// Field is the JSON-path to the failing field.
	Field string `json:"field"`
	// Message is a human-readable description of the failure.
	Message string `json:"message"`
}
