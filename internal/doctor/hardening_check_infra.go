package doctor

// hardening_check_infra.go — infrastructure and operational security checks for `nself doctor`.
//
// Purpose: SEC-HARDENING-08 (encryption-at-rest env var or disk marker),
//          SEC-CORS-01 (HASURA_GRAPHQL_CORS_DOMAIN not wildcard in prod),
//          SEC-DEVMODE-01 (Hasura dev-mode not true in staging/prod).
// Inputs:  projectDir string for env-file reads.
// Outputs: CheckResult for each check — pass/warn/fail with remediation hint.
// Constraints: prometheusContainerSuffix/ExpectedPort/WildcardPort are package-
//              level consts so tests can reference them without magic strings.
//              encryptionEnvKeys and encryptedDiskMarkers are vars (not consts)
//              so tests can inject alternative lists. SEC-OFFLINE-01 and
//              SEC-METRICS-01 (which use these same consts) live in
//              hardening_check_license_metrics.go — split out (CLI-R12) as a
//              pure move from this file.
// SPORT:   cli/internal/doctor — decomposed from hardening_check.go (T-E2-06).

import (
	"fmt"
	"os"
	"strings"
)

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

// prometheusContainerSuffix is the Docker container name suffix used by the nself
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

// ─── SEC-HARDENING-08: Encryption-at-rest env var or disk marker ─────────────

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
// in a staging or production environment. A wildcard CORS policy allows any
// origin to read Hasura responses, bypassing the authentication layer for
// browsers. The validator.go check catches this at startup; this doctor check
// surfaces it for running deployments.
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

	nenv := envKeyValue(projectDir, "NSELF_ENV")
	if nenv == "" {
		nenv = envKeyValue(projectDir, "ENV")
	}
	isProd := nenv == "staging" || nenv == "prod" || nenv == "production"
	if !isProd {
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
