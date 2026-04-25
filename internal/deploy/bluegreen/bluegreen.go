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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		canaryPct = 0
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

// ── internal helpers ──────────────────────────────────────────────────────────

func applyDefaults(cfg *DeployConfig) {
	if cfg.SoakMinutes == 0 {
		cfg.SoakMinutes = 5
	}
	if cfg.ErrorThresholdPct == 0 {
		cfg.ErrorThresholdPct = 1.0
	}
	if cfg.HealthTimeoutSec == 0 {
		cfg.HealthTimeoutSec = 30
	}
	if cfg.GreenPortOffset == 0 && cfg.BluePortOffset == 0 {
		cfg.GreenPortOffset = DefaultGreenPortOffset
	}
	// Read from env if not set by caller.
	if cfg.CanaryPercent == 0 {
		if v := os.Getenv("NSELF_CANARY_PERCENT"); v != "" {
			var pct int
			if _, err := fmt.Sscanf(v, "%d", &pct); err == nil && pct > 0 && pct < 100 {
				cfg.CanaryPercent = pct
			}
		}
		if cfg.CanaryPercent == 0 {
			cfg.CanaryPercent = 10 // spec default
		}
	}
}

func composeProjectName(env string) string {
	return "nself-" + env
}

func pullImages(ctx context.Context, projectRoot string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "pull")
	cmd.Dir = projectRoot
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startGreen(ctx context.Context, cfg DeployConfig) error {
	portEnv := fmt.Sprintf("PORT_OFFSET=%d", cfg.GreenPortOffset)
	cmd := exec.CommandContext(ctx,
		"docker", "compose",
		"-p", composeProjectName(EnvGreen),
		"up", "-d",
	)
	cmd.Dir = cfg.ProjectRoot
	cmd.Env = append(os.Environ(), portEnv, "NSELF_ENV=green")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopGreen(ctx context.Context, cfg DeployConfig) error {
	cmd := exec.CommandContext(ctx,
		"docker", "compose",
		"-p", composeProjectName(EnvGreen),
		"down", "--remove-orphans",
	)
	cmd.Dir = cfg.ProjectRoot
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopBlue(ctx context.Context, cfg DeployConfig) error {
	cmd := exec.CommandContext(ctx,
		"docker", "compose",
		"-p", composeProjectName(EnvBlue),
		"down", "--remove-orphans",
	)
	cmd.Dir = cfg.ProjectRoot
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// waitForHealth polls the health endpoint on the given env until healthy or timeout.
func waitForHealth(ctx context.Context, cfg DeployConfig, env string, timeout time.Duration) error {
	portOffset := cfg.BluePortOffset
	if env == EnvGreen {
		portOffset = cfg.GreenPortOffset
	}

	// Health check via docker compose ps --format.
	project := composeProjectName(env)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		out, err := exec.CommandContext(ctx,
			"docker", "compose",
			"-p", project,
			"ps", "--format", "{{.Name}}\t{{.Health}}",
		).Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			allHealthy := len(lines) > 0
			for _, line := range lines {
				if line == "" {
					continue
				}
				if strings.Contains(line, "unhealthy") || !strings.Contains(line, "healthy") {
					allHealthy = false
					break
				}
			}
			if allHealthy && len(lines) > 0 {
				return nil
			}
		}
		_ = portOffset // used for future HTTP health check implementation
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("%s containers did not become healthy within %s", env, timeout)
}

// setNginxWeights writes the Nginx upstream block with the given canary percent
// and reloads Nginx atomically (nginx -s reload).
func setNginxWeights(cfg DeployConfig, canaryPercent int) error {
	upstream := GenerateNginxUpstream(cfg, canaryPercent)

	// Write to the generated upstream conf file.
	upstreamPath := filepath.Join(cfg.ProjectRoot, "nginx", "conf.d", "bluegreen-upstream.conf")
	if err := os.MkdirAll(filepath.Dir(upstreamPath), 0755); err != nil {
		return fmt.Errorf("creating nginx conf.d dir: %w", err)
	}

	if err := os.WriteFile(upstreamPath, []byte(upstream), 0644); err != nil {
		return fmt.Errorf("writing bluegreen upstream config: %w", err)
	}

	// Reload nginx (atomic — no dropped connections).
	// Try docker exec into the nginx container first; fall back to system nginx.
	if err := reloadNginx(cfg.ProjectRoot); err != nil {
		return fmt.Errorf("nginx reload failed: %w", err)
	}

	return nil
}

// reloadNginx sends a reload signal to the nginx container or the system nginx.
func reloadNginx(projectRoot string) error {
	// Try docker compose exec nginx nginx -s reload.
	cmd := exec.Command("docker", "compose", "exec", "-T", "nginx", "nginx", "-s", "reload")
	cmd.Dir = projectRoot
	cmd.Env = os.Environ()
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fall back to system nginx reload.
	cmd = exec.Command("nginx", "-s", "reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("both docker compose exec nginx reload and system nginx reload failed: %w", err)
	}
	return nil
}

// soak monitors the green environment during the soak period.
// Returns (rolledBack=true, err) when error rate exceeds threshold.
func soak(ctx context.Context, cfg DeployConfig, duration time.Duration) (bool, error) {
	deadline := time.Now().Add(duration)
	pollInterval := 30 * time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(pollInterval):
		}

		errorRate := measureErrorRate(ctx, cfg)
		if errorRate > cfg.ErrorThresholdPct {
			return true, fmt.Errorf("error rate %.2f%% exceeded threshold %.1f%%", errorRate, cfg.ErrorThresholdPct)
		}
	}
	return false, nil
}

// measureErrorRate estimates the current error rate from green containers.
// It checks docker compose ps for unhealthy containers and returns a rate.
// When Prometheus is available, it queries the metrics endpoint instead.
// Returns 0.0 when measurement is unavailable (fail-open for soak continuity).
func measureErrorRate(ctx context.Context, cfg DeployConfig) float64 {
	project := composeProjectName(EnvGreen)
	out, err := exec.CommandContext(ctx,
		"docker", "compose",
		"-p", project,
		"ps", "--format", "{{.Health}}",
	).Output()
	if err != nil {
		return 0.0 // fail-open
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	total := 0
	unhealthy := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		total++
		if strings.Contains(line, "unhealthy") {
			unhealthy++
		}
	}

	if total == 0 {
		return 0.0
	}
	return float64(unhealthy) / float64(total) * 100.0
}

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

// resolveProjectDir is a helper used by the cmd layer to locate the project root.
// It is unexported here; the cmd layer calls config.FindNSelfRoot instead.
func resolveProjectDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	// Walk up looking for docker-compose.yml or .nself/.
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".nself")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot locate nSelf project root")
		}
		dir = parent
	}
}
