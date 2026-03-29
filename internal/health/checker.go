package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nself/internal/build"
	"nself/internal/config"
	"nself/internal/docker"
	"nself/internal/errs"
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
func RunAllChecks(ctx context.Context, cfg *config.Config) (*HealthReport, error) {
	services := build.DetectServices(cfg)

	report := &HealthReport{
		Timestamp: time.Now(),
		Results:   make([]HealthResult, 0, len(services)),
		Total:     len(services),
	}

	for _, svc := range services {
		result, err := checkContainer(ctx, cfg.ProjectName, svc)
		if err != nil {
			result = &HealthResult{
				Service: svc,
				Status:  "error",
				Details: err.Error(),
			}
		}
		if result.Status == "healthy" {
			report.Healthy++
		} else {
			report.Unhealthy++
		}
		report.Results = append(report.Results, *result)
	}

	return report, nil
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
	resp, err := http.DefaultClient.Do(req)
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
func WatchHealth(ctx context.Context, cfg *config.Config, interval time.Duration) (<-chan *HealthReport, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("interval must be positive, got %v", interval)
	}

	ch := make(chan *HealthReport)

	go func() {
		defer close(ch)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run an immediate first check before waiting for the ticker.
		if report, err := RunAllChecks(ctx, cfg); err == nil {
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
				report, err := RunAllChecks(ctx, cfg)
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
	}()

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
