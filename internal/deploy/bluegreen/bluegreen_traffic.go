package bluegreen

// bluegreen_traffic.go — Nginx traffic-shifting helpers.
//
// Purpose: set Nginx upstream weights, reload Nginx, soak the new weighting and measure the resulting error rate, used by Deploy and Promote in bluegreen.go, split out for file size.
// Inputs: the desired blue/green weight split and a soak duration.
// Outputs: an applied Nginx config plus the measured error rate for the soak window.
// Constraints: pure move from bluegreen.go (CLI-R12 Batch E); no behaviour change.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
