package deploy

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRunBlueGreen_dryRun verifies the blue-green adapter in dry-run mode:
// no external commands are run, all steps are marked pending, and
// .nself/state/deploy.json is written with success=true.
func TestRunBlueGreen_dryRun(t *testing.T) {
	tmp := t.TempDir()

	cfg := BlueGreenConfig{
		ProjectRoot: tmp,
		Target:      "staging",
		DryRun:      true,
	}

	result := RunBlueGreen(nil, cfg) //nolint:staticcheck // nil ctx safe for dry-run

	if !result.Success {
		t.Fatalf("expected dry-run success, got error: %s", result.Error)
	}
	if result.RolledBack {
		t.Errorf("expected RolledBack=false in dry-run")
	}
	if len(result.Steps) == 0 {
		t.Errorf("expected steps in dry-run result")
	}
	// All steps should indicate dry-run.
	for _, step := range result.Steps {
		if !strings.Contains(step.Status, "dry-run") {
			t.Errorf("expected dry-run status for step %q, got %q", step.Name, step.Status)
		}
	}
}

// TestRunBlueGreen_dryRunWritesState verifies that state is persisted to
// .nself/state/deploy.json even in dry-run mode.
func TestRunBlueGreen_dryRunWritesState(t *testing.T) {
	tmp := t.TempDir()

	cfg := BlueGreenConfig{
		ProjectRoot: tmp,
		Target:      "staging",
		DryRun:      true,
	}

	result := RunBlueGreen(nil, cfg) //nolint:staticcheck
	if !result.Success {
		t.Fatalf("unexpected failure: %s", result.Error)
	}

	// State file must exist at the canonical path.
	statePath := filepath.Join(tmp, DeployStateFile)
	state, err := LoadDeployState(tmp)
	if err != nil {
		t.Fatalf("LoadDeployState failed: %v", err)
	}

	_ = statePath
	if state.Strategy != StrategyBlueGreen {
		t.Errorf("Strategy: got %q, want %q", state.Strategy, StrategyBlueGreen)
	}
	if state.Target != "staging" {
		t.Errorf("Target: got %q, want %q", state.Target, "staging")
	}
	if !state.Success {
		t.Errorf("expected Success=true in state file")
	}
}

// TestRunBlueGreen_skipCanarySet verifies that the blue-green adapter always
// sets SkipCanary=true in the underlying bluegreen config (blue-green = full flip).
// We verify this indirectly: in dry-run, the step list must NOT contain a canary step.
func TestRunBlueGreen_noCanarySteps(t *testing.T) {
	tmp := t.TempDir()

	cfg := BlueGreenConfig{
		ProjectRoot: tmp,
		Target:      "local",
		DryRun:      true,
	}

	result := RunBlueGreen(nil, cfg) //nolint:staticcheck
	if !result.Success {
		t.Fatalf("unexpected failure: %s", result.Error)
	}

	for _, step := range result.Steps {
		if strings.Contains(strings.ToLower(step.Name), "canary soak") {
			t.Errorf("blue-green strategy must not have canary soak steps, got: %q", step.Name)
		}
	}
}

// TestBlueGreenConfig_stateReferencesBlueGreenStateFile verifies that the
// state written by the adapter records the bluegreen sub-package state file path.
func TestBlueGreenConfig_stateReferencesBlueGreenStateFile(t *testing.T) {
	tmp := t.TempDir()

	cfg := BlueGreenConfig{
		ProjectRoot: tmp,
		Target:      "prod",
		DryRun:      true,
	}

	RunBlueGreen(nil, cfg) //nolint:staticcheck

	state, err := LoadDeployState(tmp)
	if err != nil {
		t.Fatalf("LoadDeployState: %v", err)
	}

	if state.BlueGreenStateFile == "" {
		t.Errorf("expected BlueGreenStateFile to be set in persisted state")
	}
}

// TestRunBlueGreen_rollbackPath verifies that after a blue-green deploy the
// unified state file records StrategyBlueGreen, so that runDeployRollback will
// select the bluegreen.Rollback path (not promote.Rollback).
//
// This is the unit-level counterpart to the CR-C Fix 2 acceptance criterion:
// "deploy with --strategy=blue-green then rollback → verifies bluegreen.Rollback
// was invoked, NOT promote.Rollback".
//
// We cannot call runDeployRollback directly from this package (it lives in
// cmd/commands), but we can assert the state that determines the dispatch.
func TestRunBlueGreen_rollbackPath(t *testing.T) {
	tmp := t.TempDir()

	cfg := BlueGreenConfig{
		ProjectRoot: tmp,
		Target:      "staging",
		DryRun:      true,
	}

	result := RunBlueGreen(nil, cfg) //nolint:staticcheck
	if !result.Success {
		t.Fatalf("expected dry-run success, got: %s", result.Error)
	}

	// Load the unified state written by the adapter.
	state, err := LoadDeployState(tmp)
	if err != nil {
		t.Fatalf("LoadDeployState: %v", err)
	}

	// runDeployRollback switches on state.Strategy. Assert it is StrategyBlueGreen
	// so that the bluegreen.Rollback branch will be taken (not promote.Rollback).
	if state.Strategy != StrategyBlueGreen {
		t.Errorf("state.Strategy: got %q, want %q — rollback will use wrong path",
			state.Strategy, StrategyBlueGreen)
	}

	// Confirm the state file carries the bluegreen sub-state reference, which is
	// the trigger that distinguishes this from a rolling deploy in rollback dispatch.
	if state.BlueGreenStateFile == "" {
		t.Errorf("expected BlueGreenStateFile non-empty — rollback needs it to locate green state")
	}
}
