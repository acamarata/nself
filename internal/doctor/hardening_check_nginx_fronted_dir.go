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

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/nginxtopo"
)

// resolveNginxFrontedDir resolves the on-disk nginx directory for a project
// fronted by another stack (NGINX_FRONTED_BY — see config.NginxConfig.FrontedBy).
//
// FrontedBy carries only a stack NAME (e.g. "nself-web"), never a path.
// Turning it into a filesystem location is internal/nginxtopo.
// ResolveFrontingDir's job — this wrapper exists only so this file's
// existing callers (planNginxAudit) keep their local name. See that
// package for the structural convention this delegates to (cli#380/cli#371
// on staging; reused as-is by the nginx generator for cli#385 rather than
// inventing a second convention).
func resolveNginxFrontedDir(projectDir, frontedBy string) (dir string, ok bool) {
	return nginxtopo.ResolveFrontingDir(projectDir, frontedBy)
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
