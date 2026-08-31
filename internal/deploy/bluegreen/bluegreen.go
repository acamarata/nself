// Package bluegreen implements zero-downtime blue/green and canary deploys
// for the nSelf CLI (B47 + B48).
//
// Architecture:
//   - Blue  = currently live containers (docker compose project: nself-blue)
//   - Green = shadow / new containers   (docker compose project: nself-green)
//
// On fresh install only blue exists. On deploy, green comes up alongside blue.
// Traffic is shifted via Nginx upstream weight configuration. Rollback in
// under 5 seconds by resetting Nginx weights and stopping green.
//
// Feature flag: blue_green_deploy (Y17). When OFF, callers should fall back to
// the existing rolling strategy. When ON, this package drives the deploy.
package bluegreen

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Environment labels used for docker compose projects.
const (
	EnvBlue  = "blue"
	EnvGreen = "green"
)

// StateFile is the path (relative to project root) where blue/green state is persisted.
const StateFile = ".nself/bluegreen/state.json"

// DefaultBluePortOffset is the port offset for blue containers (0 = base ports).
const DefaultBluePortOffset = 0

// DefaultGreenPortOffset is the port offset for green containers (+100).
const DefaultGreenPortOffset = 100

// DeployConfig holds all parameters for a blue/green or canary deploy.
type DeployConfig struct {
	// ProjectRoot is the nSelf project root (contains docker-compose.yml).
	ProjectRoot string

	// CanaryPercent is the initial canary traffic percentage (1-99).
	// 0 means skip canary and go straight to full flip.
	CanaryPercent int

	// SoakMinutes is the canary soak period in minutes. Default: 5.
	SoakMinutes int

	// ErrorThresholdPct is the error rate percentage that triggers auto-rollback.
	// Default: 1.0.
	ErrorThresholdPct float64

	// HealthTimeoutSec is the number of seconds to wait for green health.
	// Default: 30.
	HealthTimeoutSec int

	// ForceMigration disables canary and performs a full downtime deploy.
	// Required when a migration is not backward-compatible.
	ForceMigration bool

	// SkipCanary skips the canary phase and flips to 100% immediately.
	SkipCanary bool

	// DryRun prints steps without executing them.
	DryRun bool

	// BluePortOffset is the port offset for blue containers. Default: 0.
	BluePortOffset int

	// GreenPortOffset is the port offset for green containers. Default: 100.
	GreenPortOffset int
}

// DeployState persists the current blue/green state to disk.
type DeployState struct {
	// Active is the currently live environment ("blue" or "green").
	Active string `json:"active"`

	// BlueVersion is the image tag running in blue.
	BlueVersion string `json:"blue_version,omitempty"`

	// GreenVersion is the image tag running in green.
	GreenVersion string `json:"green_version,omitempty"`

	// CanaryPercent is the current canary traffic split (0 = not in canary).
	CanaryPercent int `json:"canary_percent"`

	// LastDeploy is the timestamp of the last successful deploy.
	LastDeploy time.Time `json:"last_deploy"`

	// LastRollback is the timestamp of the last rollback, if any.
	LastRollback *time.Time `json:"last_rollback,omitempty"`
}

// DeployResult is the outcome of a blue/green or canary deploy.
type DeployResult struct {
	// Success is true when the deploy completed without error.
	Success bool

	// Steps is the ordered list of deploy steps with their status.
	Steps []DeployStep

	// Duration is how long the deploy took.
	Duration time.Duration

	// CanaryPercent is the final canary percentage (100 if fully promoted).
	CanaryPercent int

	// RolledBack is true when an auto-rollback was triggered during the soak.
	RolledBack bool

	// Error is set on failure.
	Error string
}

// DeployStep is a single step in the deploy sequence.
type DeployStep struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pending, running, done, failed, skipped
}

// RollbackResult is the outcome of a manual rollback.
type RollbackResult struct {
	// Success is true when the rollback completed without error.
	Success bool

	// Duration is how long the rollback took.
	Duration time.Duration

	// Error is set on failure.
	Error string
}

// MigrationCheckResult describes whether a migration is backward-compatible.
type MigrationCheckResult struct {
	// Compatible is true when all pending migrations are safe to run during canary.
	Compatible bool

	// IncompatibleFiles lists migration files that are NOT backward-compatible.
	IncompatibleFiles []string

	// Reason explains the incompatibility in human-readable terms.
	Reason string
}

// Deploy performs a blue/green canary deploy:
//  1. Pull new images (tagged as green).
//  2. Start green containers (docker compose -p nself-green up -d).
//  3. Health check green (30s timeout).
//  4. Route canary % to green via Nginx weight update.
//  5. Canary soak period with error-rate monitoring.
//  6. Promote to 100% green (or auto-rollback on error threshold).
//  7. Stop and remove blue containers.
//  8. Rename green -> blue in state file.
func Deploy(ctx context.Context, cfg DeployConfig) DeployResult {
	start := time.Now()
	applyDefaults(&cfg)

	steps := []DeployStep{}
	addStep := func(name, status string) {
		steps = append(steps, DeployStep{Name: name, Status: status})
	}

	fail := func(name, reason string, err error) DeployResult {
		addStep(name, "failed")
		return DeployResult{
			Success:  false,
			Steps:    steps,
			Duration: time.Since(start),
			Error:    fmt.Sprintf("%s: %v", reason, err),
		}
	}

	if cfg.DryRun {
		return dryRunDeploy(cfg, start)
	}

	// Step 1: Migration compatibility check.
	if !cfg.ForceMigration {
		check := CheckMigrationCompatibility(cfg.ProjectRoot)
		if !check.Compatible {
			return DeployResult{
				Success:  false,
				Steps:    []DeployStep{{Name: "Migration compatibility check", Status: "failed"}},
				Duration: time.Since(start),
				Error: fmt.Sprintf(
					"Migration %s is not backward-compatible. Run with --force-migration to apply (disables canary, full downtime deploy). Reason: %s",
					strings.Join(check.IncompatibleFiles, ", "), check.Reason,
				),
			}
		}
	}
	addStep("Migration compatibility check", "done")

	// Step 2: Pull new images for green.
	if err := pullImages(ctx, cfg.ProjectRoot); err != nil {
		return fail("Pull new images", "image pull failed", err)
	}
	addStep("Pull new images", "done")

	// Step 3: Start green containers.
	if err := startGreen(ctx, cfg); err != nil {
		return fail("Start green containers", "green startup failed", err)
	}
	addStep("Start green containers", "done")

	// Step 4: Health check green.
	healthTimeout := time.Duration(cfg.HealthTimeoutSec) * time.Second
	if err := waitForHealth(ctx, cfg, EnvGreen, healthTimeout); err != nil {
		_ = stopGreen(ctx, cfg)
		return fail("Health check green", "green unhealthy", err)
	}
	addStep("Health check green", "done")

	// Step 5: Canary traffic split.
	canaryPct := cfg.CanaryPercent
	if cfg.SkipCanary || canaryPct == 0 {
		addStep("Canary traffic split", "skipped")
	} else {
		if err := setNginxWeights(cfg, canaryPct); err != nil {
			_ = stopGreen(ctx, cfg)
			return fail("Canary traffic split", "nginx weight update failed", err)
		}
		addStep(fmt.Sprintf("Canary traffic split (%d%%)", canaryPct), "done")

		// Step 6: Soak period with auto-rollback.
		soakDuration := time.Duration(cfg.SoakMinutes) * time.Minute
		rolledBack, soakErr := soak(ctx, cfg, soakDuration)
		if rolledBack || soakErr != nil {
			_ = setNginxWeights(cfg, 0) // reset to blue-only
			_ = stopGreen(ctx, cfg)
			addStep("Canary soak", "failed (auto-rollback triggered)")
			return DeployResult{
				Success:       false,
				Steps:         steps,
				Duration:      time.Since(start),
				CanaryPercent: 0,
				RolledBack:    true,
				Error:         fmt.Sprintf("auto-rollback: error rate exceeded %.1f%% threshold during soak", cfg.ErrorThresholdPct),
			}
		}
		addStep(fmt.Sprintf("Canary soak (%d min)", cfg.SoakMinutes), "done")
	}

	// Step 7: Promote green to 100%.
	if err := setNginxWeights(cfg, 100); err != nil {
		_ = setNginxWeights(cfg, 0)
		_ = stopGreen(ctx, cfg)
		return fail("Promote green to 100%", "nginx promote failed", err)
	}
	addStep("Promote green to 100%", "done")

	// Step 8: Health check at 100%.
	if err := waitForHealth(ctx, cfg, EnvGreen, healthTimeout); err != nil {
		_ = setNginxWeights(cfg, 0)
		_ = stopGreen(ctx, cfg)
		return fail("Health check at 100%", "green unhealthy after full promote", err)
	}
	addStep("Health check green (100%)", "done")

	// Step 9: Stop blue containers.
	if err := stopBlue(ctx, cfg); err != nil {
		// Non-fatal: log but continue. Blue is idle, green is serving.
		addStep("Stop blue containers", "failed (non-fatal)")
	} else {
		addStep("Stop blue containers", "done")
	}

	// Step 10: Persist state (green is now blue for next deploy).
	if err := swapState(cfg.ProjectRoot); err != nil {
		addStep("Persist blue/green state", "failed (non-fatal)")
	} else {
		addStep("Persist blue/green state", "done")
	}

	return DeployResult{
		Success:       true,
		Steps:         steps,
		Duration:      time.Since(start),
		CanaryPercent: 100,
	}
}
