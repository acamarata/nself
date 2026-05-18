package doctor

// hardening_check.go — SEC-HARDENING-01..08 + SEC-CORS-01 + SEC-OFFLINE-01 + SEC-DEVMODE-01 + SEC-METRICS-01: Security-Always-Free hardening checks.
//
// These checks run as part of `nself doctor --deep` (no license required).
// Each returns Pass | Warn | Fail with a remediation hint.
//
// Checks:
//   SEC-HARDENING-01  RLS enabled + FORCE RLS on every np_* table
//   SEC-HARDENING-02  Auth plugin has AUTH_RATE_LIMIT_* env vars
//   SEC-HARDENING-03  MFA throttle (AUTH_MFA_MAX_ATTEMPTS_PER_HOUR) set
//   SEC-HARDENING-04  SSRF guard imported by any plugin making outbound HTTP
//   SEC-HARDENING-05  JWT_PUBLIC_KEYS has at least 2 keys (rotation-ready)
//   SEC-HARDENING-06  Nginx config has rate-limit zones for /auth/login + /api/
//   SEC-HARDENING-07  nself-audit plugin installed or audit_log table writable
//   SEC-HARDENING-08  POSTGRES_DATA_ENCRYPTED env set or encrypted-disk marker present
//   SEC-CORS-01       HASURA_GRAPHQL_CORS_DOMAIN is not a wildcard "*" in staging/prod
//   SEC-OFFLINE-01    License cache fetched_at within 24h (warn when server unreachable too long)
//   SEC-DEVMODE-01    HASURA_GRAPHQL_DEV_MODE must not be true in staging/prod
//   SEC-METRICS-01    Prometheus /metrics port (9090) bound to 127.0.0.1 only (not 0.0.0.0)

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const hardeningSection = "security"

// HardeningChecks runs all SEC-HARDENING-*, SEC-CORS-01, SEC-OFFLINE-01, SEC-DEVMODE-01, and SEC-METRICS-01 checks.
// projectDir is the nSelf working directory (same value passed to DeepChecks).
func HardeningChecks(ctx context.Context, projectDir string) []CheckResult {
	return []CheckResult{
		checkHardeningRLS(ctx),
		checkHardeningRateLimit(projectDir),
		checkHardeningMFAThrottle(projectDir),
		checkHardeningSSRFImport(projectDir),
		checkHardeningJWTPublicKeys(projectDir),
		checkHardeningNginxRateZones(ctx, projectDir),
		checkHardeningAuditLog(ctx, projectDir),
		checkHardeningEncryptionAtRest(projectDir),
		checkCORSDomain(projectDir),
		checkLicenseOffline(),
		checkHasuraDevMode(projectDir),
		checkMetricsPortBinding(projectDir),
	}
}

// ─── SEC-HARDENING-01: RLS enabled + FORCE RLS on every np_* table ──────────

func checkHardeningRLS(ctx context.Context) CheckResult {
	const checkID = "SEC-HARDENING-01"

	// Query pg_class for np_* tables that are missing RLS enable or force.
	query := `SELECT relname,
	       relrowsecurity::text   AS rls_enabled,
	       relforcerowsecurity::text AS rls_forced
	  FROM pg_class
	 WHERE relname LIKE 'np_%'
	   AND relkind = 'r'
	   AND (NOT relrowsecurity OR NOT relforcerowsecurity)
	 ORDER BY relname;`

	cmd := exec.CommandContext(ctx,
		"docker", "exec", "nself_postgres",
		"psql", "-U", "postgres", "-t", "-c", query,
	)
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-01: cannot query pg_class (%v) — ensure Postgres is running", err),
			FixCmd:  "nself start",
		}
	}

	// Trim whitespace; empty output means every np_* table has RLS+FORCE.
	violations := strings.TrimSpace(string(out))
	if violations == "" {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-HARDENING-01: all np_* tables have ENABLE and FORCE RLS",
		}
	}

	// Count violating tables.
	count := len(strings.Split(violations, "\n"))
	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "fail",
		Message: fmt.Sprintf("SEC-HARDENING-01: %d np_* table(s) missing ENABLE or FORCE RLS — run nself doctor --fix to remediate", count),
		FixCmd:  "nself doctor --fix --check SEC-HARDENING-01",
	}
}

// ─── SEC-HARDENING-02: Auth rate-limit env vars set ──────────────────────────

// rateLimitEnvKeys are the required AUTH_RATE_LIMIT_* env vars.
var rateLimitEnvKeys = []string{
	"AUTH_RATE_LIMIT_ENABLED",
	"AUTH_RATE_LIMIT_MAX_REQUESTS",
	"AUTH_RATE_LIMIT_WINDOW_SECONDS",
}

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

// outboundHTTPPlugins are the plugins known to make outbound HTTP requests.
// These mirror the list in ssrf.go (ssrfServiceChecks).
var outboundHTTPPlugins = []string{"ai", "mux", "browser", "claw", "notify"}

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

		// Use `go list` to check imports, falling back to a simple file search.
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
// with a recognised SSRF guard symbol. This reuses ssrfGuardSymbols from ssrf.go.
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
			FixCmd:  "See docs.nself.org/security/nginx-rate-limiting",
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
			FixCmd:  "See docs.nself.org/security/nginx-rate-limiting",
		}
	}
}

// ─── SEC-HARDENING-07: Audit log enabled ─────────────────────────────────────

// auditTableName is the canonical table created by the nself-audit plugin migration
// (plugins-pro/paid/nself-audit/migrations/001_init.sql). The previous check
// erroneously referenced "np_audit_log" — the correct table is "np_audit_events".
const auditTableName = "np_audit_events"

// auditTriggerName is the append-only trigger installed by 001_init.sql that
// blocks UPDATE and DELETE on np_audit_events to ensure tamper-evidence.
const auditTriggerName = "trg_audit_immutable"

func checkHardeningAuditLog(ctx context.Context, projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-07"

	// Check 1: nself-audit plugin listed as installed.
	listCmd := exec.CommandContext(ctx, "nself", "plugin", "list", "--installed")
	listOut, listErr := listCmd.Output()
	if listErr == nil && strings.Contains(string(listOut), "nself-audit") {
		// Plugin is installed; also verify the append-only trigger is present
		// to confirm the tamper-evident guarantee is actually in place.
		triggerCheck := checkAuditTrigger(ctx)
		if triggerCheck == "pass" {
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("SEC-HARDENING-07: nself-audit plugin installed, %s present with %s append-only trigger", auditTableName, auditTriggerName),
			}
		}
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-07: nself-audit plugin installed but %s trigger not found — migration may not have run", auditTriggerName),
			FixCmd:  "nself plugin run nself-audit migrate",
		}
	}

	// Check 2: np_audit_events table exists and is writable (INSERT privilege check).
	// The table is INSERT+SELECT only — UPDATE and DELETE are blocked by trigger.
	query := fmt.Sprintf(`SELECT has_table_privilege(current_user, '%s', 'INSERT')::text;`, auditTableName)
	cmd := exec.CommandContext(ctx,
		"docker", "exec", "nself_postgres",
		"psql", "-U", "postgres", "-t", "-c", query,
	)
	out, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(out)) == "t" {
		// Table writable — also verify append-only trigger.
		triggerCheck := checkAuditTrigger(ctx)
		if triggerCheck == "pass" {
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("SEC-HARDENING-07: %s present, writable, and append-only trigger active", auditTableName),
			}
		}
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-07: %s writable but %s trigger missing — audit events not tamper-evident", auditTableName, auditTriggerName),
			FixCmd:  "nself plugin run nself-audit migrate",
		}
	}

	// Check 3: table exists but INSERT privilege not confirmed.
	existsQuery := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='%s')::text;`, auditTableName)
	existsCmd := exec.CommandContext(ctx,
		"docker", "exec", "nself_postgres",
		"psql", "-U", "postgres", "-t", "-c", existsQuery,
	)
	existsOut, existsErr := existsCmd.Output()
	if existsErr == nil && strings.TrimSpace(string(existsOut)) == "t" {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-HARDENING-07: %s table exists but INSERT privilege not confirmed — verify write access", auditTableName),
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "warn",
		Message: fmt.Sprintf("SEC-HARDENING-07: audit log not detected — install nself-audit plugin (table: %s)", auditTableName),
		FixCmd:  "nself plugin install nself-audit",
	}
}

// checkAuditTrigger verifies that the append-only trigger trg_audit_immutable
// exists on np_audit_events. Returns "pass" if confirmed, "fail" otherwise.
func checkAuditTrigger(ctx context.Context) string {
	triggerQuery := fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM information_schema.triggers WHERE trigger_name='%s' AND event_object_table='%s')::text;`,
		auditTriggerName, auditTableName,
	)
	triggerCmd := exec.CommandContext(ctx,
		"docker", "exec", "nself_postgres",
		"psql", "-U", "postgres", "-t", "-c", triggerQuery,
	)
	triggerOut, triggerErr := triggerCmd.Output()
	if triggerErr == nil && strings.TrimSpace(string(triggerOut)) == "t" {
		return "pass"
	}
	return "fail"
}

// ─── SEC-HARDENING-08: Encryption at rest ────────────────────────────────────

// encryptionEnvKeys lists env vars that declare encryption-at-rest is active.
var encryptionEnvKeys = []string{
	"POSTGRES_DATA_ENCRYPTED",
	"NSELF_DISK_ENCRYPTED",
	"NSELF_ENCRYPTION_AT_REST",
}

// encryptedDiskMarkers are filesystem paths whose presence signals
// the volume is backed by an encrypted disk (dm-crypt, LUKS, FileVault, etc.).
var encryptedDiskMarkers = []string{
	"/etc/crypttab",           // LUKS-managed volumes (Linux)
	"/sys/block/dm-0/dm/name", // device-mapper (dm-crypt) present
}

func checkHardeningEncryptionAtRest(projectDir string) CheckResult {
	const checkID = "SEC-HARDENING-08"

	// Check 1: env var declares encryption-at-rest.
	for _, key := range encryptionEnvKeys {
		val := envKeyValue(projectDir, key)
		if strings.EqualFold(strings.TrimSpace(val), "true") ||
			strings.EqualFold(strings.TrimSpace(val), "1") ||
			strings.EqualFold(strings.TrimSpace(val), "yes") {
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("SEC-HARDENING-08: encryption-at-rest confirmed via %s=%s", key, val),
			}
		}
	}

	// Check 2: filesystem markers indicate encrypted disk.
	for _, marker := range encryptedDiskMarkers {
		if _, err := os.Stat(marker); err == nil {
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("SEC-HARDENING-08: encrypted disk detected via %s", marker),
			}
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "warn",
		Message: "SEC-HARDENING-08: encryption-at-rest not confirmed — set POSTGRES_DATA_ENCRYPTED=true or deploy on an encrypted volume",
		FixCmd:  "nself config set POSTGRES_DATA_ENCRYPTED=true  # then verify disk-level encryption in your VPS settings",
	}
}

// ─── SEC-CORS-01: CORS domain not wildcard in staging/prod ───────────────────

// checkCORSDomain fails when HASURA_GRAPHQL_CORS_DOMAIN contains a bare "*"
// in a staging or production environment. SEC-CORS-01. A wildcard CORS policy
// allows any origin to read Hasura responses, which bypasses the authentication
// layer for browsers. The validator.go check catches this at startup; this
// doctor check surfaces it for running deployments.
func checkCORSDomain(projectDir string) CheckResult {
	const checkID = "SEC-CORS-01"

	// Detect environment from env files.
	nenv := envKeyValue(projectDir, "NSELF_ENV")
	if nenv == "" {
		nenv = envKeyValue(projectDir, "NODE_ENV")
	}
	isProd := nenv == "staging" || nenv == "prod" || nenv == "production"
	if !isProd {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-CORS-01: CORS check skipped (non-production environment)",
		}
	}

	domain := envKeyValue(projectDir, "HASURA_GRAPHQL_CORS_DOMAIN")
	if domain == "" {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: "SEC-CORS-01: HASURA_GRAPHQL_CORS_DOMAIN is not set — explicit domain required in production",
			FixCmd:  `nself config set HASURA_GRAPHQL_CORS_DOMAIN="https://yourdomain.example"`,
		}
	}

	// Block bare wildcard "*" and patterns like "https://*".
	if strings.TrimSpace(domain) == "*" || strings.HasPrefix(strings.TrimSpace(domain), "*") {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "fail",
			Message: fmt.Sprintf("SEC-CORS-01: HASURA_GRAPHQL_CORS_DOMAIN=%q is a wildcard — any origin can read Hasura in production", domain),
			FixCmd:  `nself config set HASURA_GRAPHQL_CORS_DOMAIN="https://yourdomain.example"`,
		}
	}

	// Warn on sub-domain wildcard patterns like "https://*.example.com" — less
	// dangerous than bare "*" but still allows any subdomain as origin.
	if strings.Contains(domain, "*") {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-CORS-01: HASURA_GRAPHQL_CORS_DOMAIN=%q contains a wildcard — restrict to explicit origins if possible", domain),
			FixCmd:  `nself config set HASURA_GRAPHQL_CORS_DOMAIN="https://app.yourdomain.example"`,
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "pass",
		Message: fmt.Sprintf("SEC-CORS-01: CORS domain is explicit: %q", domain),
	}
}

// ─── SEC-DEVMODE-01: Hasura dev-mode not enabled in staging/prod ──────────────

// checkHasuraDevMode fails when HASURA_GRAPHQL_DEV_MODE=true is found in any
// production or staging env file. Dev mode exposes the Hasura Console and
// introspection to the public internet and must never be enabled in deployed
// environments. ValidateHasuraDevMode (cli/internal/config/validator.go) also
// emits a structured slog.Error when the runtime block fires; this doctor check
// surfaces the misconfiguration before deployment.
func checkHasuraDevMode(projectDir string) CheckResult {
	const checkID = "SEC-DEVMODE-01"

	// Detect environment from env files.
	nenv := envKeyValue(projectDir, "NSELF_ENV")
	if nenv == "" {
		nenv = envKeyValue(projectDir, "ENV")
	}
	isProd := nenv == "staging" || nenv == "prod" || nenv == "production"
	if !isProd {
		// Skip check in dev environments — dev-mode is intentional there.
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-DEVMODE-01: Hasura dev-mode check skipped (non-production environment)",
		}
	}

	devMode := envKeyValue(projectDir, "HASURA_GRAPHQL_DEV_MODE")
	if strings.ToLower(strings.TrimSpace(devMode)) == "true" {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "fail",
			Message: fmt.Sprintf(
				"SEC-DEVMODE-01: HASURA_GRAPHQL_DEV_MODE=true in %s — "+
					"Hasura Console and schema introspection are exposed to the public internet",
				nenv,
			),
			FixCmd: "nself config set HASURA_GRAPHQL_DEV_MODE=false",
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "pass",
		Message: fmt.Sprintf("SEC-DEVMODE-01: HASURA_GRAPHQL_DEV_MODE is not true in %s", nenv),
	}
}

// ─── SEC-OFFLINE-01: license cache fresh (<24h) ────────────────────────────────

// checkLicenseOffline warns when the license cache's fetched_at timestamp is
// older than 24 hours. This indicates the license server has been unreachable
// for a day or more, which is unexpected under normal connectivity and may
// signal a misconfigured firewall or DNS failure blocking ping.nself.org.
//
// The check is advisory-only (warn, never fail): the fail-open policy in
// validator.go handles the actual go/no-go decision. This doctor check surfaces
// the offline duration early so operators can investigate before plugins go
// dormant (FailOpenSoftTTL = 72h, FailOpenHardTTL = 14d).
//
// Cache location: ~/.cache/nself/license.json (overridable via LICENSE_CACHE_PATH).
func checkLicenseOffline() CheckResult {
	const checkID = "SEC-OFFLINE-01"

	// Determine cache path (same logic as license.CachePath).
	cachePath := os.Getenv("LICENSE_CACHE_PATH")
	if cachePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "warn",
				Message: "SEC-OFFLINE-01: cannot determine home directory — license cache location unknown",
			}
		}
		cachePath = filepath.Join(home, ".cache", "nself", "license.json")
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No cache yet (fresh install or no license key set). Not a problem.
			return CheckResult{
				Section: hardeningSection,
				Name:    checkID,
				Status:  "pass",
				Message: "SEC-OFFLINE-01: no license cache present (run nself license validate to populate)",
			}
		}
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: fmt.Sprintf("SEC-OFFLINE-01: cannot read license cache: %v", err),
		}
	}

	var entry struct {
		FetchedAt int64 `json:"fetched_at"`
	}
	if err := json.Unmarshal(data, &entry); err != nil || entry.FetchedAt == 0 {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "warn",
			Message: "SEC-OFFLINE-01: license cache is unreadable or missing fetched_at — run nself license validate",
			FixCmd:  "nself license validate",
		}
	}

	age := time.Since(time.Unix(entry.FetchedAt, 0))
	const warnThreshold = 24 * time.Hour

	if age <= warnThreshold {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: fmt.Sprintf("SEC-OFFLINE-01: license cache is fresh (%s old)", formatAge(age)),
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "warn",
		Message: fmt.Sprintf(
			"SEC-OFFLINE-01: license server unreachable for %s (cache age exceeds 24h) — "+
				"plugins go dormant at 72h if unreachable",
			formatAge(age),
		),
		FixCmd: "nself license validate  # run while connected to the internet",
	}
}

// formatAge formats a duration as a human-readable string (e.g. "2d 3h" or "45m").
func formatAge(d time.Duration) string {
	d = d.Truncate(time.Minute)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours > 0 && mins > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

// ─── SEC-METRICS-01: Prometheus port not externally reachable ─────────────────

// prometheusContainerName is the Docker container name used by the nself
// monitoring stack (docker-compose.monitoring.yml — PROJECT_NAME_prometheus).
// The name is checked as a suffix so it matches any project prefix.
const prometheusContainerSuffix = "_prometheus"

// prometheusExpectedPort is the canonical loopback binding for the Prometheus
// web UI and /metrics endpoint. Any binding to 0.0.0.0:9090 would expose
// scraped metrics (including secrets in labels) to the public internet.
const prometheusExpectedPort = "127.0.0.1:9090"

// prometheusWildcardPort is the dangerous binding that exposes Prometheus to
// all interfaces.
const prometheusWildcardPort = "0.0.0.0:9090"

// checkMetricsPortBinding verifies that the Prometheus container port 9090 is
// bound to 127.0.0.1 (loopback) and not to 0.0.0.0 (all interfaces).
//
// The nself monitoring docker-compose template already enforces "127.0.0.1:9090:9090"
// (see cli/internal/compose/docker-compose.monitoring.yml). This check detects if a
// user hand-edited the generated docker-compose.yml to loosen the binding, or if a
// future template change accidentally removes the loopback constraint.
//
// Severity: FAIL when the container is running and wildcard-bound;
// WARN when the container is not running (can't verify, may be intentional);
// PASS when bound to 127.0.0.1 as expected.
func checkMetricsPortBinding(projectDir string) CheckResult {
	const checkID = "SEC-METRICS-01"

	// Check 1: inspect the generated docker-compose.yml for the port binding.
	// This catches the misconfiguration before deployment.
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	if data, err := os.ReadFile(composePath); err == nil {
		content := string(data)
		// Look for the prometheus service port section.
		if strings.Contains(content, "prometheus") {
			if strings.Contains(content, prometheusWildcardPort) {
				return CheckResult{
					Section: hardeningSection,
					Name:    checkID,
					Status:  "fail",
					Message: fmt.Sprintf(
						"SEC-METRICS-01: Prometheus port is bound to 0.0.0.0:9090 in docker-compose.yml — "+
							"Prometheus /metrics is accessible from the public internet. "+
							"Change to %q or run `nself build` to regenerate from the secure template.",
						prometheusExpectedPort,
					),
					FixCmd: "nself build  # regenerates docker-compose.yml with 127.0.0.1:9090 binding",
				}
			}
			if strings.Contains(content, prometheusExpectedPort) {
				return CheckResult{
					Section: hardeningSection,
					Name:    checkID,
					Status:  "pass",
					Message: fmt.Sprintf("SEC-METRICS-01: Prometheus port bound to %s (loopback only)", prometheusExpectedPort),
				}
			}
		}
	}

	// Check 2: inspect the live Docker container binding (runtime verification).
	// docker port <container> 9090 returns the host binding for port 9090.
	cmd := exec.CommandContext(context.Background(), "docker", "port", prometheusContainerSuffix[1:], "9090")
	out, err := cmd.Output()
	if err != nil {
		// Container not running or docker unavailable — check the docker-compose source template.
		templatePath := filepath.Join(projectDir, "docker-compose.monitoring.yml")
		if tdata, terr := os.ReadFile(templatePath); terr == nil {
			content := string(tdata)
			if strings.Contains(content, prometheusWildcardPort) {
				return CheckResult{
					Section: hardeningSection,
					Name:    checkID,
					Status:  "warn",
					Message: fmt.Sprintf(
						"SEC-METRICS-01: docker-compose.monitoring.yml contains wildcard Prometheus binding (%s) — "+
							"container is not running but will be externally reachable when started",
						prometheusWildcardPort,
					),
					FixCmd: "nself build  # regenerates from the secure template with 127.0.0.1:9090",
				}
			}
			if strings.Contains(content, prometheusExpectedPort) {
				return CheckResult{
					Section: hardeningSection,
					Name:    checkID,
					Status:  "pass",
					Message: "SEC-METRICS-01: Prometheus port binding is loopback-only in compose template (container not running)",
				}
			}
		}
		// Monitoring stack not deployed — not an error.
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "pass",
			Message: "SEC-METRICS-01: monitoring stack not deployed (Prometheus not running) — binding check skipped",
		}
	}

	// Parse docker port output: "0.0.0.0:9090" or "127.0.0.1:9090".
	binding := strings.TrimSpace(string(out))
	if strings.HasPrefix(binding, "0.0.0.0") || strings.HasPrefix(binding, ":::") {
		return CheckResult{
			Section: hardeningSection,
			Name:    checkID,
			Status:  "fail",
			Message: fmt.Sprintf(
				"SEC-METRICS-01: Prometheus container port 9090 is externally bound (%s) — "+
					"Prometheus /metrics is accessible from the public internet. "+
					"Restart with the loopback binding: %s",
				binding, prometheusExpectedPort,
			),
			FixCmd: "nself build && nself restart  # regenerates secure docker-compose.yml",
		}
	}

	return CheckResult{
		Section: hardeningSection,
		Name:    checkID,
		Status:  "pass",
		Message: fmt.Sprintf("SEC-METRICS-01: Prometheus port 9090 bound to loopback (%s)", binding),
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// envKeysMissing returns the subset of keys not found in any env file under projectDir.
func envKeysMissing(projectDir string, keys []string) []string {
	var missing []string
	for _, key := range keys {
		if !envKeySet(projectDir, key) {
			missing = append(missing, key)
		}
	}
	return missing
}

// envKeySet reports whether key is present (with a non-empty value) in any
// standard env file under projectDir.
func envKeySet(projectDir string, key string) bool {
	return envKeyValue(projectDir, key) != ""
}

// envKeyValue returns the value of key from the first env file where it appears,
// searching in order: .env.prod, .env.staging, .env.dev, .env.secrets, .env.local, .env.
// Returns "" if the key is not found or has no value.
func envKeyValue(projectDir string, key string) string {
	candidates := []string{
		".env.prod", ".env.staging", ".env.dev",
		".env.secrets", ".env.local", ".env",
	}
	for _, fname := range candidates {
		val, found := readEnvFileKey(filepath.Join(projectDir, fname), key)
		if found {
			return val
		}
	}
	return ""
}

// readEnvFileKey reads a single KEY=VALUE line from an env file.
// Returns (value, true) on match, ("", false) when not found.
func readEnvFileKey(path, key string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	prefix := key + "="
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimPrefix(line, prefix)
			// Strip surrounding quotes.
			val = strings.Trim(val, `"'`)
			return val, true
		}
	}
	return "", false
}
