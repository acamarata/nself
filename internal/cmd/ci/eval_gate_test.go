package ci

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

// evalGateServer creates a test HTTP server that returns the given response for
// GET /eval/gate/{tier} requests. statusCode controls the HTTP response code.
func evalGateServer(t *testing.T, statusCode int, body interface{}) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestEvalGateExitCode_Cleared verifies exit 0 when the gate is cleared.
func TestEvalGateExitCode_Cleared(t *testing.T) {
	srv := evalGateServer(t, http.StatusOK, map[string]interface{}{
		"tier":     "semi-auto",
		"cleared":  true,
		"enforced": true,
	})

	cmd := newTestGateCmd()
	cmd.SetArgs([]string{"--tier", "semi-auto", "--eval-url", srv.URL})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	// Exit 0 means Execute returns nil.
}

// TestEvalGateExitCode_Blocked verifies exit 1 when gate is NOT cleared (regression).
// We test this at the unit level via a subprocess to capture os.Exit.
func TestEvalGateExitCode_Blocked(t *testing.T) {
	if os.Getenv("TEST_EXIT_BLOCKED") == "1" {
		srv := evalGateServer(t, http.StatusOK, map[string]interface{}{
			"tier":            "semi-auto",
			"cleared":         false,
			"enforced":        true,
			"blocking_suites": []string{"recall-quality-v1"},
		})
		cmd := newTestGateCmd()
		cmd.SetArgs([]string{"--tier", "semi-auto", "--eval-url", srv.URL})
		cmd.Execute() //nolint:errcheck
		return
	}

	// Rerun this test function as a subprocess to capture the os.Exit code.
	subcmd := exec.Command(os.Args[0], "-test.run=TestEvalGateExitCode_Blocked")
	subcmd.Env = append(os.Environ(), "TEST_EXIT_BLOCKED=1")
	err := subcmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit but got exit 0")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != exitBelowTreshold {
		t.Errorf("exit code = %d; want %d (exitBelowTreshold/regression)", exitErr.ExitCode(), exitBelowTreshold)
	}
}

// TestEvalGateExitCode_Precondition verifies exit 3 when eval-gate returns 503.
// A 503 means plugin-retrieval is down — this is NOT a regression (exit 3, not 1).
func TestEvalGateExitCode_Precondition(t *testing.T) {
	if os.Getenv("TEST_EXIT_PRECONDITION") == "1" {
		srv := evalGateServer(t, http.StatusServiceUnavailable, map[string]interface{}{
			"error": "service unavailable",
		})
		cmd := newTestGateCmd()
		cmd.SetArgs([]string{"--tier", "semi-auto", "--eval-url", srv.URL})
		cmd.Execute() //nolint:errcheck
		return
	}

	subcmd := exec.Command(os.Args[0], "-test.run=TestEvalGateExitCode_Precondition")
	subcmd.Env = append(os.Environ(), "TEST_EXIT_PRECONDITION=1")
	err := subcmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit but got exit 0")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != exitPrecondition {
		t.Errorf("exit code = %d; want %d (exitPrecondition, not regression)", exitErr.ExitCode(), exitPrecondition)
	}
}

// TestEvalGateExitCode_InvalidTier verifies exit 2 for an unrecognised tier name.
func TestEvalGateExitCode_InvalidTier(t *testing.T) {
	if os.Getenv("TEST_EXIT_INVALID_TIER") == "1" {
		cmd := newTestGateCmd()
		cmd.SetArgs([]string{"--tier", "unknown-tier", "--eval-url", "http://unused:9999"})
		cmd.Execute() //nolint:errcheck
		return
	}

	subcmd := exec.Command(os.Args[0], "-test.run=TestEvalGateExitCode_InvalidTier")
	subcmd.Env = append(os.Environ(), "TEST_EXIT_INVALID_TIER=1")
	err := subcmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit but got exit 0")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != exitValidation {
		t.Errorf("exit code = %d; want %d (exitValidation)", exitErr.ExitCode(), exitValidation)
	}
}

// TestEvalGateExitCode_TierFlagCoverage verifies --tier flag is tested for all valid tiers.
func TestEvalGateExitCode_TierFlagCoverage(t *testing.T) {
	for _, tier := range validTiers {
		tier := tier
		t.Run("tier="+tier, func(t *testing.T) {
			srv := evalGateServer(t, http.StatusOK, map[string]interface{}{
				"tier":     tier,
				"cleared":  true,
				"enforced": tier != "supervised",
			})
			cmd := newTestGateCmd()
			cmd.SetArgs([]string{"--tier", tier, "--eval-url", srv.URL})
			err := cmd.Execute()
			if err != nil {
				t.Errorf("tier %q: Execute() error: %v", tier, err)
			}
		})
	}
}

// newTestGateCmd returns a fresh cobra.Command for the gate subcommand.
// Each call returns an independent instance safe for parallel tests.
func newTestGateCmd() *gateTestHelper {
	return &gateTestHelper{cmd: NewEvalGateCmd()}
}

// gateTestHelper is a thin wrapper to keep test call-sites tidy.
type gateTestHelper struct {
	cmd interface {
		SetArgs([]string)
		Execute() error
	}
}

func (h *gateTestHelper) SetArgs(args []string) { h.cmd.SetArgs(args) }
func (h *gateTestHelper) Execute() error        { return h.cmd.Execute() }
