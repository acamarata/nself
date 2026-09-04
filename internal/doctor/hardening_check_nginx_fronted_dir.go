package doctor

// hardening_check_nginx_fronted_dir.go — resolves which on-disk directory a
// fronted project's nginx checks should audit.
//
// Purpose: split out of hardening_check_nginx_zones.go so "which directory
// do we audit" (a filesystem/topology question) stays separate from "does
// that directory's config carry the right rate limits" (a content-scanning
// question). Also gives a future ResolveServedNginxDir generalization (any
// check that needs the served nginx dir, not just SEC-HARDENING-06) a home.
// Inputs: projectDir (the nSelf working directory) and the configured
// NGINX_FRONTED_BY stack name.
// Outputs: the resolved directory and whether resolution succeeded.
// Constraints: never guesses — returns ok=false rather than a wrong path.
// SPORT: cli/internal/doctor — SEC-HARDENING-06 fronted-topology fix.

import (
	"fmt"
	"path/filepath"

	"github.com/nself-org/cli/internal/config"
)

// resolveNginxFrontedDir resolves the on-disk nginx directory for a project
// fronted by another stack (NGINX_FRONTED_BY — see config.NginxConfig.FrontedBy).
//
// FrontedBy carries only a stack NAME (e.g. "nself-web"), never a path, and
// none of the codebase's other three consumers of it
// (internal/build/detection.go, internal/build/manifest_resolve.go,
// internal/compose/generator.go) resolve that name to a filesystem
// location — all three only test it for emptiness, to decide whether to
// generate this project's own nginx service. There is no config-driven
// convention anywhere in this CLI for turning a stack name into a directory
// in the general case, so this function does not attempt one.
//
// The one topology it trusts is the one that caused the regression this
// check exists to catch (cli#380/cli#371 on staging): this project living
// as a subdirectory (conventionally "backend") directly under the fronting
// stack's own directory, whose basename equals FrontedBy — e.g. projectDir
// ".../nself-web/backend" with FrontedBy "nself-web" resolves to
// ".../nself-web/nginx". That is a structural inference from the directory
// layout the operator already chose, not a second FrontedBy-to-path
// convention grafted onto config. When the parent directory's name does
// not match, this function returns ok=false — the caller must not guess
// further and silently audit the wrong directory.
func resolveNginxFrontedDir(projectDir, frontedBy string) (dir string, ok bool) {
	parent := filepath.Dir(projectDir)
	if parent == projectDir || filepath.Base(parent) != frontedBy {
		return "", false
	}
	return parent, true
}

// nginxAuditPlan is the outcome of deciding which directory
// checkHardeningNginxRateZones should scan for SEC-HARDENING-06. skip is
// non-nil when the plan could not be resolved for a fronted project — the
// caller must return it as-is rather than falling through to a scan.
type nginxAuditPlan struct {
	auditDir           string
	skipDockerFallback bool
	skip               *CheckResult
}

// planNginxAudit decides which directory's nginx/** SEC-HARDENING-06 should
// read for projectDir.
//
// Fronted deployments (cli#380/cli#371 staging regression): when
// NGINX_FRONTED_BY names another stack as this project's ingress, this
// project's own <projectDir>/nginx/** is not what the running nginx reads
// — the fronting stack's nginx directory is, and this project generates no
// nginx container of its own to fall back to (internal/build/detection.go).
// Auditing <projectDir>/nginx/** unconditionally let SEC-HARDENING-06 pass
// or fail off a directory nginx never serves from.
//
// When unfronted (the default), the plan is unchanged from before this
// fix: auditDir is projectDir itself and the docker-exec fallback applies.
// When fronted and resolveNginxFrontedDir cannot confirm the real
// directory, skip carries a "skip" CheckResult stating why, so the caller
// never silently scans (and passes or fails on) the wrong directory.
func planNginxAudit(projectDir string) nginxAuditPlan {
	plan := nginxAuditPlan{auditDir: projectDir}

	cfg, err := config.Load(projectDir)
	if err != nil || cfg.Nginx.FrontedBy == "" {
		return plan
	}

	// Fronted: this project has no nginx container of its own to exec
	// into (internal/build/detection.go), regardless of whether the real
	// directory below resolves.
	plan.skipDockerFallback = true

	resolved, ok := resolveNginxFrontedDir(projectDir, cfg.Nginx.FrontedBy)
	if !ok {
		plan.skip = &CheckResult{
			Section: hardeningSection,
			Name:    "SEC-HARDENING-06",
			Status:  "skip",
			Message: fmt.Sprintf("SEC-HARDENING-06: skipped — audited nothing. Project is fronted by nginx stack %q (NGINX_FRONTED_BY) but its nginx directory could not be resolved from %q; %s's own nginx config was not scanned here and must be audited directly for rate-limit zones", cfg.Nginx.FrontedBy, projectDir, cfg.Nginx.FrontedBy),
		}
		return plan
	}

	plan.auditDir = resolved
	return plan
}
