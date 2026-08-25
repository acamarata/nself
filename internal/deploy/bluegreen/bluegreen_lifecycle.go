package bluegreen

// bluegreen_lifecycle.go — deploy lifecycle operations after the initial Deploy.
//
// Purpose: promote, roll back, and inspect an in-progress or completed blue/green deploy, plus the migration-compatibility check and Nginx upstream generator, split out of bluegreen.go for file size.
// Inputs: a DeployConfig identifying the target project and the persisted DeployState written by Deploy.
// Outputs: an updated DeployState, a RollbackResult, a MigrationCheckResult, or generated Nginx upstream config, depending on the function.
// Constraints: pure move from bluegreen.go (CLI-R12 Batch E); no behaviour change.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Promote flips Nginx to 100% green without a new deploy.
// Used by `nself deploy --promote` after a manual canary review.
func Promote(ctx context.Context, cfg DeployConfig) DeployResult {
	start := time.Now()
	applyDefaults(&cfg)

	if cfg.DryRun {
		return DeployResult{
			Success: true,
			Steps: []DeployStep{
				{Name: "Nginx upstream flip to 100% green", Status: "pending (dry-run)"},
				{Name: "Reload Nginx", Status: "pending (dry-run)"},
			},
			Duration:      time.Since(start),
			CanaryPercent: 100,
		}
	}

	if err := setNginxWeights(cfg, 100); err != nil {
		return DeployResult{
			Success:  false,
			Steps:    []DeployStep{{Name: "Nginx upstream flip to 100% green", Status: "failed"}},
			Duration: time.Since(start),
			Error:    err.Error(),
		}
	}

	healthTimeout := time.Duration(cfg.HealthTimeoutSec) * time.Second
	if err := waitForHealth(ctx, cfg, EnvGreen, healthTimeout); err != nil {
		return DeployResult{
			Success:  false,
			Steps:    []DeployStep{{Name: "Health check green (100%)", Status: "failed"}},
			Duration: time.Since(start),
			Error:    err.Error(),
		}
	}

	_ = swapState(cfg.ProjectRoot)

	return DeployResult{
		Success: true,
		Steps: []DeployStep{
			{Name: "Nginx upstream flip to 100% green", Status: "done"},
			{Name: "Health check green (100%)", Status: "done"},
			{Name: "Persist blue/green state", Status: "done"},
		},
		Duration:      time.Since(start),
		CanaryPercent: 100,
	}
}

// Rollback resets Nginx to 100% blue and stops green containers.
// Target rollback time: < 5 seconds.
func Rollback(ctx context.Context, cfg DeployConfig) RollbackResult {
	start := time.Now()
	applyDefaults(&cfg)

	if cfg.DryRun {
		return RollbackResult{
			Success:  true,
			Duration: time.Since(start),
		}
	}

	// Reset Nginx weights to 100% blue (0% green).
	if err := setNginxWeights(cfg, 0); err != nil {
		return RollbackResult{
			Success:  false,
			Duration: time.Since(start),
			Error:    fmt.Sprintf("nginx weight reset failed: %v", err),
		}
	}

	// Stop green containers.
	if err := stopGreen(ctx, cfg); err != nil {
		return RollbackResult{
			Success:  false,
			Duration: time.Since(start),
			Error:    fmt.Sprintf("stop green failed: %v", err),
		}
	}

	// Record rollback in state file.
	_ = recordRollback(cfg.ProjectRoot)

	return RollbackResult{
		Success:  true,
		Duration: time.Since(start),
	}
}

// Status returns the current blue/green state from the state file.
// Returns a zero-value DeployState when no state file exists (fresh install).
func Status(projectRoot string) (DeployState, error) {
	return loadState(projectRoot)
}

// CheckMigrationCompatibility checks whether pending migrations are safe to
// run during a canary deploy (i.e., backward-compatible with the blue version).
//
// The check is heuristic-based: it scans migration filenames for destructive
// SQL patterns (DROP COLUMN, DROP TABLE, RENAME COLUMN, ALTER COLUMN TYPE,
// NOT NULL without DEFAULT). A migration that matches these patterns is flagged
// as incompatible.
//
// For authoritative checks, users should run `nself migrate --check` which
// executes the full migration engine against a preview schema.
func CheckMigrationCompatibility(projectRoot string) MigrationCheckResult {
	migrationsDir := filepath.Join(projectRoot, "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		// No migrations directory: compatible by definition.
		return MigrationCheckResult{Compatible: true}
	}

	incompatible := []string{}
	incompatiblePatterns := []string{
		"drop_column",
		"drop_table",
		"rename_column",
		"alter_column_type",
		"not_null_no_default",
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		for _, pattern := range incompatiblePatterns {
			if strings.Contains(name, pattern) {
				incompatible = append(incompatible, entry.Name())
				break
			}
		}
	}

	if len(incompatible) == 0 {
		return MigrationCheckResult{Compatible: true}
	}

	return MigrationCheckResult{
		Compatible:        false,
		IncompatibleFiles: incompatible,
		Reason:            "migration filename indicates a destructive schema change incompatible with canary deploy",
	}
}

// GenerateNginxUpstream generates the nginx upstream block for blue/green traffic split.
// canaryPercent is the percentage of traffic to route to green (0-100).
// When canaryPercent is 0, all traffic goes to blue. When 100, all traffic goes to green.
func GenerateNginxUpstream(cfg DeployConfig, canaryPercent int) string {
	blueBase := 8080 + cfg.BluePortOffset
	greenBase := 8080 + cfg.GreenPortOffset

	blueWeight := 100 - canaryPercent
	greenWeight := canaryPercent

	var sb strings.Builder
	sb.WriteString("upstream nself_api {\n")

	if blueWeight > 0 {
		sb.WriteString(fmt.Sprintf("    server 127.0.0.1:%d weight=%d;   # blue\n", blueBase, blueWeight))
	}
	if greenWeight > 0 {
		sb.WriteString(fmt.Sprintf("    server 127.0.0.1:%d weight=%d;   # green (%d%% canary)\n", greenBase, greenWeight, canaryPercent))
	}

	// When one side has 0 weight, still include it as backup to avoid nginx config errors.
	if blueWeight == 0 {
		sb.WriteString(fmt.Sprintf("    server 127.0.0.1:%d weight=0 backup;   # blue (idle)\n", blueBase))
	}
	if greenWeight == 0 {
		sb.WriteString(fmt.Sprintf("    server 127.0.0.1:%d weight=0 backup;   # green (idle)\n", greenBase))
	}

	sb.WriteString("}\n")
	return sb.String()
}
