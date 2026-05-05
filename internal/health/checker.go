package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/httptimeout"
	saferecover "github.com/nself-org/cli/internal/recover"
)

// HealthResult holds the outcome of a single service health check.
type HealthResult struct {
	Service  string        `json:"service"`
	Status   string        `json:"status"` // healthy, unhealthy, starting, none, not_found, error
	Duration time.Duration `json:"duration"`
	Details  string        `json:"details"`
}

// HealthReport aggregates the results of checking all services.
type HealthReport struct {
	Timestamp time.Time      `json:"timestamp"`
	Results   []HealthResult `json:"results"`
	Healthy   int            `json:"healthy"`
	Unhealthy int            `json:"unhealthy"`
	Total     int            `json:"total"`
}

// containerName returns the Docker container name for a given service.
// nSelf compose uses the pattern {ProjectName}_{service} with hyphens replaced
// by underscores in the service portion.
func containerName(projectName, service string) string {
	svc := strings.ReplaceAll(service, "-", "_")
	return fmt.Sprintf("%s_%s", projectName, svc)
}

// RunAllChecks queries Docker health status for every enabled service and
// returns a consolidated HealthReport.
//
// It uses docker compose ps --format json (via the compose manifest) to query
// health status directly from the Docker Compose project, which is immune to
// container name guessing issues. It falls back to per-container docker inspect
// for any service not reported by compose ps.
func RunAllChecks(ctx context.Context, cfg *config.Config, workdir string) (*HealthReport, error) {
	services := build.DetectServices(cfg)

	report := &HealthReport{
		Timestamp: time.Now(),
		Results:   make([]HealthResult, 0, len(services)),
		Total:     len(services),
	}

	// Build a health map from docker compose ps output. This is more reliable
	// than docker inspect because it queries the Compose project context directly
	// and does not require us to guess container names.
	composeHealth := buildComposeHealthMap(ctx, workdir)

	for _, svc := range services {
		result := resolveServiceHealth(ctx, cfg.ProjectName, svc, composeHealth)
		if result.Status == "healthy" || result.Status == "running" {
			report.Healthy++
		} else {
			report.Unhealthy++
		}
		report.Results = append(report.Results, *result)
	}

	return report, nil
}

// buildComposeHealthMap runs docker compose ps --format json and returns a map
// of both service name and full container name → health status string. On any
// error (compose not running, workdir missing, no manifest) it returns an empty
// map so callers fall back to docker inspect.
//
// The map is keyed by both the Docker Compose service name (e.g. "postgres")
// and the full container name (e.g. "nclaw_postgres") so that resolveServiceHealth
// can find a match regardless of which naming convention the caller uses.
func buildComposeHealthMap(ctx context.Context, workdir string) map[string]string {
	if workdir == "" {
		return make(map[string]string)
	}

	composeFiles, err := build.ReadComposeManifest(workdir)
	if err != nil || len(composeFiles) == 0 {
		return make(map[string]string)
	}

	comp := docker.NewCompose(composeFiles...)
	infos, err := comp.ComposePs(ctx, workdir)
	if err != nil {
		return make(map[string]string)
	}

	m := make(map[string]string, len(infos)*2)
	for _, info := range infos {
		// docker compose ps --format json Health field values:
		//   "healthy"   — healthcheck passing
		//   "unhealthy" — healthcheck failing
		//   "starting"  — healthcheck running but not yet passed
		//   ""          — no healthcheck configured for this service
		status := info.Health
		if status == "" {
			// No healthcheck configured — container is running without one.
			// Use "running" so it is not counted as unhealthy.
			status = "running"
		}
		// Index by Docker Compose service name for direct lookup.
		if info.Service != "" {
			m[info.Service] = status
		}
		// Also index by full container name as a fallback.
		if info.Name != "" {
			m[info.Name] = status
		}
	}
	return m
}

// resolveServiceHealth looks up a service's health from the compose ps map
// first (by service name, then by container name variants), then falls back
// to a direct docker inspect call.
func resolveServiceHealth(ctx context.Context, projectName, service string, composeHealth map[string]string) *HealthResult {
	// Primary lookup: Docker Compose service name (most reliable — matches
	// exactly what DetectServices returns and what docker compose ps reports).
	if status, ok := composeHealth[service]; ok {
		return &HealthResult{
			Service: service,
			Status:  status,
			Details: fmt.Sprintf("compose service %s: %s", service, status),
		}
	}

	// Secondary lookup: explicit container_name used by nSelf compose generator.
	// Pattern: {project}_{service} with hyphens → underscores.
	cname := containerName(projectName, service)
	if status, ok := composeHealth[cname]; ok {
		return &HealthResult{
			Service: service,
			Status:  status,
			Details: fmt.Sprintf("container %s: %s", cname, status),
		}
	}

	// Tertiary lookup: Docker Compose v2 auto-naming ({project}-{service}-1).
	// Used when container_name is not set in the compose file.
	cnamev2 := fmt.Sprintf("%s-%s-1", projectName, service)
	if status, ok := composeHealth[cnamev2]; ok {
		return &HealthResult{
			Service: service,
			Status:  status,
			Details: fmt.Sprintf("container %s: %s", cnamev2, status),
		}
	}

	// Fallback: docker inspect per container (handles cases where compose ps
	// is not available or the manifest is missing).
	result, err := checkContainer(ctx, projectName, service)
	if err != nil {
		return &HealthResult{
			Service: service,
			Status:  "error",
			Details: err.Error(),
		}
	}
	return result
}

// CheckService checks the Docker health status of a single named service.
// The service name should match the docker-compose service name (e.g. "postgres",
// "hasura", "redis") or a full container name.
func CheckService(ctx context.Context, service string) (*HealthResult, error) {
	start := time.Now()
	status, err := docker.GetHealthStatus(ctx, service)
	elapsed := time.Since(start)
	if err != nil {
		return &HealthResult{
			Service:  service,
			Status:   "error",
			Duration: elapsed,
			Details:  err.Error(),
		}, err
	}
	if status == "not_found" {
		return &HealthResult{
			Service:  service,
			Status:   status,
			Duration: elapsed,
			Details:  fmt.Sprintf("container %q not found", service),
		}, fmt.Errorf("service %q: %w", service, errs.ErrServiceNotFound)
	}
	if status == "unhealthy" {
		return &HealthResult{
			Service:  service,
			Status:   status,
			Duration: elapsed,
			Details:  fmt.Sprintf("docker health: %s", status),
		}, fmt.Errorf("service %q: %w", service, errs.ErrServiceUnhealthy)
	}
	return &HealthResult{
		Service:  service,
		Status:   status,
		Duration: elapsed,
		Details:  fmt.Sprintf("docker health: %s", status),
	}, nil
}

// CheckEndpoint performs an HTTP GET against the given URL and reports whether
// it returned a 2xx status code. The request uses a 10-second timeout by default,
// or inherits the context deadline if shorter.
func CheckEndpoint(ctx context.Context, url string) (*HealthResult, error) {
	const defaultTimeout = 10 * time.Second

	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return &HealthResult{
			Service: url,
			Status:  "error",
			Details: fmt.Sprintf("bad request: %v", err),
		}, err
	}

	start := time.Now()
	resp, err := httptimeout.Health.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return &HealthResult{
			Service:  url,
			Status:   "unhealthy",
			Duration: elapsed,
			Details:  err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	status := "unhealthy"
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		status = "healthy"
	}

	return &HealthResult{
		Service:  url,
		Status:   status,
		Duration: elapsed,
		Details:  fmt.Sprintf("HTTP %d", resp.StatusCode),
	}, nil
}

// WatchHealth starts a continuous monitoring loop that sends a fresh HealthReport
// on the returned channel at the given interval. The loop exits when ctx is
// cancelled; the channel is closed on exit.
func WatchHealth(ctx context.Context, cfg *config.Config, workdir string, interval time.Duration) (<-chan *HealthReport, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("interval must be positive, got %v", interval)
	}

	ch := make(chan *HealthReport)

	saferecover.SafeGo("health_watch", func() {
		defer close(ch)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run an immediate first check before waiting for the ticker.
		if report, err := RunAllChecks(ctx, cfg, workdir); err == nil {
			select {
			case ch <- report:
			case <-ctx.Done():
				return
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				report, err := RunAllChecks(ctx, cfg, workdir)
				if err != nil {
					continue
				}
				select {
				case ch <- report:
				case <-ctx.Done():
					return
				}
			}
		}
	})

	return ch, nil
}

// checkContainer queries the health of a single service by its compose service
// name, resolving the full container name from the project prefix.
func checkContainer(ctx context.Context, projectName, service string) (*HealthResult, error) {
	name := containerName(projectName, service)
	start := time.Now()
	status, err := docker.GetHealthStatus(ctx, name)
	elapsed := time.Since(start)
	if err != nil {
		return nil, err
	}
	return &HealthResult{
		Service:  service,
		Status:   status,
		Duration: elapsed,
		Details:  fmt.Sprintf("container %s: %s", name, status),
	}, nil
}
