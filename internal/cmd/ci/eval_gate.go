package ci

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/eval"
	"github.com/spf13/cobra"
)

// Purpose: nself ci eval gate — check the autonomy tier gate via nself-eval-gate.
//   Fetches GET /eval/gate/{tier} and exits 0=cleared, 1=regression/blocked,
//   2=invalid tier argument, 3=precondition not met (plugin-retrieval down).
//   Prints blocking suites and gap to stderr for triage.
// Inputs:  --tier (required), --eval-url, --source-account flags.
// Outputs: gate status to stdout; blocking suites on stderr.
// Constraints: Exit codes: 0=cleared, 1=blocked/regression, 2=validation error,
//   3=precondition not met (BGE-M3/gateway unavailable — NOT a regression).
//   supervised tier always exits 0 (enforced=false by design).
// SPORT: CLI-CMD-EVAL-GATE-001

// validTiers lists the allowed autonomy tier names.
var validTiers = []string{"supervised", "semi-auto", "full-auto"}

// NewEvalGateCmd creates the `nself ci eval gate` cobra sub-command.
func NewEvalGateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Check whether an autonomy tier gate is cleared",
		Long: `Check the eval gate for a given autonomy tier.

Fetches the gate status from the nself-eval-gate plugin and exits with:
  0 — tier is cleared (all required suites pass thresholds)
  1 — tier is blocked (one or more suites are below threshold or have no runs)
  2 — invalid tier name provided
  3 — precondition not met (BGE-M3 or AI gateway unavailable; NOT a regression)

The supervised tier always exits 0 (enforcement disabled by design).

Valid tiers: supervised, semi-auto, full-auto

Examples:
  nself ci eval gate --tier supervised
  nself ci eval gate --tier semi-auto
  nself ci eval gate --tier full-auto`,
		SilenceUsage: true,
		RunE:         runEvalGate,
	}

	cmd.Flags().String("tier", "", fmt.Sprintf("Autonomy tier to check (required): %s", strings.Join(validTiers, "|")))
	cmd.Flags().String("eval-url", "http://localhost:3770", "nself-eval-gate plugin base URL")
	cmd.Flags().String("source-account", "primary", "Source account ID header value")

	_ = cmd.MarkFlagRequired("tier")

	return cmd
}

func runEvalGate(cmd *cobra.Command, _ []string) error {
	tier, _ := cmd.Flags().GetString("tier")
	evalURL, _ := cmd.Flags().GetString("eval-url")
	sourceAccount, _ := cmd.Flags().GetString("source-account")

	if !isValidTier(tier) {
		cmd.PrintErrf("error: invalid tier %q — valid values: %s\n", tier, strings.Join(validTiers, ", "))
		os.Exit(exitValidation)
	}

	client := &eval.Client{
		BaseURL:         evalURL,
		SourceAccountID: sourceAccount,
	}

	ctx := context.Background()

	gs, err := client.GetGateStatus(ctx, tier)
	if err != nil {
		// HTTP 503 or ErrPreconditionNotMet signals an infrastructure precondition failure
		// (e.g. plugin-retrieval / BGE-M3 not wired). This is NOT a regression — exit code 3.
		if isPreconditionError(err) {
			cmd.PrintErrf("Precondition not met: %v\n", err)
			cmd.PrintErrf("hint: ensure plugin-retrieval and nself-ai-gateway are running.\n")
			os.Exit(exitPrecondition)
		}
		// Any other HTTP/infra error — treat as infra failure (exit 2, not regression).
		cmd.PrintErrf("error: get gate status: %v\n", err)
		os.Exit(exitValidation)
	}

	eval.WriteGateStatus(cmd.OutOrStdout(), gs, true)

	if !gs.Cleared {
		os.Exit(exitBelowTreshold)
	}
	return nil
}

// isPreconditionError returns true if err indicates a BGE-M3 / gateway precondition failure.
// These are reported as HTTP 503 from nself-eval-gate or as ErrPreconditionNotMet from the
// scorer layer. Precondition failures must exit code 3 — they are NOT quality regressions.
func isPreconditionError(err error) bool {
	if errors.Is(err, eval.ErrPreconditionNotMet) {
		return true
	}
	// HTTP 503 or 502 in the error message from the do() client.
	msg := err.Error()
	return strings.Contains(msg, "HTTP 503") ||
		strings.Contains(msg, "HTTP 502") ||
		strings.Contains(msg, "precondition not met") ||
		strings.Contains(msg, "service unavailable")
}

// isValidTier returns true if tier is one of the known autonomy tier names.
func isValidTier(tier string) bool {
	for _, v := range validTiers {
		if v == tier {
			return true
		}
	}
	return false
}
