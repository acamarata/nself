package commands

// Purpose: Small helpers shared by every release-cascade command (release,
// release status, release rollback): runStep executes and times one named
// cascade step, appending its outcome to the shared releaseResult; emitResult
// prints (or JSON-encodes) the final cascade outcome. Split out of
// release.go (CLI-R12) purely to keep release.go under the file-size cap —
// these two functions have no logic of their own beyond formatting/bookkeeping
// shared across the release/status/rollback cobra commands.
// Inputs: a context.Context, the shared *releaseResult being built up, and
// (for runStep) the step's name/number and the closure that performs it.
// Outputs: the accumulated releaseStepResult entries and the final
// human-readable or JSON summary printed to stdout.
// Constraints: pure move — no behavior changes.

import (
	"context"
	"fmt"
	"time"

	"github.com/nself-org/cli/internal/ui"
)

// runStep executes a named release step, logs it, and appends the result.
func runStep(ctx context.Context, result *releaseResult, stepNum int, name string, dryRun bool, fn func() error) error {
	start := time.Now()
	s := releaseStepResult{Step: stepNum, Name: name}

	if dryRun {
		s.Status = "dry-run"
		s.Message = "skipped (dry-run)"
		fmt.Printf("  [dry-run] Step %2d: %s\n", stepNum, name)
		result.Steps = append(result.Steps, s)
		return nil
	}

	fmt.Printf("  [running] Step %2d: %s\n", stepNum, name)
	err := fn()
	s.Elapsed = time.Since(start).Round(time.Millisecond).String()
	if err != nil {
		s.Status = "failed"
		s.Message = err.Error()
		result.Status = "failed"
		result.Error = fmt.Sprintf("step %d (%s) failed: %s", stepNum, name, err)
		result.Steps = append(result.Steps, s)
		fmt.Printf("  %s   Step %2d failed: %s\n", ui.C(ui.Red, "✗"), stepNum, err)
		return fmt.Errorf("release aborted at step %d (%s): %w", stepNum, name, err)
	}
	s.Status = "done"
	result.Steps = append(result.Steps, s)
	fmt.Printf("  %s   Step %2d done (%s)\n", ui.C(ui.Green, "✓"), stepNum, s.Elapsed)
	return nil
}

func emitResult(result releaseResult, jsonOut bool, err error) error {
	if jsonOut {
		_ = ui.PrintJSON(result)
	} else if err != nil {
		fmt.Printf("\n%s Release cascade FAILED: %s\n", ui.C(ui.Red, "✗"), err)
		fmt.Printf("\nRun 'nself release-rollback %s <prior>' to roll back if needed.\n", result.Tag)
	} else if result.DryRun {
		fmt.Printf("\n%s Dry-run complete — all %d steps logged.\n", ui.C(ui.Yellow, "[DRY-RUN]"), len(result.Steps))
	} else {
		fmt.Printf("\n%s Release %s complete in %s\n", ui.C(ui.Green, "✓"), ui.C(ui.Bold, result.Tag), result.Elapsed)
		fmt.Println("\nNext: monitor the 48h soak at status.nself.org")
	}
	return err
}
