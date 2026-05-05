package deploy

import (
	"strings"
	"testing"
)

// TestRunCanary_dryRun verifies the canary adapter in dry-run mode:
// no external commands are run, all steps are marked pending, and
// .nself/state/deploy.json is written with success=true.
func TestRunCanary_dryRun(t *testing.T) {
	tmp := t.TempDir()

	cfg := CanaryConfig{
		ProjectRoot:    tmp,
		Target:         "staging",
		InitialPercent: 10,
		DryRun:         true,
	}

	result := RunCanary(nil, cfg) //nolint:staticcheck // nil ctx safe for dry-run

	if !result.Success {
		t.Fatalf("expected dry-run success, got error: %s", result.Error)
	}
	if result.RolledBack {
		t.Errorf("expected RolledBack=false in dry-run")
	}
	if len(result.Steps) == 0 {
		t.Errorf("expected steps in dry-run result")
	}
	for _, step := range result.Steps {
		if !strings.Contains(step.Status, "dry-run") {
			t.Errorf("expected dry-run status for step %q, got %q", step.Name, step.Status)
		}
	}
}

// TestRunCanary_dryRunWritesState verifies that state is persisted to
// .nself/state/deploy.json even in dry-run mode.
func TestRunCanary_dryRunWritesState(t *testing.T) {
	tmp := t.TempDir()

	cfg := CanaryConfig{
		ProjectRoot:    tmp,
		Target:         "staging",
		InitialPercent: 10,
		DryRun:         true,
	}

	result := RunCanary(nil, cfg) //nolint:staticcheck
	if !result.Success {
		t.Fatalf("unexpected failure: %s", result.Error)
	}

	state, err := LoadDeployState(tmp)
	if err != nil {
		t.Fatalf("LoadDeployState failed: %v", err)
	}

	if state.Strategy != StrategyCanary {
		t.Errorf("Strategy: got %q, want %q", state.Strategy, StrategyCanary)
	}
	if state.Target != "staging" {
		t.Errorf("Target: got %q, want %q", state.Target, "staging")
	}
	if !state.Success {
		t.Errorf("expected Success=true in state file")
	}
}

// TestRunCanary_defaultInitialPercent verifies that when InitialPercent is 0,
// the adapter defaults to 10% as specified.
func TestRunCanary_defaultInitialPercent(t *testing.T) {
	tmp := t.TempDir()

	cfg := CanaryConfig{
		ProjectRoot:    tmp,
		Target:         "local",
		InitialPercent: 0, // should default to 10
		DryRun:         true,
	}

	result := RunCanary(nil, cfg) //nolint:staticcheck
	if !result.Success {
		t.Fatalf("unexpected failure: %s", result.Error)
	}

	// In dry-run the result reports the configured canary percent.
	// Step list should contain a canary traffic split step mentioning 10%.
	foundCanaryStep := false
	for _, step := range result.Steps {
		if strings.Contains(step.Name, "10%") || strings.Contains(step.Name, "Canary traffic") {
			foundCanaryStep = true
			break
		}
	}
	if !foundCanaryStep {
		t.Errorf("expected canary traffic step at 10%% default, steps: %v", result.Steps)
	}
}

// TestRunCanary_hasSoakStep verifies that canary deploys include a soak step
// (distinguishing them from blue-green deploys).
func TestRunCanary_hasSoakStep(t *testing.T) {
	tmp := t.TempDir()

	cfg := CanaryConfig{
		ProjectRoot:    tmp,
		Target:         "staging",
		InitialPercent: 20,
		DryRun:         true,
	}

	result := RunCanary(nil, cfg) //nolint:staticcheck

	foundSoak := false
	for _, step := range result.Steps {
		if strings.Contains(strings.ToLower(step.Name), "soak") {
			foundSoak = true
			break
		}
	}
	if !foundSoak {
		t.Errorf("expected canary soak step in result, steps: %v", result.Steps)
	}
}

// TestRunCanary_stateReferencesBlueGreenStateFile verifies that the state
// written by the canary adapter records the bluegreen sub-package state file path.
func TestRunCanary_stateReferencesBlueGreenStateFile(t *testing.T) {
	tmp := t.TempDir()

	cfg := CanaryConfig{
		ProjectRoot: tmp,
		Target:      "prod",
		DryRun:      true,
	}

	RunCanary(nil, cfg) //nolint:staticcheck

	state, err := LoadDeployState(tmp)
	if err != nil {
		t.Fatalf("LoadDeployState: %v", err)
	}

	if state.BlueGreenStateFile == "" {
		t.Errorf("expected BlueGreenStateFile to be set in persisted state")
	}
}
