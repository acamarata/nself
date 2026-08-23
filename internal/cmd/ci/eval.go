package ci

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/eval"
	"github.com/spf13/cobra"
)

// Purpose: nself ci eval — run a named eval suite against the nself-eval-gate plugin.
//   Supports validate-only mode (schema check without running), polling for completion,
//   artifact writing, and multiple output formats.
// Inputs:  --suite (suite slug), --validate-only, --output, --k, --dry-run, --repo, --tier flags.
// Outputs: eval-results.json in {repo}/.nself-ci/artifacts/; formatted result to stdout.
// Constraints: Exit codes: 0=passed, 1=below threshold, 2=validation error,
//   3=precondition not met (BGE-M3/gateway unavailable).
//   --all and --fail-fast are accepted flags (reserved for future multi-suite support).
// SPORT: CLI-CMD-EVAL-001

// evalExitCodes documents the exit code contract.
const (
	exitPassed        = 0
	exitBelowTreshold = 1
	exitValidation    = 2
	exitPrecondition  = 3
)

// NewEvalCmd creates the `nself ci eval` cobra sub-command.
func NewEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Run eval-gate suite or validate an eval-set YAML",
		Long: `Run a named eval suite via the nself-eval-gate plugin, or validate an
eval-set YAML file without running it.

Exit codes:
  0 — suite passed (pass_rate and suite_score >= configured threshold)
  1 — suite ran but is below threshold
  2 — validation error (bad YAML or invalid eval-set schema)
  3 — precondition not met (BGE-M3 or AI gateway unavailable)

Examples:
  nself ci eval --suite recall-quality-v1
  nself ci eval --suite recall-quality-v1 --output table
  nself ci eval --validate-only --suite recall-quality-v1
  nself ci eval --dry-run --suite recall-quality-v1`,
		SilenceUsage: true,
		RunE:         runEval,
	}

	cmd.Flags().String("suite", "", "Suite slug to run (required unless --all)")
	cmd.Flags().String("output", eval.FormatSummary, "Output format: json|table|summary")
	cmd.Flags().String("repo", ".", "Repo root path for artifact output")
	cmd.Flags().String("tier", "", "Autonomy tier for threshold lookup (supervised|semi-auto|full-auto)")
	cmd.Flags().String("eval-url", "http://localhost:3770", "nself-eval-gate plugin base URL")
	cmd.Flags().String("source-account", "primary", "Source account ID header value")
	cmd.Flags().Int("k", 3, "Top-k for recall-quality suites")
	cmd.Flags().Bool("validate-only", false, "Validate YAML schema without running the suite")
	cmd.Flags().Bool("dry-run", false, "Print what would run without executing")
	cmd.Flags().Bool("all", false, "Run all registered suites (reserved; not yet implemented)")
	cmd.Flags().Bool("fail-fast", false, "Stop after first failing task (reserved; not yet implemented)")
	cmd.Flags().Bool("no-cache", false, "Disable embed/rubric cache for this run (reserved; not yet implemented)")

	return cmd
}

func runEval(cmd *cobra.Command, _ []string) error {
	suiteSlug, _ := cmd.Flags().GetString("suite")
	outputFmt, _ := cmd.Flags().GetString("output")
	repoRoot, _ := cmd.Flags().GetString("repo")
	evalURL, _ := cmd.Flags().GetString("eval-url")
	sourceAccount, _ := cmd.Flags().GetString("source-account")
	validateOnly, _ := cmd.Flags().GetBool("validate-only")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Exit codes here are a documented contract for CI wrappers, so they are
	// preserved exactly; only the os.Exit call moves to main() (CLI-R04).
	if suiteSlug == "" {
		cmd.PrintErrln("error: --suite is required")
		return errs.Exit(exitValidation)
	}

	repoAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}

	client := &eval.Client{
		BaseURL:         evalURL,
		SourceAccountID: sourceAccount,
	}

	ctx := context.Background()

	if validateOnly {
		return runValidateOnly(ctx, cmd, client, suiteSlug, repoAbs)
	}

	if dryRun {
		cmd.Printf("dry-run: would run suite=%s via %s\n", suiteSlug, evalURL)
		cmd.Printf("dry-run: artifact output → %s/.nself-ci/artifacts/eval-results.json\n", repoAbs)
		return nil
	}

	return runSuite(ctx, cmd, client, suiteSlug, outputFmt, repoAbs)
}

// runValidateOnly validates eval-set YAML without executing the suite.
// Reads the eval-set file from {repoRoot}/evals/{suiteSlug}.yaml or a standard path.
func runValidateOnly(ctx context.Context, cmd *cobra.Command, client *eval.Client, suiteSlug, repoRoot string) error {
	// Try to read the YAML from the canonical eval-set path.
	candidates := []string{
		filepath.Join(repoRoot, "evals", suiteSlug+".yaml"),
		filepath.Join(repoRoot, ".nself-ci", "evals", suiteSlug+".yaml"),
		filepath.Join(repoRoot, suiteSlug+".yaml"),
	}

	var yamlContent []byte
	var readErr error
	for _, path := range candidates {
		yamlContent, readErr = os.ReadFile(path)
		if readErr == nil {
			break
		}
	}

	if readErr != nil {
		// Fall back to asking the plugin to validate an empty file — this will return validation errors.
		yamlContent = []byte{}
	}

	result, err := client.ValidateYAML(ctx, yamlContent)
	if err != nil {
		cmd.PrintErrf("error: %v\n", err)
		return errs.Exit(exitValidation)
	}

	if result.Valid {
		cmd.Printf("valid: eval-set schema OK for suite=%s\n", suiteSlug)
		return nil
	}

	eval.WriteValidationErrors(cmd.OutOrStdout(), result.Errors)
	return errs.Exit(exitValidation)
}

// runSuite triggers the eval run and polls for completion.
func runSuite(ctx context.Context, cmd *cobra.Command, client *eval.Client, suiteSlug, outputFmt, repoRoot string) error {
	cmd.Printf("running eval suite: %s\n", suiteSlug)

	queued, err := client.RunSuite(ctx, suiteSlug)
	if err != nil {
		cmd.PrintErrf("error: trigger eval: %v\n", err)
		return errs.Exit(exitBelowTreshold)
	}

	cmd.Printf("run queued: %s (polling...)\n", queued.RunID)

	result, err := client.WaitForRun(ctx, queued.RunID)
	if err != nil {
		if err == eval.ErrPreconditionNotMet {
			cmd.PrintErrf("error: precondition not met — BGE-M3 or AI gateway unavailable\n")
			return errs.Exit(exitPrecondition)
		}
		if err == eval.ErrPollTimeout {
			cmd.PrintErrf("error: run did not complete within 10 minutes\n")
			return errs.Exit(exitBelowTreshold)
		}
		cmd.PrintErrf("error: poll run: %v\n", err)
		return errs.Exit(exitBelowTreshold)
	}

	if writeErr := eval.WriteResult(cmd.OutOrStdout(), result, outputFmt, repoRoot); writeErr != nil {
		cmd.PrintErrf("warning: output error: %v\n", writeErr)
	}

	if !result.Passed {
		return errs.Exit(exitBelowTreshold)
	}
	return nil
}
