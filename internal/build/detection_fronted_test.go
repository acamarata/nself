package build

// detection_fronted_test.go — regression coverage for the 2026-09-03 staging
// incident's "nself status" symptom: ntask's status stayed at 6/7 forever
// because DetectServices always counts "nginx" as an expected service, even
// on a stack whose compose file never generates one (NGINX_FRONTED_BY set —
// see internal/compose/fronted_stack_test.go for the compose-side half of
// this fix). RunAllChecks (internal/health/checker.go) sizes its Total
// directly off len(DetectServices(cfg)), so this list being wrong is exactly
// what makes status report a permanently-unhealthy phantom service.
//
// Purpose: prove DetectServices excludes "nginx" once FrontedBy is set, and
// that the unset (default) case is unchanged.

import (
	"testing"

	"github.com/nself-org/cli/internal/config"
)

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func TestDetectServices_ExcludesNginxWhenFronted(t *testing.T) {
	cfg := &config.Config{}
	cfg.Nginx.FrontedBy = "nself-web"

	services := DetectServices(cfg)
	if containsString(services, "nginx") {
		t.Errorf("DetectServices returned nginx despite FrontedBy being set: %v", services)
	}
	for _, want := range []string{"postgres", "hasura", "auth"} {
		if !containsString(services, want) {
			t.Errorf("DetectServices is missing %q when fronted: %v", want, services)
		}
	}
}

func TestDetectServices_IncludesNginxByDefault(t *testing.T) {
	cfg := &config.Config{}
	// FrontedBy left empty — the default for every existing project.

	services := DetectServices(cfg)
	if !containsString(services, "nginx") {
		t.Errorf("DetectServices dropped nginx for an unfronted (default) stack — regression: %v", services)
	}
}

func TestExpectedCoreServices_ExcludesNginxWhenFronted(t *testing.T) {
	cfg := &config.Config{}
	cfg.Nginx.FrontedBy = "nself-web"

	services := expectedCoreServices(cfg)
	if containsString(services, "nginx") {
		t.Errorf("expectedCoreServices returned nginx despite FrontedBy being set: %v", services)
	}
}
