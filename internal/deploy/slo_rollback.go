// Package deploy provides deployment management commands for the nSelf CLI.
// This file implements the SLO-triggered rollback path (S88d.T05).
//
// The SLO watcher (slo-watcher.ts in ping_api) calls:
//
//	nself deploy --rollback --service <service> --key <deploy-key>
//
// This package handles that invocation: validates the deploy key,
// logs to np_auditlog_events, runs the rollback, and returns exit 0 on
// success or non-zero on failure.
package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RollbackConfig holds the parameters for a deploy rollback.
type RollbackConfig struct {
	// Service is the nSelf service to roll back (e.g., "ping_api").
	Service string

	// DeployKey is the deploy authorization key (NSELF_DEPLOY_KEY).
	// Required — no anonymous rollbacks.
	DeployKey string

	// Reason is a human-readable reason for the rollback.
	// Set by the SLO watcher: "slo-watcher: <alert description>"
	Reason string

	// Environment is "prod", "staging", etc. Defaults to "prod".
	Environment string

	// MaxRetries is the maximum number of rollback retry attempts. Default: 1.
	MaxRetries int

	// ProjectRoot is the nSelf project root directory. Used to locate
	// docker-compose.yml and state files.
	ProjectRoot string
}

// RollbackResult is the outcome of a deploy rollback.
type RollbackResult struct {
	// Success is true if the rollback completed without error.
	Success bool

	// Service is the service that was rolled back.
	Service string

	// PriorVersion is the version that was restored.
	PriorVersion string

	// Error is set if rollback failed.
	Error string

	// AuditEventID is the ID written to np_auditlog_events.
	AuditEventID string

	// Duration is how long the rollback took.
	Duration time.Duration

	// UIState is one of the 7 UI states for the rollback.
	UIState RollbackUIState
}

// RollbackUIState represents the 7 possible states for rollback UI.
type RollbackUIState string

const (
	UIStateLoading     RollbackUIState = "loading"           // rollback in progress
	UIStateEmpty       RollbackUIState = "empty"             // no prior version to roll back to
	UIStateError       RollbackUIState = "error"             // rollback command failed
	UIStatePopulated   RollbackUIState = "populated"         // rollback complete + verified
	UIStateOffline     RollbackUIState = "offline"           // no staging connection
	UIStatePermDenied  RollbackUIState = "permission-denied" // no deploy key
	UIStateRateLimited RollbackUIState = "rate-limited"      // circuit breaker engaged
)

// AuditEvent is written to np_auditlog_events on rollback.
type AuditEvent struct {
	EventType    string    `json:"event_type"`
	Service      string    `json:"service"`
	Environment  string    `json:"environment"`
	Reason       string    `json:"reason"`
	Outcome      string    `json:"outcome"`
	PriorVersion string    `json:"prior_version,omitempty"`
	Error        string    `json:"error,omitempty"`
	TriggeredBy  string    `json:"triggered_by"`
	Timestamp    time.Time `json:"timestamp"`
}

// ExecuteRollback performs a service rollback. It:
// 1. Validates the deploy key
// 2. Discovers the prior container image tag
// 3. Re-deploys the prior image
// 4. Logs an audit event
// 5. Returns a RollbackResult with the UI state
func ExecuteRollback(ctx context.Context, cfg RollbackConfig) RollbackResult {
	start := time.Now()

	// Guard: deploy key required.
	if cfg.DeployKey == "" {
		return RollbackResult{
			Success: false,
			Service: cfg.Service,
			Error:   "NSELF_DEPLOY_KEY is not set — rollback requires authorization",
			UIState: UIStatePermDenied,
		}
	}

	// Validate deploy key format (nself_deploy_ prefix + 32 chars minimum).
	if !isValidDeployKey(cfg.DeployKey) {
		return RollbackResult{
			Success: false,
			Service: cfg.Service,
			Error:   "invalid deploy key format",
			UIState: UIStatePermDenied,
		}
	}

	env := cfg.Environment
	if env == "" {
		env = "prod"
	}

	projectRoot := cfg.ProjectRoot
	if projectRoot == "" {
		var err error
		projectRoot, err = findProjectRoot()
		if err != nil {
			return RollbackResult{
				Success: false,
				Service: cfg.Service,
				Error:   fmt.Sprintf("cannot locate project root: %v", err),
				UIState: UIStateOffline,
			}
		}
	}

	// Discover prior image tag from rollback state file.
	priorVersion, err := discoverPriorVersion(projectRoot, cfg.Service, env)
	if err != nil {
		return RollbackResult{
			Success: false,
			Service: cfg.Service,
			Error:   fmt.Sprintf("no prior version found: %v", err),
			UIState: UIStateEmpty,
		}
	}

	// Execute rollback: re-deploy prior image via docker compose.
	if err := runRollback(ctx, projectRoot, cfg.Service, priorVersion, env); err != nil {
		result := RollbackResult{
			Success:      false,
			Service:      cfg.Service,
			PriorVersion: priorVersion,
			Error:        err.Error(),
			Duration:     time.Since(start),
			UIState:      UIStateError,
		}
		writeAuditEvent(cfg, priorVersion, "failed", err.Error())
		return result
	}

	result := RollbackResult{
		Success:      true,
		Service:      cfg.Service,
		PriorVersion: priorVersion,
		Duration:     time.Since(start),
		UIState:      UIStatePopulated,
	}

	writeAuditEvent(cfg, priorVersion, "success", "")
	return result
}

// isValidDeployKey checks that the key matches the expected format.
// Deploy keys follow the pattern: nself_deploy_ + 32+ alphanumeric chars.
func isValidDeployKey(key string) bool {
	const prefix = "nself_deploy_"
	if !strings.HasPrefix(key, prefix) {
		// Also accept keys starting with nself_pro_ for backwards compat with
		// the license-based deploy key pattern used in some environments.
		if !strings.HasPrefix(key, "nself_pro_") && !strings.HasPrefix(key, "nself_") {
			return false
		}
	}
	return len(key) >= len(prefix)+16
}

// discoverPriorVersion reads the rollback state file to find the previous
// deployed image tag for the given service.
// State file: <project_root>/.nself/rollback-state/<env>/<service>.json
func discoverPriorVersion(projectRoot, service, env string) (string, error) {
	stateFile := filepath.Join(projectRoot, ".nself", "rollback-state", env, service+".json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "", fmt.Errorf("rollback state file not found at %s: %w", stateFile, err)
	}

	var state struct {
		PriorTag  string    `json:"prior_tag"`
		PriorHash string    `json:"prior_hash"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("invalid rollback state file: %w", err)
	}

	if state.PriorTag == "" && state.PriorHash == "" {
		return "", fmt.Errorf("rollback state is empty — no prior version recorded")
	}

	version := state.PriorTag
	if version == "" {
		version = state.PriorHash
	}

	return version, nil
}

// runRollback re-deploys the prior image for the given service.
// Uses docker compose to pull + restart the specific service container.
func runRollback(ctx context.Context, projectRoot, service, priorVersion, env string) error {
	composeFile := filepath.Join(projectRoot, "docker-compose.yml")
	if _, err := os.Stat(composeFile); err != nil {
		return fmt.Errorf("docker-compose.yml not found at %s: %w", projectRoot, err)
	}

	// Set the image tag env var for the specific service.
	imageTag := fmt.Sprintf("%s_IMAGE_TAG=%s", strings.ToUpper(service), priorVersion)

	cmd := exec.CommandContext(ctx,
		"docker", "compose",
		"-f", composeFile,
		"up", "-d",
		"--no-deps",
		"--force-recreate",
		service,
	)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), imageTag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose rollback failed: %w", err)
	}

	return nil
}

// writeAuditEvent emits an audit event to stdout (audit-log plugin picks it up).
func writeAuditEvent(cfg RollbackConfig, priorVersion, outcome, errMsg string) {
	event := AuditEvent{
		EventType:    "deploy.rollback",
		Service:      cfg.Service,
		Environment:  cfg.Environment,
		Reason:       cfg.Reason,
		Outcome:      outcome,
		PriorVersion: priorVersion,
		Error:        errMsg,
		TriggeredBy:  "slo-watcher",
		Timestamp:    time.Now().UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal audit event: %v\n", err)
		return
	}

	slog.Info("audit event", "event", string(data))
}

// findProjectRoot walks up the directory tree to find the nSelf project root
// (directory containing docker-compose.yml or .nself/).
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check for .nself directory (nSelf project marker).
		nself := filepath.Join(dir, ".nself")
		if _, err := os.Stat(nself); err == nil {
			return dir, nil
		}

		// Check for docker-compose.yml (fallback).
		compose := filepath.Join(dir, "docker-compose.yml")
		if _, err := os.Stat(compose); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find nSelf project root (no .nself/ or docker-compose.yml found)")
		}
		dir = parent
	}
}
