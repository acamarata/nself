package doctor

// hardening_check_nginx_zones.go — SEC-HARDENING-06: nginx rate-limit zones
// for /auth/login and /api/. Split out of hardening_check_auth_net.go
// (CLI-R12) as a pure move.
//
// Inputs: a context (for the docker-exec fallback) and the project directory.
// Outputs: a single CheckResult — pass/warn/fail with remediation hint.
// Constraints: depends on hardeningSection, defined elsewhere in this package.
// SPORT: cli/internal/doctor — decomposed from hardening_check.go (T-E2-06).

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/health"
)

// checkHardeningNginxRateZones verifies nginx has limit_req_zone + limit_req
// directives covering /auth/login and /api/, first by scanning local config
// files, then by grepping inside the running nginx container as a fallback.
func checkHardeningNginxRateZones(ctx context.Context, projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-06"

	// Search nginx/conf.d/ and nginx/sites/ for limit_req_zone + limit_req directives
	// covering the two required paths.
	nginxDirs := []string{
		filepath.Join(projectDir, "nginx", "conf.d"),
		filepath.Join(projectDir, "nginx", "sites"),
		filepath.Join(projectDir, "nginx", "nginx.conf"),
	}

	hasAuthZone := false
	hasAPIZone := false

	for _, root := range nginxDirs {
		info, err := os.Stat(root)
		if err != nil {
			continue
		}

		var files []string
		if info.IsDir() {
			entries, err := os.ReadDir(root)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					files = append(files, filepath.Join(root, e.Name()))
				}
			}
		} else {
			files = []string{root}
		}

		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			content := string(data)
			if strings.Contains(content, "/auth/login") && strings.Contains(content, "limit_req") {
				hasAuthZone = true
			}
			if strings.Contains(content, "/api/") && strings.Contains(content, "limit_req") {
				hasAPIZone = true
			}
		}
	}

	// Fallback: inspect nginx container config if local files not found.
	if !hasAuthZone || !hasAPIZone {
		nginxContainer := health.ContainerName(resolveProjectName(projectDir), "nginx")
		cmd := exec.CommandContext(ctx, "docker", "exec", nginxContainer,
			"grep", "-r", "limit_req", "/etc/nginx/")
		out, err := cmd.Output()
		if err == nil {
			content := string(out)
			if strings.Contains(content, "auth/login") {
				hasAuthZone = true
			}
			if strings.Contains(content, "/api/") {
				hasAPIZone = true
			}
		}
	}

	switch {
	case hasAuthZone && hasAPIZone:
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-HARDENING-06: nginx rate-limit zones set for /auth/login and /api/",
		}
	case !hasAuthZone && !hasAPIZone:
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "fail",
			Message: "SEC-HARDENING-06: nginx missing rate-limit zones for /auth/login and /api/ — add limit_req_zone directives",
			FixCmd:  "See nself.org/docs/security/nginx-rate-limiting",
		}
	default:
		missing := "/api/"
		if !hasAuthZone {
			missing = "/auth/login"
		}
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-06: nginx rate-limit zone missing for %s — add limit_req_zone directive", missing),
			FixCmd:  "See nself.org/docs/security/nginx-rate-limiting",
		}
	}
}
