package health

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/ui"
)

// DockerClient abstracts the Docker operations needed by Restarter for testability.
type DockerClient interface {
	// ContainerList returns running containers, optionally filtered by label.
	ContainerList(ctx context.Context, filters map[string]string) ([]RestartContainer, error)
	// ContainerInspect returns detailed info for a single container by ID or name.
	ContainerInspect(ctx context.Context, id string) (RestartContainerInfo, error)
	// ContainerRestart restarts a container. timeout is in seconds; 0 uses Docker default.
	ContainerRestart(ctx context.Context, id string, timeout int) error
}

// RestartContainer is a minimal container descriptor returned by ContainerList.
type RestartContainer struct {
	ID      string
	Name    string
	Service string
}

// RestartContainerInfo holds the health state of an inspected container.
type RestartContainerInfo struct {
	ID     string
	Health string // healthy, unhealthy, starting, none
}

// RestartPolicy configures the behaviour of a Restarter.
type RestartPolicy struct {
	MaxAttempts  int           // default: 3
	PollInterval time.Duration // default: 30s
	RestartDelay time.Duration // default: 5s
}

// DefaultRestartPolicy returns a RestartPolicy with sensible defaults, overridden
// by the NSELF_HEALTH_POLL_INTERVAL and NSELF_HEALTH_MAX_RESTARTS env vars if set.
func DefaultRestartPolicy() RestartPolicy {
	return defaultPolicy()
}

// defaultPolicy is the unexported implementation used internally and in tests.
func defaultPolicy() RestartPolicy {
	p := RestartPolicy{
		MaxAttempts:  3,
		PollInterval: 30 * time.Second,
		RestartDelay: 5 * time.Second,
	}

	if v := os.Getenv("NSELF_HEALTH_MAX_RESTARTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.MaxAttempts = n
		}
	}

	if v := os.Getenv("NSELF_HEALTH_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			p.PollInterval = d
		}
	}

	return p
}

// ServiceState records the restart history for a single service.
type ServiceState struct {
	Service       string
	Attempts      int
	LastRestart   time.Time
	PermanentFail bool
}

// Restarter polls container health and automatically restarts unhealthy services
// according to the configured RestartPolicy.
type Restarter struct {
	docker DockerClient
	policy RestartPolicy

	mu     sync.Mutex
	states map[string]*ServiceState // keyed by service name
}

// NewRestarter creates a new Restarter. Call Start to begin polling.
func NewRestarter(docker DockerClient, policy RestartPolicy) *Restarter {
	return &Restarter{
		docker: docker,
		policy: policy,
		states: make(map[string]*ServiceState),
	}
}

// Start begins the health-poll loop in a goroutine. It returns immediately.
// Cancel ctx to stop the loop cleanly.
func (r *Restarter) Start(ctx context.Context) error {
	if r.policy.PollInterval <= 0 {
		return fmt.Errorf("restarter: PollInterval must be positive")
	}

	go r.loop(ctx)
	return nil
}

// Status returns a snapshot of all tracked service states.
func (r *Restarter) Status() []ServiceState {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]ServiceState, 0, len(r.states))
	for _, s := range r.states {
		out = append(out, *s)
	}
	return out
}

// loop is the polling goroutine body.
func (r *Restarter) loop(ctx context.Context) {
	ticker := time.NewTicker(r.policy.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.poll(ctx)
		}
	}
}

// poll lists containers and checks health once.
func (r *Restarter) poll(ctx context.Context) {
	containers, err := r.docker.ContainerList(ctx, nil)
	if err != nil {
		// Transient Docker error; will retry next tick.
		return
	}

	for _, c := range containers {
		info, err := r.docker.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}

		if info.Health != "unhealthy" {
			// Not unhealthy: reset attempts so a recovered service can be
			// restarted again if it becomes unhealthy later.
			r.mu.Lock()
			if st, ok := r.states[c.Service]; ok && !st.PermanentFail {
				st.Attempts = 0
			}
			r.mu.Unlock()
			continue
		}

		r.mu.Lock()
		st, ok := r.states[c.Service]
		if !ok {
			st = &ServiceState{Service: c.Service}
			r.states[c.Service] = st
		}

		if st.PermanentFail {
			r.mu.Unlock()
			continue
		}

		if st.Attempts >= r.policy.MaxAttempts {
			st.PermanentFail = true
			r.mu.Unlock()
			ui.Error(fmt.Sprintf("Service %s permanently failed after %d restart attempts", c.Service, r.policy.MaxAttempts))
			continue
		}

		// Snapshot mutable state before releasing the lock.
		attempt := st.Attempts + 1
		r.mu.Unlock()

		// Wait RestartDelay outside the lock to avoid blocking Status().
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.policy.RestartDelay):
		}

		ui.Warn(fmt.Sprintf("Service %s unhealthy — restarting (attempt %d/%d)", c.Service, attempt, r.policy.MaxAttempts))

		if err := r.docker.ContainerRestart(ctx, c.ID, 0); err != nil {
			ui.Warn(fmt.Sprintf("Service %s restart failed: %v", c.Service, err))
		}

		r.mu.Lock()
		// Count the attempt regardless of restart success so we don't loop forever.
		st.Attempts = attempt
		st.LastRestart = time.Now()
		r.mu.Unlock()
	}
}

// ── ShellDockerClient ─────────────────────────────────────────────────────────
// ShellDockerClient is a DockerClient implementation backed by the local docker
// package (shell exec). Use NewShellDockerClient to construct one.

// ShellDockerClient implements DockerClient by delegating to the local docker
// CLI and the docker.InspectContainer helper.
type ShellDockerClient struct {
	projectName string
}

// NewShellDockerClient returns a ShellDockerClient scoped to the given project.
// The project name is used to filter containers by the compose project label.
func NewShellDockerClient(projectName string) *ShellDockerClient {
	return &ShellDockerClient{projectName: projectName}
}

// ContainerList returns running containers for this project using docker ps.
func (s *ShellDockerClient) ContainerList(ctx context.Context, _ map[string]string) ([]RestartContainer, error) {
	filter := fmt.Sprintf("label=com.docker.compose.project=%s", s.projectName)
	cmd := exec.CommandContext(ctx, "docker", "ps",
		"--filter", filter,
		"--filter", "status=running",
		"--format", "{{.ID}}\t{{.Names}}\t{{.Label \"com.docker.compose.service\"}}",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}

	var result []RestartContainer
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		result = append(result, RestartContainer{
			ID:      parts[0],
			Name:    parts[1],
			Service: parts[2],
		})
	}
	return result, nil
}

// ContainerInspect returns health info for the given container ID using
// docker.InspectContainer.
func (s *ShellDockerClient) ContainerInspect(ctx context.Context, id string) (RestartContainerInfo, error) {
	info, err := docker.InspectContainer(ctx, id)
	if err != nil {
		return RestartContainerInfo{}, fmt.Errorf("inspecting container %s: %w", id, err)
	}
	return RestartContainerInfo{
		ID:     info.ID,
		Health: info.Health,
	}, nil
}

// ContainerRestart restarts the given container using docker restart.
func (s *ShellDockerClient) ContainerRestart(ctx context.Context, id string, timeout int) error {
	args := []string{"restart"}
	if timeout > 0 {
		args = append(args, "-t", strconv.Itoa(timeout))
	}
	args = append(args, id)
	cmd := exec.CommandContext(ctx, "docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker restart %s: %w\n%s", id, err, string(out))
	}
	return nil
}
