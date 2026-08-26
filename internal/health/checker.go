package health

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/docker"
)

// HealthResult holds the outcome of a single service health check.
type HealthResult struct {
	Service  string        `json:"service"`
	Status   string        `json:"status"` // healthy, unhealthy, starting, none, not_found, error
	Duration time.Duration `json:"duration"`
	Details  string        `json:"details"`
}

// OK reports whether this result counts as healthy. A container with no
// Docker healthcheck configured reports "running" rather than "healthy"
// (see resolveServiceHealth's comment on buildComposeHealthMap) and is
// intentionally accepted here.
//
// This is the single decision point for what counts as healthy. Every
// caller — the aggregate count in RunAllChecks and every per-service
// printer in cmd/commands/start_health.go — must call this method rather
// than re-deriving the comparison, or the aggregate and per-service verdicts
// can silently drift apart again (see issue #268: a report could print
// "4/4 healthy (100%)" on one line and "✗ nginx: running" on the next,
// because the printers compared against only "healthy" while the aggregate
// also accepted "running").
func (r HealthResult) OK() bool {
	return r.Status == "healthy" || r.Status == "running"
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
		if result.OK() {
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
	comp.EnvFiles = build.ComposeEnvFiles(workdir)
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
