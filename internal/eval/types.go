// Package eval provides CLI-side types and HTTP client for the nself-eval-gate plugin API.
//
// Purpose: Typed representation of nself-eval-gate HTTP responses consumed by
// `nself ci eval` and `nself ci eval gate` commands. Types mirror the plugin's
// JSON API contract (eval-harness-foundation-spec.md §6).
// Inputs: JSON payloads from nself-eval-gate HTTP endpoints.
// Outputs: Strongly-typed Go structs for CLI command logic and output formatters.
// Constraints: schema_ver field on EvalRunResult must be checked before parsing;
// breaking schema changes must surface as errors, not silent parse failures.
// SPORT: F02-COMMAND-INVENTORY.md (nself ci eval + nself ci eval gate entries).
package eval

import "time"

// ErrPreconditionNotMet is returned by the client when the plugin indicates
// a precondition is not met (e.g., BGE-M3 not wired). CLI exits with code 3.
const ErrPreconditionNotMet = evalError("precondition_not_met: BGE-M3 or plugin-retrieval unavailable")

// evalError is an unexported string type for sentinel errors in this package.
type evalError string

func (e evalError) Error() string { return string(e) }

// ErrPluginUnavailable is returned when the nself-eval-gate plugin cannot be reached.
// CLI exits with code 2.
const ErrPluginUnavailable = evalError("plugin_unavailable: nself-eval-gate not responding")

// RunRequest is the JSON body sent to POST /eval/run.
// Purpose: Trigger an eval suite run on the nself-eval-gate plugin.
// Fields: SuiteSlug (required), Repo (detected from cwd), K for recall@k metrics.
type RunRequest struct {
	SuiteSlug string `json:"suite_slug"`
	Repo      string `json:"repo,omitempty"`
	CommitSHA string `json:"commit_sha,omitempty"`
	Branch    string `json:"branch,omitempty"`
	K         int    `json:"k,omitempty"` // default 3 per spec §11
}

// RunResponse is the JSON body returned by POST /eval/run.
// Purpose: Carry the queued run ID for the caller to poll via GetRun.
type RunResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"` // "queued"
}

// TaskResult is a single task's scoring outcome within an eval run.
// Purpose: Per-task score breakdown emitted in the eval-results.json artifact.
// Fields: ID matches the task YAML id; Metrics holds per-metric 0-1 float values.
type TaskResult struct {
	ID        string             `json:"id"`
	Score     float64            `json:"score"`
	Metrics   map[string]float64 `json:"metrics"`
	Passed    bool               `json:"passed"`
	Rationale string             `json:"rationale,omitempty"` // rubric mode only
}

// EvalRunResult is the full run result returned by GET /eval/runs/{id}.
// Purpose: Carry all eval run data to the CLI for artifact writing and table display.
// Constraints: SchemaVer must be checked before consuming; increment breaks compat.
// Fields: Passed=true when SuiteScore >= suite threshold AND all required tasks pass.
type EvalRunResult struct {
	RunID      string       `json:"run_id"`
	Suite      string       `json:"suite"`
	Commit     string       `json:"commit,omitempty"`
	PassRate   float64      `json:"pass_rate"`
	SuiteScore float64      `json:"suite_score"`
	Passed     bool         `json:"passed"`
	Tasks      []TaskResult `json:"tasks"`
	DurationMS int          `json:"duration_ms,omitempty"`
	SchemaVer  string       `json:"schema_ver"` // must be "1"; error on mismatch
	CreatedAt  time.Time    `json:"created_at,omitempty"`
}

// RunStatus represents possible states of an eval run returned by GET /eval/runs/{id}.
type RunStatus string

const (
	RunStatusQueued  RunStatus = "queued"
	RunStatusRunning RunStatus = "running"
	RunStatusPassed  RunStatus = "passed"
	RunStatusFailed  RunStatus = "failed"
)

// GateStatus is the JSON body returned by GET /eval/gate/{tier}.
// Purpose: Report whether an autonomy tier is cleared by the latest eval runs.
// BlockingSuites lists slugs of suites that have not cleared their thresholds.
type GateStatus struct {
	Tier           string   `json:"tier"`
	Cleared        bool     `json:"cleared"`
	BlockingSuites []string `json:"blocking_suites,omitempty"`
	MinPassRate    float64  `json:"min_pass_rate,omitempty"`
	MinSuiteScore  float64  `json:"min_suite_score,omitempty"`
}

// ValidateResponse is returned by POST /eval/validate.
// Purpose: Report YAML validation result to `nself ci eval --validate-only`.
// Errors lists field-level validation failures; empty when Valid=true.
type ValidateResponse struct {
	Valid  bool            `json:"valid"`
	Errors []ValidateError `json:"errors,omitempty"`
}

// ValidateError is a single validation error from POST /eval/validate.
type ValidateError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// EvalResultsArtifact is the schema for the eval-results.json artifact written
// to {repo}/.nself-ci/artifacts/ on run completion (spec §6 CLI output format).
// Purpose: Machine-readable artifact for CI consumers downstream of `nself ci eval`.
type EvalResultsArtifact struct {
	RunID      string       `json:"run_id"`
	Suite      string       `json:"suite"`
	Commit     string       `json:"commit,omitempty"`
	PassRate   float64      `json:"pass_rate"`
	SuiteScore float64      `json:"suite_score"`
	Passed     bool         `json:"passed"`
	Tasks      []TaskResult `json:"tasks"`
	DurationMS int          `json:"duration_ms,omitempty"`
}
