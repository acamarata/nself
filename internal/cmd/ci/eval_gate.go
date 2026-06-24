package ci

// eval_gate.go — `nself ci eval gate` subcommand implementation.
//
// Purpose: Check whether an autonomy tier is cleared by calling GET /eval/gate/{tier}
// on the nself-eval-gate plugin. Exits 0 when cleared, 1 when blocked (prints
// blocking suites and the gap), 2 on infrastructure error, 3 on precondition failure.
// Inputs: --tier flag (required); --endpoint override; --output for format selection.
// Outputs: tier clearance status to stdout; blocking suites and gap on failure.
// Constraints: never exits 0 if any required suite is below threshold (AND semantics);
// spec §6 exit codes are a HARD RULE — never change without cross-team review.
// SPORT: F02-COMMAND-INVENTORY.md — nself ci eval gate entry.

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/eval"
	"github.com/nself-org/cli/internal/ports"
	"github.com/spf13/cobra"
)

// EvalGateCmd returns the `nself ci eval gate` cobra subcommand.
//
// Purpose: Subcommand factory attached to EvalCmd(). Checks tier clearance from
// the nself-eval-gate plugin and maps the result to the correct exit code.
// Inputs: none — flag definitions attached inline.
// Outputs: *cobra.Command ready to be added to EvalCmd.
func EvalGateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Check autonomy-tier eval gate clearance",
		Long: `Check whether all eval suites required for a given autonomy tier have
cleared their configured thresholds.

Exit codes (HARD RULE — never change without cross-team review):
  0 — tier cleared (all required suites passed)
  1 — regression (one or more suites below threshold)
  2 — infrastructure error (plugin unavailable, network issue, etc.)
  3 — precondition not met (BGE-M3 not wired) — NOT a regression`,
		RunE: runEvalGate,
	}

	cmd.Flags().String("tier", "", "Autonomy tier to check: supervised|semi-auto|full-auto (required)")
	cmd.Flags().String("endpoint", "", "Override plugin base URL (default: http://localhost:3770)")
	_ = cmd.MarkFlagRequired("tier")

	return cmd
}

// runEvalGate is the RunE handler for `nself ci eval gate`.
//
// Purpose: Call GetGateStatus, print clearance status or blocking suite detail,
// and return the correct exitError for cobra to propagate as the process exit code.
// Inputs: --tier (required); --endpoint (optional override).
// Outputs:
//   - Cleared: prints "[GATE CLEARED] tier=<tier>" to stdout; exits 0.
//   - Blocked: prints "[GATE BLOCKED] tier=<tier>" + blocking suites to stderr; exits 1.
//   - Precondition: exits 3 (classifyClientError).
//   - Infra error: exits 2 (classifyClientError).
func runEvalGate(cmd *cobra.Command, _ []string) error {
	tier, _ := cmd.Flags().GetString("tier")
	endpoint, _ := cmd.Flags().GetString("endpoint")

	baseURL := endpoint
	if baseURL == "" {
		baseURL = ports.EvalGateBaseURL()
	}

	client := eval.NewClient(baseURL)
	ctx := cmd.Context()

	status, err := client.GetGateStatus(ctx, tier)
	if err != nil {
		return classifyClientError(err)
	}

	if status.Cleared {
		fmt.Fprintf(os.Stdout, "[GATE CLEARED] tier=%s\n", status.Tier)
		return nil
	}

	// Blocked — print detail then exit 1.
	fmt.Fprintf(os.Stderr, "[GATE BLOCKED] tier=%s\n", status.Tier)
	if len(status.BlockingSuites) > 0 {
		fmt.Fprintf(os.Stderr, "  blocking suites: %s\n", strings.Join(status.BlockingSuites, ", "))
	}
	if status.MinPassRate > 0 || status.MinSuiteScore > 0 {
		fmt.Fprintf(os.Stderr, "  required: pass_rate>=%.2f suite_score>=%.2f\n",
			status.MinPassRate, status.MinSuiteScore)
	}

	return &exitError{
		code: ExitCodeBelowThreshold,
		msg:  fmt.Sprintf("gate blocked for tier %q: %s", tier, strings.Join(status.BlockingSuites, ", ")),
	}
}
