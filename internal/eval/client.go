package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Purpose: Typed HTTP client for the nself-eval-gate plugin (port 3770).
//   Wraps all eval API calls and handles poll loop for run completion.
// Inputs:  BaseURL of the eval-gate plugin; optional HTTPClient for testing.
// Outputs: Typed result structs (EvalRunResult, GateStatus, ValidationResult).
// Constraints: Poll loop: 2s interval, 10min hard ceiling. Timeout = error (not false).
//   ErrPreconditionNotMet returned when plugin signals precondition_failed=true.
//   ErrSchemaVersion returned when schema_ver mismatch detected.
// SPORT: CLI-CMD-EVAL-001

const (
	// pollInterval is the wait between GET /eval/runs/{id} calls.
	pollInterval = 2 * time.Second
	// pollTimeout is the hard ceiling for the poll loop.
	pollTimeout = 10 * time.Minute
)

// ErrPreconditionNotMet is returned when BGE-M3 or AI gateway is unavailable.
// CLI exits with code 3 on this error.
var ErrPreconditionNotMet = errors.New("eval precondition not met (BGE-M3 or AI gateway unavailable)")

// ErrSchemaVersion is returned when the plugin response uses an unknown schema_ver.
var ErrSchemaVersion = errors.New("eval schema version mismatch — plugin may need upgrade")

// ErrPollTimeout is returned when a run does not complete within pollTimeout.
var ErrPollTimeout = errors.New("eval run polling timed out after 10 minutes")

// Client is the typed HTTP client for nself-eval-gate.
type Client struct {
	// BaseURL is the plugin base URL, e.g. "http://localhost:3770".
	BaseURL string
	// HTTP is the underlying client; set to &http.Client{} if nil.
	HTTP *http.Client
	// SourceAccountID is forwarded as X-Nself-Source-Account-Id header.
	SourceAccountID string
}

// newHTTPClient returns the HTTP client, initialising a default if needed.
func (c *Client) newHTTPClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// do executes an HTTP request, decoding the JSON response into dst.
// Purpose: Centralise request construction + header injection + error surfacing.
// Inputs:  ctx, method, path (relative), body (nil for GET), dst (decode target).
// Outputs: error if non-2xx or JSON decode failure.
func (c *Client) do(ctx context.Context, method, path string, body any, dst any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("eval client marshal: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("eval client build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.SourceAccountID != "" {
		req.Header.Set("X-Nself-Source-Account-Id", c.SourceAccountID)
	}

	resp, err := c.newHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("eval client request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("eval client read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("eval plugin returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	if dst != nil {
		if err := json.Unmarshal(respBytes, dst); err != nil {
			return fmt.Errorf("eval client decode response: %w", err)
		}
	}
	return nil
}

// RunSuite triggers a suite evaluation via POST /eval/run.
// Purpose: Queue a new eval run for the specified suite slug.
// Inputs:  ctx, suiteSlug — the slug registered in np_eval_suites.
// Outputs: RunQueued{RunID, Status:"queued"} or error.
// Constraints: Does not wait for completion — call GetRun to poll.
func (c *Client) RunSuite(ctx context.Context, suiteSlug string) (RunQueued, error) {
	var result RunQueued
	if err := c.do(ctx, http.MethodPost, "/eval/run", RunRequest{SuiteSlug: suiteSlug}, &result); err != nil {
		return RunQueued{}, err
	}
	return result, nil
}

// GetRun fetches a single run by ID via GET /eval/runs/{id}.
// Purpose: Single-fetch (no polling) — use for inspection; poll via WaitForRun.
// Inputs:  ctx, runID.
// Outputs: EvalRunResult; ErrSchemaVersion if schema_ver doesn't match.
// Constraints: Returns nil error + zero-value if run not found (404).
func (c *Client) GetRun(ctx context.Context, runID string) (EvalRunResult, error) {
	var result EvalRunResult
	if err := c.do(ctx, http.MethodGet, "/eval/runs/"+runID, nil, &result); err != nil {
		return EvalRunResult{}, err
	}
	if result.SchemaVer != 0 && result.SchemaVer != expectedSchemaVer {
		return EvalRunResult{}, fmt.Errorf("%w: got %d, expected %d", ErrSchemaVersion, result.SchemaVer, expectedSchemaVer)
	}
	return result, nil
}

// WaitForRun polls GET /eval/runs/{id} every 2s until the run reaches a terminal state.
// Purpose: Block CLI until eval run completes; enforce 10min hard ceiling.
// Inputs:  ctx, runID.
// Outputs: Final EvalRunResult; ErrPollTimeout after 10min; ErrPreconditionNotMet
//
//	if the run signals precondition_failed=true.
//
// Constraints: Terminal states: "passed", "failed". "queued"/"running" → keep polling.
func (c *Client) WaitForRun(ctx context.Context, runID string) (EvalRunResult, error) {
	deadline := time.Now().Add(pollTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return EvalRunResult{}, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return EvalRunResult{}, ErrPollTimeout
			}

			result, err := c.GetRun(ctx, runID)
			if err != nil {
				return EvalRunResult{}, err
			}

			switch result.Status {
			case "passed", "failed":
				if result.PreconditionFailed {
					return result, ErrPreconditionNotMet
				}
				return result, nil
			// queued, running — keep polling
			default:
				continue
			}
		}
	}
}

// ValidateYAML validates eval-set YAML via POST /eval/validate.
// Purpose: Dry-run schema validation without executing a run.
// Inputs:  ctx, yamlContent — raw bytes of the eval-set YAML file.
// Outputs: ValidationResult{Valid, Errors}; non-nil error on HTTP/network failure only.
// Constraints: Validation errors are in ValidationResult.Errors, not as Go errors.
func (c *Client) ValidateYAML(ctx context.Context, yamlContent []byte) (ValidationResult, error) {
	var result ValidationResult
	payload := map[string]string{"content": string(yamlContent)}
	if err := c.do(ctx, http.MethodPost, "/eval/validate", payload, &result); err != nil {
		return ValidationResult{}, err
	}
	return result, nil
}

// GetGateStatus fetches the gate check result for a given autonomy tier.
// Purpose: Determine whether a tier is cleared for AI autonomy progression.
// Inputs:  ctx, tier — one of: supervised, semi-auto, full-auto.
// Outputs: GateStatus{Tier, Cleared, BlockingSuites, Enforced}.
// Constraints: supervised tier always returns Cleared=true (enforced=false by design).
func (c *Client) GetGateStatus(ctx context.Context, tier string) (GateStatus, error) {
	var result GateStatus
	if err := c.do(ctx, http.MethodGet, "/eval/gate/"+tier, nil, &result); err != nil {
		return GateStatus{}, err
	}
	return result, nil
}
