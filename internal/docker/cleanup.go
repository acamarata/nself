package docker

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CleanupInitContainers finds containers with label "nself.type=init-container"
// and force-removes them. Init containers (minio-init, meilisearch-init, etc.)
// complete their job and sit in "exited" state. This cleans them up.
func cleanupInitContainers(ctx context.Context) error {
	ids, err := findContainersByLabel(ctx, "nself.type=init-container")
	if err != nil {
		return fmt.Errorf("finding init containers: %w", err)
	}
	for _, id := range ids {
		if err := forceRemoveContainer(ctx, id); err != nil {
			// Best-effort: log but don't fail on individual removals.
			continue
		}
	}
	return nil
}

// CleanupZombieContainers finds containers in "created" state belonging to the
// given compose project and removes them. These are zombie containers left from
// interrupted starts.
func cleanupZombieContainers(ctx context.Context, projectName string) error {
	// Find containers belonging to this project that are in "created" state.
	filter := fmt.Sprintf("com.docker.compose.project=%s", projectName)
	ids, err := findContainersByLabelAndStatus(ctx, filter, "created")
	if err != nil {
		return fmt.Errorf("finding zombie containers: %w", err)
	}
	for _, id := range ids {
		if err := forceRemoveContainer(ctx, id); err != nil {
			continue
		}
	}
	return nil
}

// RunPostStartCleanup runs the full post-startup cleanup sequence:
//  1. Init container cleanup, repeated 3 times with 500ms delays (handles race conditions
//     where containers are still being marked as exited).
//  2. Zombie container cleanup for containers stuck in "created" state.
func RunPostStartCleanup(ctx context.Context, projectName string) error {
	// Run init container cleanup 3 times with delays to handle race conditions.
	for i := 0; i < 3; i++ {
		if err := cleanupInitContainers(ctx); err != nil {
			// Non-fatal: continue with remaining passes.
			_ = err
		}
		if i < 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}

	// Clean up zombie containers from interrupted starts.
	if err := cleanupZombieContainers(ctx, projectName); err != nil {
		// Non-fatal: best effort.
		_ = err
	}

	return nil
}

// findContainersByLabel runs `docker ps -aq --filter label=<label>` and returns
// the container IDs (including stopped containers).
func findContainersByLabel(ctx context.Context, label string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", fmt.Sprintf("label=%s", label))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps filter label=%s: %w", label, err)
	}
	return parseLines(string(out)), nil
}

// findContainersByLabelAndStatus runs `docker ps -aq` filtering by both a label
// and a container status (e.g., "created", "exited").
func findContainersByLabelAndStatus(ctx context.Context, label, status string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-aq",
		"--filter", fmt.Sprintf("label=%s", label),
		"--filter", fmt.Sprintf("status=%s", status),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps filter label=%s status=%s: %w", label, status, err)
	}
	return parseLines(string(out)), nil
}

// forceRemoveContainer runs `docker rm -f <id>`.
func forceRemoveContainer(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", id)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker rm -f %s: %w (%s)", id, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// parseLines splits command output into non-empty trimmed lines.
func parseLines(output string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
