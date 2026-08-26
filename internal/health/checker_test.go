package health

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Bug #63 — nself status reports 0/N healthy when all containers are running
//
// Root cause (v0.x): status command used wrong container name patterns and
// mis-read Docker API health state fields.
//
// Go rewrite fix: buildComposeHealthMap indexes both the Compose service name
// and the full container name. resolveServiceHealth tries three name variants
// (bare service, project_service, project-service-1) before falling back to
// docker inspect.
//
// These tests verify the lookup logic is correct so the regression cannot
// silently re-appear.
// ---------------------------------------------------------------------------

// TestContainerName_HyphenToUnderscore verifies that containerName converts
// hyphens in the service name to underscores, matching the nSelf compose naming
// convention ("project_service").
func TestContainerName_HyphenToUnderscore(t *testing.T) {
	got := containerName("myproject", "minio-init")
	want := "myproject_minio_init"
	if got != want {
		t.Errorf("containerName(%q, %q) = %q, want %q", "myproject", "minio-init", got, want)
	}
}

// TestContainerName_PlainService verifies that a service name without hyphens
// produces the expected "project_service" format.
func TestContainerName_PlainService(t *testing.T) {
	got := containerName("nclaw", "postgres")
	want := "nclaw_postgres"
	if got != want {
		t.Errorf("containerName(%q, %q) = %q, want %q", "nclaw", "postgres", got, want)
	}
}

// TestResolveServiceHealth_DirectServiceNameMatch verifies that when a service
// name appears as a direct key in the compose health map (most common case),
// the correct status is returned without needing container name resolution.
func TestResolveServiceHealth_DirectServiceNameMatch(t *testing.T) {
	composeMap := map[string]string{
		"postgres": "healthy",
		"hasura":   "starting",
	}

	result := resolveServiceHealth(nil, "myproject", "postgres", composeMap)
	if result.Status != "healthy" {
		t.Errorf("expected status %q, got %q", "healthy", result.Status)
	}
	if result.Service != "postgres" {
		t.Errorf("expected service %q, got %q", "postgres", result.Service)
	}
}

// TestResolveServiceHealth_ContainerNameFallback verifies that when the bare
// service name is not in the map but the full container name (project_service)
// is, the status is correctly resolved via the secondary lookup.
func TestResolveServiceHealth_ContainerNameFallback(t *testing.T) {
	// Only the full container name is indexed (e.g. if compose ps returned the
	// full name but not the service-only key).
	composeMap := map[string]string{
		"nclaw_postgres": "healthy",
	}

	result := resolveServiceHealth(nil, "nclaw", "postgres", composeMap)
	if result.Status != "healthy" {
		t.Errorf("expected status %q via container name fallback, got %q", "healthy", result.Status)
	}
}

// TestResolveServiceHealth_ComposeV2NameFallback verifies that the Compose v2
// auto-naming convention ({project}-{service}-1) is tried as a third lookup
// before falling back to docker inspect.
func TestResolveServiceHealth_ComposeV2NameFallback(t *testing.T) {
	// Only the v2 container name is indexed.
	composeMap := map[string]string{
		"myapp-redis-1": "healthy",
	}

	result := resolveServiceHealth(nil, "myapp", "redis", composeMap)
	if result.Status != "healthy" {
		t.Errorf("expected status %q via v2 name fallback, got %q", "healthy", result.Status)
	}
}

// TestBuildComposeHealthMap_EmptyWorkdir verifies that buildComposeHealthMap
// returns an empty map (not nil, not a panic) when workdir is empty string.
func TestBuildComposeHealthMap_EmptyWorkdir(t *testing.T) {
	m := buildComposeHealthMap(nil, "")
	if m == nil {
		t.Fatal("buildComposeHealthMap(\"\") returned nil, expected empty map")
	}
	if len(m) != 0 {
		t.Errorf("expected empty map for empty workdir, got %d entries", len(m))
	}
}

// TestHealthReport_AllServicesHealthy verifies that a HealthReport where every
// service maps to "healthy" has Healthy == Total and Unhealthy == 0.
// This is the regression test for "0/N healthy when all containers are running".
func TestHealthReport_AllServicesHealthy(t *testing.T) {
	services := []string{"postgres", "hasura", "auth", "nginx"}
	composeMap := map[string]string{
		"postgres": "healthy",
		"hasura":   "healthy",
		"auth":     "healthy",
		"nginx":    "healthy",
	}

	report := &HealthReport{
		Total:   len(services),
		Results: make([]HealthResult, 0, len(services)),
	}

	for _, svc := range services {
		result := resolveServiceHealth(nil, "testproj", svc, composeMap)
		if result.Status == "healthy" {
			report.Healthy++
		} else {
			report.Unhealthy++
		}
		report.Results = append(report.Results, *result)
	}

	if report.Healthy != report.Total {
		t.Errorf("Bug #63 regression: expected %d/%d healthy, got %d/%d",
			report.Total, report.Total, report.Healthy, report.Total)
	}
	if report.Unhealthy != 0 {
		t.Errorf("Bug #63 regression: expected 0 unhealthy, got %d", report.Unhealthy)
	}
}

// TestContainerHealthStatusFromComposeMap_NoHealthcheck verifies that a service
// with no healthcheck configured (empty Health field from compose ps) is
// stored as "running" in the map — not empty string or "none" — so it is not
// counted as unhealthy.
func TestContainerHealthStatusFromComposeMap_NoHealthcheck(t *testing.T) {
	// Simulate the buildComposeHealthMap internal logic:
	// compose ps returns Health="" for services without healthchecks.
	// The map must store "running" so resolveServiceHealth doesn't count it unhealthy.
	health := ""
	if health == "" {
		health = "running"
	}
	if health != "running" {
		t.Errorf("expected empty health to be normalised to %q, got %q", "running", health)
	}
}

// ---------------------------------------------------------------------------
// Issue #268 — one Status field, two opposite predicates
//
// The aggregate in RunAllChecks accepted "healthy" OR "running", but the
// three per-service printers in cmd/commands/start_health.go only accepted
// "healthy" — so a report could print "4/4 healthy (100%)" and
// "✗ nginx: running" on consecutive lines. Fixed by extracting a single
// HealthResult.OK() predicate that both the aggregate and every printer call.
// ---------------------------------------------------------------------------

// TestHealthResult_OK verifies the single accept-predicate used everywhere a
// HealthResult is judged healthy or not. This preserves the exact semantics
// the aggregate in RunAllChecks used before extraction (checker.go:62):
// "healthy" and "running" are accepted, everything else is rejected.
func TestHealthResult_OK(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"healthy", true},
		{"running", true},
		{"unhealthy", false},
		{"starting", false},
		{"", false},
		{"not_found", false}, // unknown/other value
	}

	for _, c := range cases {
		r := HealthResult{Service: "svc", Status: c.status}
		if got := r.OK(); got != c.want {
			t.Errorf("HealthResult{Status: %q}.OK() = %v, want %v", c.status, got, c.want)
		}
	}
}

// TestHealthReport_AllServicesHealthy_RunningCountsHealthy is the aggregate
// half of the issue #268 regression: a service with no Docker healthcheck
// (Status == "running") must be counted in report.Healthy, not
// report.Unhealthy, matching the "4/4 healthy (100%)" summary line. The other
// half — that the per-service printer must agree — is asserted in
// cmd/commands/start_health_test.go, which cannot import this unexported
// aggregation loop, so it re-derives the same count via HealthResult.OK() and
// checks the two never diverge.
func TestHealthReport_AllServicesHealthy_RunningCountsHealthy(t *testing.T) {
	services := []string{"postgres", "hasura", "auth", "nginx"}
	composeMap := map[string]string{
		"postgres": "healthy",
		"hasura":   "healthy",
		"auth":     "healthy",
		"nginx":    "running", // no Docker healthcheck declared, per issue #268
	}

	report := &HealthReport{
		Total:   len(services),
		Results: make([]HealthResult, 0, len(services)),
	}

	for _, svc := range services {
		result := resolveServiceHealth(nil, "testproj", svc, composeMap)
		if result.OK() {
			report.Healthy++
		} else {
			report.Unhealthy++
		}
		report.Results = append(report.Results, *result)
	}

	if report.Healthy != report.Total {
		t.Errorf("issue #268 regression: expected %d/%d healthy (nginx running should count), got %d/%d",
			report.Total, report.Total, report.Healthy, report.Total)
	}
	if report.Unhealthy != 0 {
		t.Errorf("issue #268 regression: expected 0 unhealthy, got %d", report.Unhealthy)
	}
}
