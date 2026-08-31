package bluegreen

// bluegreen_state.go — deploy state persistence and dry-run support.
//
// Purpose: swap and persist DeployState, record rollback history, and support dry-run deploys, used throughout bluegreen.go, split out for file size.
// Inputs: a DeployConfig and the in-memory DeployState to persist or load.
// Outputs: DeployState read from or written to the state file at StateFile.
// Constraints: pure move from bluegreen.go (CLI-R12 Batch E); no behaviour change.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// swapState marks green as the new active (blue) in the state file.
// After a successful deploy, green becomes the next blue for future deploys.
func swapState(projectRoot string) error {
	state, _ := loadState(projectRoot)
	state.Active = EnvBlue // after swap, blue is always the live env label
	state.BlueVersion = state.GreenVersion
	state.GreenVersion = ""
	state.CanaryPercent = 0
	state.LastDeploy = time.Now().UTC()
	return saveState(projectRoot, state)
}

func recordRollback(projectRoot string) error {
	state, _ := loadState(projectRoot)
	now := time.Now().UTC()
	state.LastRollback = &now
	state.CanaryPercent = 0
	return saveState(projectRoot, state)
}

func loadState(projectRoot string) (DeployState, error) {
	path := filepath.Join(projectRoot, StateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return DeployState{Active: EnvBlue}, nil
	}
	var s DeployState
	if err := json.Unmarshal(data, &s); err != nil {
		return DeployState{Active: EnvBlue}, nil
	}
	return s, nil
}

func saveState(projectRoot string, state DeployState) error {
	path := filepath.Join(projectRoot, StateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// dryRunDeploy returns a synthetic result listing all steps as pending.
func dryRunDeploy(cfg DeployConfig, start time.Time) DeployResult {
	steps := []DeployStep{
		{Name: "Migration compatibility check", Status: "pending (dry-run)"},
		{Name: "Pull new images", Status: "pending (dry-run)"},
		{Name: "Start green containers", Status: "pending (dry-run)"},
		{Name: "Health check green", Status: "pending (dry-run)"},
	}
	if !cfg.SkipCanary {
		steps = append(steps,
			DeployStep{Name: fmt.Sprintf("Canary traffic split (%d%%)", cfg.CanaryPercent), Status: "pending (dry-run)"},
			DeployStep{Name: fmt.Sprintf("Canary soak (%d min)", cfg.SoakMinutes), Status: "pending (dry-run)"},
		)
	}
	steps = append(steps,
		DeployStep{Name: "Promote green to 100%", Status: "pending (dry-run)"},
		DeployStep{Name: "Health check green (100%)", Status: "pending (dry-run)"},
		DeployStep{Name: "Stop blue containers", Status: "pending (dry-run)"},
		DeployStep{Name: "Persist blue/green state", Status: "pending (dry-run)"},
	)
	return DeployResult{
		Success:       true,
		Steps:         steps,
		Duration:      time.Since(start),
		CanaryPercent: cfg.CanaryPercent,
	}
}
