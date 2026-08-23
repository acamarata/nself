package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Purpose: post-build nginx config syntax verification (`nginx -t`), plus the
// nginx.conf path resolver it depends on. Split out of postvalidate.go
// (CLI-R12) as a pure move.
// Inputs: the nginx config dir (nginx/ or nginx/sites/) and the shared
// *PostValidateResult to append findings to.
// Outputs: appends to result.Errors/Warnings; never returns a Go error.
// Constraints: skips (warn, not fail) when nginx is not installed locally.

// checkNginxSyntax runs `nginx -t -c <nginx.conf>` if the nginx binary is
// available in PATH and the main nginx.conf can be found. Skips with a
// warning when nginx is not installed locally — that is not an error,
// since users may not have nginx on their development machine.
//
// The nginxConfDir parameter is expected to be nginx/sites/ (or nginx/).
// We look for nginx.conf one level up (the standard layout).
func checkNginxSyntax(nginxConfDir string, result *PostValidateResult) {
	// Locate nginx binary.
	nginxBin, err := exec.LookPath("nginx")
	if err != nil {
		result.Warnings = append(result.Warnings,
			"nginx binary not found in PATH — skipping nginx syntax check (install nginx locally to enable)")
		return
	}

	// Determine path to main nginx.conf. The caller passes nginx/sites/,
	// so the main conf lives in the parent directory.
	nginxConf := findNginxConf(nginxConfDir)
	if nginxConf == "" {
		result.Warnings = append(result.Warnings,
			"nginx.conf not found — skipping nginx syntax check")
		return
	}

	// Run nginx -t with a short timeout so a hung binary never blocks the build.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// nginx -t writes its output to stderr by convention.
	cmd := exec.CommandContext(ctx, nginxBin, "-t", "-c", nginxConf)
	out, err := cmd.CombinedOutput()

	// nginx -t validates config SYNTAX and then attempts runtime probes
	// (e.g. opening the pid file from the `pid` directive). When run as a
	// non-root user without a writable pid path — the norm in CI and most
	// local `nself build` runs — nginx prints "syntax is ok" and then exits
	// non-zero with "[emerg] open() /var/run/nginx.pid failed (Permission
	// denied)". That is a runtime-permission artifact, not a config-syntax
	// error, so we gate on the syntax verdict in the output rather than the
	// exit code alone: if nginx reports the syntax OK, the check passes.
	outStr := string(out)
	if strings.Contains(outStr, "syntax is ok") {
		return
	}
	if err != nil {
		result.NginxValid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("nginx syntax check failed: %v\n%s", err, strings.TrimSpace(outStr)))
		return
	}

	// Success — nginx -t exits 0 and prints "syntax is ok".
}

// findNginxConf searches for nginx.conf relative to the provided directory.
// nginxConfDir is expected to be either nginx/ or nginx/sites/; we check
// common locations and return the first one that exists.
func findNginxConf(nginxConfDir string) string {
	candidates := []string{
		// If caller passed nginx/sites/, the conf is one level up.
		filepath.Join(nginxConfDir, "..", "nginx.conf"),
		// If caller passed nginx/ directly.
		filepath.Join(nginxConfDir, "nginx.conf"),
	}

	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, err := os.Stat(clean); err == nil {
			abs, err := filepath.Abs(clean)
			if err == nil {
				return abs
			}
			return clean
		}
	}

	return ""
}
