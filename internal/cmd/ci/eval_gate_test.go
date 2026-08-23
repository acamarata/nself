package ci

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nself-org/cli/internal/errs"
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

// assertGateExitCode runs the gate command and asserts the exit code it
// requests. Before CLI-R04 these three cases had to re-exec the test binary as
// a subprocess, because runEvalGate called os.Exit directly and would have
// killed the test process. The command now returns errs.Exit(code), so the code
// is an ordinary value and the subprocess dance is gone.
func assertGateExitCode(t *testing.T, args []string, want int) {
	t.Helper()

	cmd := newTestGateCmd()
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected a non-zero exit, got nil error")
	}

	var coder errs.ExitCoder
	if !errors.As(err, &coder) {
		t.Fatalf("error carries no exit code (%T): %v", err, err)
	}
	if coder.ExitCode() != want {
		t.Errorf("exit code = %d; want %d", coder.ExitCode(), want)
	}
}

// TestEvalGateExitCode_Blocked verifies exit 1 when the gate is NOT cleared
// (a regression).
func TestEvalGateExitCode_Blocked(t *testing.T) {
	srv := evalGateServer(t, http.StatusOK, map[string]interface{}{
		"tier":            "semi-auto",
		"cleared":         false,
		"enforced":        true,
		"blocking_suites": []string{"recall-quality-v1"},
	})
	assertGateExitCode(t, []string{"--tier", "semi-auto", "--eval-url", srv.URL}, exitBelowTreshold)
}

// TestEvalGateExitCode_Precondition verifies exit 3 when eval-gate returns 503.
// A 503 means plugin-retrieval is down — this is NOT a regression (exit 3, not 1).
func TestEvalGateExitCode_Precondition(t *testing.T) {
	srv := evalGateServer(t, http.StatusServiceUnavailable, map[string]interface{}{
		"error": "service unavailable",
	})
	assertGateExitCode(t, []string{"--tier", "semi-auto", "--eval-url", srv.URL}, exitPrecondition)
}

// TestEvalGateExitCode_InvalidTier verifies exit 2 for an unrecognised tier name.
func TestEvalGateExitCode_InvalidTier(t *testing.T) {
	assertGateExitCode(t, []string{"--tier", "unknown-tier", "--eval-url", "http://unused:9999"}, exitValidation)
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
