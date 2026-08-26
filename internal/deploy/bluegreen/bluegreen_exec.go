package bluegreen

// bluegreen_exec.go — container lifecycle helpers for the blue/green pipeline.
//
// Purpose: apply config defaults and drive the docker compose project through pull, start-green, stop-green/blue and health-wait, used by Deploy and Rollback in bluegreen.go, split out for file size.
// Inputs: a DeployConfig and the compose project name for the blue or green side.
// Outputs: started or stopped containers, or an error if a step fails or health never converges.
// Constraints: pure move from bluegreen.go (CLI-R12 Batch E); no behaviour change.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

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
//
// Readiness uses the same containerHealthy predicate and the same
// {{.Name}}\t{{.State}}\t{{.Health}} sampling as the soak gate's
// measureErrorRate (bluegreen_traffic.go). This used to be a separate,
// stricter check that required every docker compose ps line to literally
// contain the substring "healthy". A container with no Docker healthcheck
// (nginx, auth, any CS_N custom service) always reports empty Health, so it
// never contained "healthy" and this wait could never succeed for a
// perfectly good environment: the opposite failure from #270's fail-open
// soak gate, on the same underlying question of what counts as healthy.
// A "starting" Health value is still not ready, so the wait correctly keeps
// polling instead of declaring readiness prematurely.
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
			"ps", "--format", containerStatusFormat,
		).Output()
		if err == nil {
			if ready, total := allContainersReady(string(out)); ready && total > 0 {
				return nil
			}
		}
		_ = portOffset // used for future HTTP health check implementation
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("%s containers did not become healthy within %s", env, timeout)
}
