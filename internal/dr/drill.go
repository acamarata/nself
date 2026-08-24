// Package dr provides disaster recovery operations: drills, standby promotion,
// rollback, and split-brain fencing.
package dr

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// Scenario identifies the type of DR drill.
type Scenario string

const (
	// ScenarioColdStart is the only supported DR drill scenario in v1.0.9.
	// It provisions a fresh VM, installs nSelf, restores the latest verified
	// backup, runs smoke queries from the verify catalog, and records RTO.
	ScenarioColdStart Scenario = "cold-start"

	// ScenarioRegionFailover is not supported in v1.0.9 (single-region
	// deployment by design). Cross-region replication is planned for v1.1.0.
	// Using this scenario returns a clear deprecation message, not a stub.
	ScenarioRegionFailover Scenario = "region-failover"

	// ScenarioDataCorruption is not supported in v1.0.9. PITR recovery via
	// pgbackrest is planned for v1.1.0. Using this scenario returns a clear
	// deprecation message, not a stub.
	ScenarioDataCorruption Scenario = "data-corruption"
)

// DrillOptions holds flags for `nself dr drill`.
type DrillOptions struct {
	Scenario Scenario
	DryRun   bool
}

// DrillResult holds the outcome of a DR drill.
type DrillResult struct {
	ID            string                 `json:"id"`
	Scenario      Scenario               `json:"scenario"`
	StartedAt     time.Time              `json:"started_at"`
	FinishedAt    time.Time              `json:"finished_at"`
	Status        string                 `json:"status"` // success, failed
	RowCountDelta map[string]int64       `json:"row_count_delta"`
	Details       map[string]interface{} `json:"details"`
}

// Drill executes a disaster recovery drill by provisioning a fresh VM,
// restoring from backup, and verifying data integrity.
func Drill(ctx context.Context, cfg *config.Config, opts DrillOptions) (*DrillResult, error) {
	start := time.Now()
	result := &DrillResult{
		ID:            fmt.Sprintf("dr-%s-%s", opts.Scenario, start.Format("20060102-150405")),
		Scenario:      opts.Scenario,
		StartedAt:     start,
		Status:        "running",
		RowCountDelta: make(map[string]int64),
		Details:       make(map[string]interface{}),
	}

	if opts.DryRun {
		slog.Info("dry-run: would execute DR drill", "scenario", opts.Scenario)
		result.Status = "dry-run"
		result.FinishedAt = time.Now()
		return result, nil
	}

	slog.Info("starting DR drill", "scenario", opts.Scenario, "id", result.ID)

	switch opts.Scenario {
	case ScenarioColdStart:
		if err := drillColdStart(ctx, cfg, result); err != nil {
			result.Status = "failed"
			result.FinishedAt = time.Now()
			result.Details["error"] = err.Error()
			return result, fmt.Errorf("%w: %v", errs.ErrDRDrillFailed, err)
		}
	case ScenarioRegionFailover:
		// Not supported in v1.0.9. nSelf is single-region by design.
		// Cross-region replication is planned for v1.1.0.
		// See docs/operations/disaster-recovery-runbook.md for details.
		return nil, fmt.Errorf("scenario %q not supported in v1.0.9 (planned v1.1.0); see docs/operations/disaster-recovery-runbook.md", opts.Scenario)
	case ScenarioDataCorruption:
		// Not supported in v1.0.9. PITR recovery via pgbackrest is planned for v1.1.0.
		// See docs/operations/disaster-recovery-runbook.md for details.
		return nil, fmt.Errorf("scenario %q not supported in v1.0.9 (planned v1.1.0); see docs/operations/disaster-recovery-runbook.md", opts.Scenario)
	default:
		return nil, fmt.Errorf("unknown DR scenario: %s", opts.Scenario)
	}

	result.Status = "success"
	result.FinishedAt = time.Now()
	slog.Info("DR drill complete", "id", result.ID, "duration", result.FinishedAt.Sub(result.StartedAt))
	return result, nil
}

// FormatDrillResult renders a drill result as JSON or table.
func FormatDrillResult(result *DrillResult, format string) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
