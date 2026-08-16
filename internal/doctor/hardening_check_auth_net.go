package doctor

// hardening_check_auth_net.go — authentication and network security checks for `nself doctor`.
//
// Purpose: SEC-HARDENING-02 (rate-limit env vars set), SEC-HARDENING-03 (MFA
//          brute-force throttle), SEC-HARDENING-04 (SSRF guard imported by all
//          outbound-HTTP plugins), SEC-HARDENING-05 (JWT_PUBLIC_KEYS has ≥2 keys),
//          SEC-HARDENING-06 (nginx rate-limit zones for /auth/login and /api/).
// Inputs:  projectDir string for env-file reads and filesystem traversal.
//          ctx context.Context for docker exec fallback in SEC-HARDENING-06.
// Outputs: CheckResult for each check — pass/warn/fail with remediation hint.
// Constraints: pluginHasSSRFGuard reads ssrfGuardSymbols from ssrf.go in the
//              same package — no import needed. rateLimitEnvKeys and
//              outboundHTTPPlugins are package-level vars so tests can override
//              them without touching the function under test.
// SPORT:   cli/internal/doctor — decomposed from hardening_check.go (T-E2-06).

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// rateLimitEnvKeys are the required AUTH_RATE_LIMIT_* env vars.
var rateLimitEnvKeys = []string{
	"AUTH_RATE_LIMIT_ENABLED",
	"AUTH_RATE_LIMIT_MAX_REQUESTS",
	"AUTH_RATE_LIMIT_WINDOW_SECONDS",
}

// outboundHTTPPlugins are the plugins known to make outbound HTTP requests.
// These mirror the list in ssrf.go (ssrfServiceChecks).
var outboundHTTPPlugins = []string{"ai", "mux", "browser", "claw", "notify"}

// ─── SEC-HARDENING-02: Auth rate-limit env vars set ──────────────────────────

func checkHardeningRateLimit(projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-02"

	missing := envKeysMissing(projectDir, rateLimitEnvKeys)
	if len(missing) == 0 {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-HARDENING-02: AUTH_RATE_LIMIT_* env vars are set",
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "warn",
		Message: fmt.Sprintf("SEC-HARDENING-02: missing rate-limit env vars: %s — add to .env.dev or .env.prod", strings.Join(missing, ", ")),
		FixCmd:  "nself config set AUTH_RATE_LIMIT_ENABLED=true AUTH_RATE_LIMIT_MAX_REQUESTS=100 AUTH_RATE_LIMIT_WINDOW_SECONDS=60",
	}
}

// ─── SEC-HARDENING-03: MFA throttle set ──────────────────────────────────────

func checkHardeningMFAThrottle(projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-03"
	const key = "AUTH_MFA_MAX_ATTEMPTS_PER_HOUR"

	if envKeySet(projectDir, key) {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: fmt.Sprintf("SEC-HARDENING-03: %s is set", key),
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "warn",
		Message: fmt.Sprintf("SEC-HARDENING-03: %s not set — MFA brute-force throttle is off. Recommended: 5", key),
		FixCmd:  fmt.Sprintf("nself config set %s=5", key),
	}
}

// ─── SEC-HARDENING-04: SSRF guard imported by outbound-HTTP plugins ──────────

func checkHardeningSSRFImport(projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-04"

	// Resolve plugins-pro/paid directory relative to the nself project root.
	paidDir := filepath.Join(projectDir, "..", "plugins-pro", "paid")
	if _, err := os.Stat(paidDir); os.IsNotExist(err) {
		// plugins-pro may not be present on a dev machine; treat as warn not fail.
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: "SEC-HARDENING-04: plugins-pro/paid directory not found — skipped (run on a full checkout)",
		}
	}

	var missing []string
	for _, plugin := range outboundHTTPPlugins {
		pluginDir := filepath.Join(paidDir, plugin)
		if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
			continue // plugin not present in this checkout — skip
		}

		// Use a simple file search for recognised SSRF guard symbols.
		if !pluginHasSSRFGuard(pluginDir) {
			missing = append(missing, plugin)
		}
	}

	if len(missing) == 0 {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-HARDENING-04: SSRF guard present in all outbound-HTTP plugins",
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "fail",
		Message: fmt.Sprintf("SEC-HARDENING-04: SSRF guard missing in plugin(s): %s", strings.Join(missing, ", ")),
		FixCmd:  "See plugins-pro docs: add shared/ssrf import to each listed plugin",
	}
}

// pluginHasSSRFGuard returns true when the plugin directory contains a file
// with a recognised SSRF guard symbol. ssrfGuardSymbols is declared in ssrf.go
// in the same package — no import needed.
func pluginHasSSRFGuard(pluginDir string) bool {
	guardPaths := []string{
		filepath.Join(pluginDir, "shared", "ssrf.go"),
		filepath.Join(pluginDir, "internal", "safety", "url_guard.go"),
		filepath.Join(pluginDir, "internal", "safety", "urlcheck.go"),
		filepath.Join(pluginDir, "internal", "tools", "browser", "client.go"),
		filepath.Join(pluginDir, "internal", "push", "ssrf.go"),
		filepath.Join(pluginDir, "providers", "ssrf.go"),
	}

	for _, p := range guardPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)
		for _, sym := range ssrfGuardSymbols {
			if strings.Contains(content, sym) {
				return true
			}
		}
	}
	return false
}

// ─── SEC-HARDENING-05: JWT_PUBLIC_KEYS has ≥2 keys ──────────────────────────

func checkHardeningJWTPublicKeys(projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-05"
	const key = "JWT_PUBLIC_KEYS"

	val := envKeyValue(projectDir, key)
	if val == "" {
		// Single-key mode (JWT_SECRET only) is functional but not rotation-ready.
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-05: %s not set — JWT key rotation requires ≥2 public keys", key),
			FixCmd:  "nself self-heal --jwt",
		}
	}

	// Count comma-separated PEM blocks or JSON array elements.
	count := countJWTKeys(val)
	if count < 2 {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-05: %s has only %d key — add a second key to enable zero-downtime rotation", key, count),
			FixCmd:  "nself self-heal --jwt",
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "pass",
		Message: fmt.Sprintf("SEC-HARDENING-05: %s has %d keys (rotation-ready)", key, count),
	}
}

// countJWTKeys counts distinct JWT public-key entries in a value string.
// Accepts a comma-separated list of PEM blocks or a JSON array of JWK objects.
func countJWTKeys(val string) int {
	val = strings.TrimSpace(val)

	// JSON array: count "{" occurrences as a proxy for array elements.
	if strings.HasPrefix(val, "[") {
		return strings.Count(val, "{")
	}

	// PEM / base64 list: count "-----BEGIN" blocks.
	if strings.Contains(val, "BEGIN") {
		return strings.Count(val, "-----BEGIN")
	}

	// Comma-separated opaque list.
	parts := strings.Split(val, ",")
	count := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}
	return count
}

// ─── SEC-HARDENING-06: Nginx rate-limit zones for /auth/login + /api/ ────────

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
		cmd := exec.CommandContext(ctx, "docker", "exec", "nself_nginx",
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
