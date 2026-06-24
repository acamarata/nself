// Package ci provides cobra subcommands for the `nself ci` command group.
//
// Purpose: Implement `nself ci eval` — run eval suites via the nself-eval-gate plugin
// and emit a structured eval-results.json artifact to {repo}/.nself-ci/artifacts/.
// This file handles the `eval` subcommand and its flag set.
// Inputs: --suite, --all, --repo, --tier, --output, --validate-only, --fail-fast,
// --k, --no-cache, --dry-run flags (full set per spec §6).
// Outputs: per-task progress, suite summary, eval-results.json artifact; exit codes
// 0=passed, 1=below threshold, 2=validation/plugin error, 3=precondition not met.
// Constraints: file must stay ≤300 lines; delegate heavy logic to internal/eval/.
// SPORT: F02-COMMAND-INVENTORY.md — nself ci eval entry.
package ci

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/eval"
	"github.com/nself-org/cli/internal/ports"
	"github.com/spf13/cobra"
)

// ExitCodePassed is returned when all suites pass their thresholds.
const ExitCodePassed = 0

// ExitCodeBelowThreshold is returned when one or more suites fail their threshold.
const ExitCodeBelowThreshold = 1

// ExitCodeValidationError is returned on YAML validation errors or plugin unavailability.
const ExitCodeValidationError = 2

// ExitCodePreconditionNotMet is returned when BGE-M3 or plugin-retrieval is not wired.
const ExitCodePreconditionNotMet = 3

// EvalCmd returns the `nself ci eval` cobra command.
//
// Purpose: Factory function called by cmd/commands/ci.go init() to attach eval
// under the ciCmd parent. Separating command construction from registration
// allows the ci package to remain independent of the commands package.
// Inputs: none — flags are attached to the returned command.
// Outputs: *cobra.Command with eval and eval-gate subcommands attached.
func EvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Run eval suites via nself-eval-gate",
		Long: `Run one or more eval suites against the nself-eval-gate plugin and emit
eval-results.json to {repo}/.nself-ci/artifacts/.

Exit codes:
  0 — all suites passed threshold
  1 — one or more suites below threshold
  2 — validation error or plugin unavailable
  3 — precondition not met (e.g. BGE-M3 not wired)`,
		RunE: runEval,
	}

	cmd.Flags().String("suite", "", "Eval suite slug (required unless --all)")
	cmd.Flags().Bool("all", false, "Run all registered suites for this repo")
	cmd.Flags().String("repo", "", "Target repo slug (default: detected from cwd)")
	cmd.Flags().String("tier", "", "Assert tier clearance after run (exits non-zero if not cleared)")
	cmd.Flags().String("output", "summary", "Output format: json|table|summary")
	cmd.Flags().Bool("validate-only", false, "Parse and validate YAML without running")
	cmd.Flags().Bool("fail-fast", false, "Stop at first failing task")
	cmd.Flags().Int("k", 3, "k for recall@k metrics")
	cmd.Flags().Bool("no-cache", false, "Disable embedding and judge cache")
	cmd.Flags().Bool("dry-run", false, "Print what would run without scoring")
	cmd.Flags().String("endpoint", "", "Override plugin base URL (default: http://localhost:3770)")

	// Attach the eval gate subcommand.
	cmd.AddCommand(EvalGateCmd())

	return cmd
}

// runEval is the RunE handler for `nself ci eval`.
//
// Purpose: Orchestrate flag parsing, plugin client construction, YAML validation
// (--validate-only), suite run + poll, artifact writing, and exit code selection.
// Inputs: cobra.Command with parsed flags; args (unused — suite via --suite flag).
// Outputs: os.Exit via cobra exit-code mechanism (return exitError).
// Constraints: all heavy logic delegated to internal/eval package; this function
// must stay ≤80 lines.
func runEval(cmd *cobra.Command, _ []string) error {
	suite, _ := cmd.Flags().GetString("suite")
	all, _ := cmd.Flags().GetBool("all")
	repo, _ := cmd.Flags().GetString("repo")
	outputFmt, _ := cmd.Flags().GetString("output")
	validateOnly, _ := cmd.Flags().GetBool("validate-only")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	k, _ := cmd.Flags().GetInt("k")
	endpoint, _ := cmd.Flags().GetString("endpoint")

	if !all && suite == "" && !validateOnly {
		return &exitError{code: ExitCodeValidationError, msg: "one of --suite or --all is required"}
	}

	baseURL := endpoint
	if baseURL == "" {
		baseURL = ports.EvalGateBaseURL()
	}
	client := eval.NewClient(baseURL)
	ctx := cmd.Context()

	// --validate-only: load YAML from cwd suite file and validate without running.
	if validateOnly {
		yamlPath := findSuiteYAML(suite)
		if yamlPath == "" {
			return &exitError{code: ExitCodeValidationError, msg: fmt.Sprintf("suite YAML not found for %q; expected at .claude/evals/%s.yaml", suite, suite)}
		}
		yamlContent, err := os.ReadFile(yamlPath)
		if err != nil {
			return &exitError{code: ExitCodeValidationError, msg: fmt.Sprintf("reading suite YAML %s: %v", yamlPath, err)}
		}
		resp, err := client.ValidateYAML(ctx, yamlContent)
		if err != nil {
			return classifyClientError(err)
		}
		if !resp.Valid {
			for _, e := range resp.Errors {
				fmt.Fprintf(os.Stderr, "  validation error: %s — %s\n", e.Field, e.Message)
			}
			return &exitError{code: ExitCodeValidationError, msg: "YAML validation failed"}
		}
		fmt.Fprintln(os.Stdout, "YAML validation passed")
		return nil
	}

	if dryRun {
		if all {
			fmt.Fprintf(os.Stdout, "[DRY-RUN] Would run all suites for repo=%s k=%d\n", repo, k)
		} else {
			fmt.Fprintf(os.Stdout, "[DRY-RUN] Would run suite=%s repo=%s k=%d\n", suite, repo, k)
		}
		return nil
	}

	// Run the suite.
	fmt.Fprintf(os.Stdout, "[EVAL] Running suite: %s...\n", suite)

	runResp, err := client.RunSuite(ctx, eval.RunRequest{
		SuiteSlug: suite,
		Repo:      repo,
		K:         k,
	})
	if err != nil {
		return classifyClientError(err)
	}

	// Poll until completion.
	result, err := client.GetRun(ctx, runResp.RunID)
	if err != nil {
		return classifyClientError(err)
	}

	// Write output.
	artifactsDir := filepath.Join(".nself-ci", "artifacts")
	if err := eval.WriteOutput(os.Stdout, result, eval.OutputFormat(outputFmt), artifactsDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: output error: %v\n", err)
	}

	if !result.Passed {
		return &exitError{code: ExitCodeBelowThreshold, msg: fmt.Sprintf("suite %q below threshold (score=%.2f)", result.Suite, result.SuiteScore)}
	}
	return nil
}

// findSuiteYAML searches for the eval YAML file for a given suite slug.
// Purpose: Locate {cwd}/.claude/evals/{slug}.yaml for --validate-only mode.
// Inputs: slug — suite slug string (may be empty for --all).
// Outputs: absolute path if found; empty string if not found.
func findSuiteYAML(slug string) string {
	if slug == "" {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	path := filepath.Join(cwd, ".claude", "evals", slug+".yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// exitError carries a CLI exit code and message for cobra's error handling.
// Purpose: Allow runEval to return a typed error that cmd/commands/ci.go can
// inspect to call os.Exit with the correct code.
// Constraints: cobra prints err.Error() to stderr; exit code is embedded.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// ExitCode returns the process exit code for this error.
func (e *exitError) ExitCode() int { return e.code }

// classifyClientError maps eval client sentinel errors to exitError with the
// correct exit code per spec §6.
//
// Purpose: Centralize the error-to-exit-code mapping so both eval.go and
// eval_gate.go use the same classification logic.
// Inputs: err from eval.Client methods.
// Outputs: *exitError with the appropriate code.
func classifyClientError(err error) *exitError {
	if errors.Is(err, eval.ErrPreconditionNotMet) {
		return &exitError{code: ExitCodePreconditionNotMet, msg: err.Error()}
	}
	if errors.Is(err, eval.ErrPluginUnavailable) {
		return &exitError{code: ExitCodeValidationError, msg: err.Error()}
	}
	return &exitError{code: ExitCodeValidationError, msg: err.Error()}
}
