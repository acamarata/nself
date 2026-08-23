package commands

// Purpose: shared deploy-strategy tables and the rolling-restart and health-check
// helpers used by runDeploy. Inputs are the target workdir and strategy name;
// outputs are per-step results consumed by the deploy command's summary.
// Constraints: split out of deploy.go (CLI-R12) as a pure move, no behavior change.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// notYetImplementedStrategies lists strategies that fall back to rolling with
// an explicit warning. Tracked for v1.1.0.
// Note: blue-green and canary are now implemented via --canary N when the
// blue_green_deploy feature flag (Y17) is ON. The --strategy=blue-green/canary
// path still falls back to rolling for backwards compat with existing scripts.
var notYetImplementedStrategies = map[string]bool{
	"blue-green": true,
	"canary":     true,
	"preview":    true,
}

// deployServiceOrder defines the sequenced restart order for the rolling
// strategy. Services are restarted in dependency order so each layer is
// healthy before the next layer comes up.
var deployServiceOrder = []string{
	"postgres",
	"hasura",
	"auth",
	"storage",
	"plugins",
}

// runRollingRestart performs a per-service sequenced restart with health-
// gating between each service. It iterates over deployServiceOrder, calling
// "docker compose up -d <service>" per entry and waiting up to 60s for
// service_healthy before continuing. The deploy halts on first unhealthy
// service with a clear error and a pointer to nself logs.
func runRollingRestart(ctx context.Context, workdir string, jsonOut bool) ([]deployStep, error) {
	steps := []deployStep{}
	for _, svc := range deployServiceOrder {
		if !jsonOut {
			fmt.Printf("  [running] Restart %s (sequenced rolling)\n", svc)
		}
		// Restart the service.
		c := exec.CommandContext(ctx, "docker", "compose", "up", "-d", "--no-deps", svc)
		c.Dir = workdir
		c.Env = os.Environ()
		if out, err := c.CombinedOutput(); err != nil {
			steps = append(steps, deployStep{Name: fmt.Sprintf("Restart %s", svc), Status: "failed"})
			return steps, fmt.Errorf("rolling restart: service %s restart failed: %w\nOutput: %s\nRun 'nself logs %s' for details", svc, err, strings.TrimSpace(string(out)), svc)
		}

		// Health-gate: poll for service_healthy up to 60s.
		if !jsonOut {
			fmt.Printf("  [waiting] Waiting for service_healthy: %s (max 60s)\n", svc)
		}
		deadline := time.Now().Add(60 * time.Second)
		healthy := false
		for time.Now().Before(deadline) {
			out, err := exec.CommandContext(ctx, "docker", "compose", "ps", "--format", "{{.Name}}\t{{.Health}}", svc).Output()
			if err == nil {
				line := strings.TrimSpace(string(out))
				if strings.Contains(line, "healthy") && !strings.Contains(line, "unhealthy") {
					healthy = true
					break
				}
			}
			time.Sleep(2 * time.Second)
		}
		if !healthy {
			steps = append(steps, deployStep{Name: fmt.Sprintf("Restart %s", svc), Status: "unhealthy"})
			return steps, fmt.Errorf("rolling restart: service %s did not become healthy within 60s. Run 'nself logs %s' for details", svc, svc)
		}
		steps = append(steps, deployStep{Name: fmt.Sprintf("Restart %s", svc), Status: "done"})
		if !jsonOut {
			fmt.Printf("  [done] %s healthy\n", svc)
		}
	}
	return steps, nil
}

// runDeployHealthCheck calls nself doctor and gates the deploy result.
// Returns an error with the failed service name when any service is unhealthy.
func runDeployHealthCheck(ctx context.Context, workdir string, jsonOut bool) (deployStep, error) {
	if !jsonOut {
		fmt.Println("  [running] Health checks (calling nself health)")
	}
	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin, _ = exec.LookPath("nself")
	}
	if bin == "" {
		return deployStep{Name: "Health checks", Status: "failed"}, fmt.Errorf("unable to locate nself binary for health check")
	}
	c := exec.CommandContext(ctx, bin, "doctor")
	c.Dir = workdir
	c.Env = os.Environ()
	out, err := c.CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		// Try to extract the failing service from doctor output.
		failedSvc := "unknown"
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "unhealthy") || strings.Contains(line, "failed") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					failedSvc = parts[0]
					break
				}
			}
		}
		return deployStep{Name: "Health checks", Status: "failed"},
			fmt.Errorf("health check failed (service: %s). Run 'nself doctor --verbose' for details", failedSvc)
	}
	if !jsonOut {
		fmt.Println("  [done] Health checks passed")
	}
	return deployStep{Name: "Health checks", Status: "done"}, nil
}
