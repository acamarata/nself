package eval

// client.go — typed HTTP client for the nself-eval-gate plugin API.
//
// Purpose: Provide typed methods to call nself-eval-gate HTTP endpoints from the
// CLI `nself ci eval` command group. Handles auth header injection, JSON encode/
// decode, poll loop for async run completion, and error classification.
// Inputs: baseURL (default: http://localhost:3770), NSELF_EVAL_GATE_TOKEN env var
// for the X-Nself-Plugin-Token auth header (empty = unauthenticated dev mode).
// Outputs: typed structs per endpoint; sentinel errors ErrPreconditionNotMet and
// ErrPluginUnavailable for CLI exit-code mapping.
// Constraints: poll loop uses 2s interval with hard 10-minute ceiling per spec §6;
// timeout returns error, never false. No goroutines — all calls are synchronous.
// SPORT: F02-COMMAND-INVENTORY.md (nself ci eval + nself ci eval gate).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// pollInterval is the delay between GET /eval/runs/{id} polling attempts.
	// Spec §6: poll every 2 seconds.
	pollInterval = 2 * time.Second

	// pollTimeout is the hard ceiling for polling a single run.
	// Spec §6: 10-minute maximum; return error (not false) on expiry.
	pollTimeout = 10 * time.Minute

	// expectedSchemaVer is the only accepted schema version for EvalRunResult.
	// A mismatch surfaces as an error so breaking schema changes are caught early.
	expectedSchemaVer = "1"
)

// Client is a typed HTTP client for the nself-eval-gate plugin API.
//
// Purpose: Encapsulate all plugin HTTP communication behind typed methods so
// cobra command files stay thin. Thread-safe (http.Client is reused).
// Inputs: BaseURL (defaults to http://localhost:3770 per ports.EvalGatePort).
// Outputs: typed response structs; error on HTTP failure or non-2xx status.
// Constraints: token is read from NSELF_EVAL_GATE_TOKEN env var at call time;
// rotate the token without restarting the CLI.
type Client struct {
	// BaseURL is the nself-eval-gate root URL, e.g. "http://localhost:3770".
	BaseURL    string
	httpClient *http.Client
}

// NewClient returns a Client targeting baseURL.
// Purpose: Constructor with sensible defaults — 30s overall timeout per call
// (separate from the poll loop timeout, which is managed by GetRun).
// Inputs: baseURL — empty string falls back to http://localhost:3770.
// Outputs: initialized *Client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:3770"
	}
	return &Client{
		BaseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// authToken returns the plugin JWT token from the environment.
// Empty string = no auth header (allowed for local dev without plugin auth).
func authToken() string {
	return os.Getenv("NSELF_EVAL_GATE_TOKEN")
}

// do executes an HTTP request, sets the auth header, and returns the response body.
// Purpose: Single point for auth injection, error classification, and response
// body reading across all client methods.
// Inputs: *http.Request — caller builds it with the correct method/URL/body.
// Outputs: raw []byte body; error on transport failure, non-2xx, or 503.
// Constraints: 503 with semantic_scoring_unavailable body → ErrPreconditionNotMet.
// Any connection-refused-class error → ErrPluginUnavailable.
func (c *Client) do(req *http.Request) ([]byte, error) {
	if tok := authToken(); tok != "" {
		req.Header.Set("X-Nself-Plugin-Token", tok)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrPluginUnavailable
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eval client: reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		// Spec §6: 503 with semantic_scoring_unavailable = precondition not met.
		var errBody struct {
			Error        string `json:"error"`
			Precondition string `json:"precondition"`
		}
		_ = json.Unmarshal(body, &errBody)
		if errBody.Error == "semantic_scoring_unavailable" {
			return nil, ErrPreconditionNotMet
		}
		return nil, ErrPluginUnavailable
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("eval client: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// RunSuite starts an eval suite run via POST /eval/run.
//
// Purpose: Trigger async suite execution on the nself-eval-gate plugin; returns
// the run ID for subsequent GetRun polling.
// Inputs: ctx for cancellation; req with SuiteSlug (required), Repo, CommitSHA, Branch, K.
// Outputs: RunResponse with RunID and initial status "queued"; error on failure.
// Constraints: does not block; caller must call GetRun to wait for completion.
func (c *Client) RunSuite(ctx context.Context, req RunRequest) (RunResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return RunResponse{}, fmt.Errorf("eval client: marshaling run request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/eval/run", bytes.NewReader(body))
	if err != nil {
		return RunResponse{}, fmt.Errorf("eval client: building run request: %w", err)
	}

	respBody, err := c.do(httpReq)
	if err != nil {
		return RunResponse{}, err
	}

	var result RunResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return RunResponse{}, fmt.Errorf("eval client: parsing run response: %w", err)
	}
	return result, nil
}

// GetRun polls GET /eval/runs/{id} at 2-second intervals until the run reaches
// a terminal state (passed or failed) or the 10-minute timeout expires.
//
// Purpose: Block the CLI until an async eval run completes, surfacing progress
// to the caller via the returned EvalRunResult.
// Inputs: ctx for cancellation (caller can add shorter deadline); id = run UUID.
// Outputs: EvalRunResult on terminal state; error on timeout or transport failure.
// Constraints:
//   - Poll every 2s (pollInterval); hard ceiling 10min (pollTimeout).
//   - Timeout → return error (never return a false/zero result silently).
//   - schema_ver != "1" → return error (breaking schema change).
func (c *Client) GetRun(ctx context.Context, id string) (EvalRunResult, error) {
	deadline := time.Now().Add(pollTimeout)
	url := fmt.Sprintf("%s/eval/runs/%s", c.BaseURL, id)

	for {
		if time.Now().After(deadline) {
			return EvalRunResult{}, fmt.Errorf("eval client: poll timeout after 10 minutes for run %s", id)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return EvalRunResult{}, fmt.Errorf("eval client: building get-run request: %w", err)
		}

		respBody, err := c.do(httpReq)
		if err != nil {
			return EvalRunResult{}, err
		}

		// Parse into a map first to check status before full decode.
		var raw struct {
			Status    RunStatus `json:"status"`
			SchemaVer string    `json:"schema_ver"`
		}
		if err := json.Unmarshal(respBody, &raw); err != nil {
			return EvalRunResult{}, fmt.Errorf("eval client: parsing run status: %w", err)
		}

		// Terminal states: parse the full result.
		if raw.Status == RunStatusPassed || raw.Status == RunStatusFailed {
			var result EvalRunResult
			if err := json.Unmarshal(respBody, &result); err != nil {
				return EvalRunResult{}, fmt.Errorf("eval client: parsing run result: %w", err)
			}
			// Check schema_ver before returning — breaking schema change must surface.
			if result.SchemaVer != expectedSchemaVer && result.SchemaVer != "" {
				return EvalRunResult{}, fmt.Errorf("eval client: unexpected schema_ver %q (want %q); update CLI", result.SchemaVer, expectedSchemaVer)
			}
			return result, nil
		}

		// Not terminal — wait before next poll, respecting context cancellation.
		select {
		case <-ctx.Done():
			return EvalRunResult{}, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// ValidateYAML sends YAML content to POST /eval/validate for schema validation.
//
// Purpose: Implement `nself ci eval --validate-only` without running eval tasks.
// Inputs: ctx; yamlContent — raw YAML bytes of the eval set file.
// Outputs: ValidateResponse with Valid flag and field-level errors; error on transport.
// Constraints: does not trigger a run; does not require BGE-M3.
func (c *Client) ValidateYAML(ctx context.Context, yamlContent []byte) (ValidateResponse, error) {
	reqBody, err := json.Marshal(map[string]string{
		"yaml_content": string(yamlContent),
	})
	if err != nil {
		return ValidateResponse{}, fmt.Errorf("eval client: marshaling validate request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/eval/validate", bytes.NewReader(reqBody))
	if err != nil {
		return ValidateResponse{}, fmt.Errorf("eval client: building validate request: %w", err)
	}

	respBody, err := c.do(httpReq)
	if err != nil {
		return ValidateResponse{}, err
	}

	var result ValidateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return ValidateResponse{}, fmt.Errorf("eval client: parsing validate response: %w", err)
	}
	return result, nil
}

// GetGateStatus calls GET /eval/gate/{tier} to check autonomy-tier clearance.
//
// Purpose: Implement `nself ci eval gate --tier <tier>` — determine whether a
// tier's required suites have all cleared their configured thresholds.
// Inputs: ctx; tier — one of "supervised", "semi-auto", "full-auto".
// Outputs: GateStatus with Cleared flag and BlockingSuites list.
// Constraints: 503 → ErrPreconditionNotMet (BGE-M3 needed but not wired).
func (c *Client) GetGateStatus(ctx context.Context, tier string) (GateStatus, error) {
	url := fmt.Sprintf("%s/eval/gate/%s", c.BaseURL, tier)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return GateStatus{}, fmt.Errorf("eval client: building gate request: %w", err)
	}

	respBody, err := c.do(httpReq)
	if err != nil {
		return GateStatus{}, err
	}

	var result GateStatus
	if err := json.Unmarshal(respBody, &result); err != nil {
		return GateStatus{}, fmt.Errorf("eval client: parsing gate response: %w", err)
	}
	return result, nil
}
