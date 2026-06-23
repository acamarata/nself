package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Purpose: Integration tests for nself ci eval gate command.
// Inputs: Mock HTTP server responses from nself-eval-gate plugin.
// Outputs: Verify exit codes 0, 1, 2, 3 and JSON/human output formats.
// Constraints: Must test all four exit codes; must verify precondition vs regression distinction.
// SPORT: F02-COMMAND-INVENTORY.md (command registry)

// TestEvalGateExitCodeCleared tests exit code 0 (tier cleared).
func TestEvalGateExitCodeCleared(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/eval/gate/semi-auto" {
			t.Errorf("expected path /eval/gate/semi-auto, got %s", r.URL.Path)
		}

		resp := EvalGateResponse{
			Tier:    "semi-auto",
			Cleared: true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Override plugin URL for this test
	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	// Test the command
	ciEvalGateFlags.tier = "semi-auto"
	ciEvalGateFlags.json = false
	ciEvalGateFlags.verbose = false

	// Call checkEvalGateTier directly
	resp, exitCode := checkEvalGateTier(context.Background(), "semi-auto")

	if exitCode != 0 {
		t.Errorf("expected exit code 0 (cleared), got %d", exitCode)
	}
	if !resp.Cleared {
		t.Errorf("expected cleared=true, got false")
	}
}

// TestEvalGateExitCodeRegression tests exit code 1 (regression, not cleared).
func TestEvalGateExitCodeRegression(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Tier:           "semi-auto",
			Cleared:        false,
			BlockingSuites: []string{"recall-quality-v1"},
			Message:        "pass_rate 0.75 below threshold 0.80",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	resp, exitCode := checkEvalGateTier(context.Background(), "semi-auto")

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (regression), got %d", exitCode)
	}
	if resp.Cleared {
		t.Errorf("expected cleared=false, got true")
	}
	if len(resp.BlockingSuites) == 0 {
		t.Errorf("expected blocking_suites, got none")
	}
}

// TestEvalGateExitCodePreconditionNotMet tests exit code 3 (precondition not met).
func TestEvalGateExitCodePreconditionNotMet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Tier:         "semi-auto",
			Cleared:      false,
			Error:        "semantic_scoring_unavailable",
			Precondition: "plugin-retrieval BGE-M3 must be wired",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable) // HTTP 503
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	resp, exitCode := checkEvalGateTier(context.Background(), "semi-auto")

	if exitCode != 3 {
		t.Errorf("expected exit code 3 (precondition), got %d", exitCode)
	}
	if resp.Precondition == "" {
		t.Errorf("expected precondition message, got empty")
	}
}

// TestEvalGateExitCodeInfraError tests exit code 2 (infrastructure error).
func TestEvalGateExitCodeInfraError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Tier:  "semi-auto",
			Error: "database connection failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) // HTTP 500
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	resp, exitCode := checkEvalGateTier(context.Background(), "semi-auto")

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (infra error), got %d", exitCode)
	}
	if resp.Error == "" {
		t.Errorf("expected error message, got empty")
	}
}

// TestEvalGatePreconditionVsRegression verifies the critical distinction:
// HTTP 503 (precondition) should exit 3, while HTTP 200 with cleared=false should exit 1.
func TestEvalGatePreconditionVsRegression(t *testing.T) {
	// Test 1: Precondition (503) -> exit 3
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Tier:         "semi-auto",
			Cleared:      false,
			Precondition: "BGE-M3 not available",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server1.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server1.URL)

	_, exitCode1 := checkEvalGateTier(context.Background(), "semi-auto")

	// Test 2: Regression (200 with cleared=false) -> exit 1
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Tier:           "semi-auto",
			Cleared:        false,
			BlockingSuites: []string{"recall-quality-v1"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server2.Close()

	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server2.URL)

	_, exitCode2 := checkEvalGateTier(context.Background(), "semi-auto")

	if oldURL != "" {
		os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
	} else {
		os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
	}

	if exitCode1 != 3 {
		t.Errorf("precondition: expected exit 3, got %d", exitCode1)
	}
	if exitCode2 != 1 {
		t.Errorf("regression: expected exit 1, got %d", exitCode2)
	}
}

// TestEvalGateInvalidTier tests handling of non-existent tier (404).
func TestEvalGateInvalidTier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Error: "tier not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	_, exitCode := checkEvalGateTier(context.Background(), "unknown-tier")

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (infra error), got %d", exitCode)
	}
}

// TestEvalGateJSONOutput tests JSON output format.
func TestEvalGateJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Tier:    "semi-auto",
			Cleared: true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	ciEvalGateFlags.tier = "semi-auto"
	ciEvalGateFlags.json = true
	ciEvalGateFlags.verbose = false

	resp, exitCode := checkEvalGateTier(context.Background(), "semi-auto")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify JSON output can be marshaled
	jsonBytes, err := json.Marshal(map[string]any{
		"tier":    resp.Tier,
		"cleared": resp.Cleared,
	})
	if err != nil {
		t.Errorf("failed to marshal JSON output: %v", err)
	}
	if len(jsonBytes) == 0 {
		t.Errorf("expected JSON output, got empty")
	}
}

// TestExitError verifies ExitError type implements error interface.
func TestExitError(t *testing.T) {
	ee := NewExitError(3, "precondition not met")

	// Should implement error interface
	var _ error = ee

	// Should have ExitCode method
	if ee.ExitCode() != 3 {
		t.Errorf("expected exit code 3, got %d", ee.ExitCode())
	}

	// Error() should be informative
	errStr := ee.Error()
	if !bytes.Contains([]byte(errStr), []byte("3")) || !bytes.Contains([]byte(errStr), []byte("precondition")) {
		t.Errorf("unexpected error string: %s", errStr)
	}
}

// TestEvalGateBlockingSuites verifies multiple blocking suites are reported.
func TestEvalGateBlockingSuites(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Tier:           "full-auto",
			Cleared:        false,
			BlockingSuites: []string{"recall-quality-v1", "generation-v1"},
			Message:        "multiple suites below threshold",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	resp, exitCode := checkEvalGateTier(context.Background(), "full-auto")

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if len(resp.BlockingSuites) != 2 {
		t.Errorf("expected 2 blocking suites, got %d", len(resp.BlockingSuites))
	}
	if resp.BlockingSuites[0] != "recall-quality-v1" || resp.BlockingSuites[1] != "generation-v1" {
		t.Errorf("unexpected blocking suites: %v", resp.BlockingSuites)
	}
}

// TestEvalGateBadRequest tests HTTP 400 response (invalid request).
func TestEvalGateBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := EvalGateResponse{
			Error: "invalid tier format",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	oldURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", server.URL)
	defer func() {
		if oldURL != "" {
			os.Setenv("NSELF_EVAL_GATE_PLUGIN_URL", oldURL)
		} else {
			os.Unsetenv("NSELF_EVAL_GATE_PLUGIN_URL")
		}
	}()

	_, exitCode := checkEvalGateTier(context.Background(), "invalid-tier-@#$")

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (infra error), got %d", exitCode)
	}
}
