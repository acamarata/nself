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
	"fmt"
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
