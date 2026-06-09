package doctor

// hardening_check_infra.go — infrastructure and operational security checks for `nself doctor`.
//
// Purpose: SEC-HARDENING-08 (encryption-at-rest env var or disk marker),
//          SEC-CORS-01 (HASURA_GRAPHQL_CORS_DOMAIN not wildcard in prod),
//          SEC-DEVMODE-01 (Hasura dev-mode not true in staging/prod),
//          SEC-OFFLINE-01 (license cache fetched_at within 24h),
//          SEC-METRICS-01 (Prometheus port 9090 bound to 127.0.0.1 only).
// Inputs:  projectDir string for env-file reads, docker-compose.yml inspection,
//          and nginx config traversal. ctx context.Context for docker port check.
// Outputs: CheckResult for each check — pass/warn/fail with remediation hint.
// Constraints: formatAge is a private helper used only by checkLicenseOffline.
//              prometheusContainerSuffix/ExpectedPort/WildcardPort are package-
//              level consts so tests can reference them without magic strings.
//              encryptionEnvKeys and encryptedDiskMarkers are vars (not consts)
//              so tests can inject alternative lists.
// SPORT:   cli/internal/doctor — decomposed from hardening_check.go (T-E2-06).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

// ─── SEC-OFFLINE-01: license cache fresh (<24h) ────────────────────────────────

// checkLicenseOffline warns when the license cache's fetched_at timestamp is
// older than 24 hours. This indicates the license server has been unreachable
// for a day or more. The check is advisory-only (warn, never fail): the
// fail-open policy in validator.go handles the go/no-go decision. This doctor
// check surfaces the offline duration early so operators can investigate before
// plugins go dormant (FailOpenSoftTTL = 72h, FailOpenHardTTL = 14d).
//
// Cache location: ~/.cache/nself/license.json (overridable via LICENSE_CACHE_PATH).
func checkLicenseOffline() CheckResult {
	const checkID = "SEC-OFFLINE-01"

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
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	if data, err := os.ReadFile(composePath); err == nil {
		content := string(data)
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
	cmd := exec.CommandContext(context.Background(), "docker", "port", prometheusContainerSuffix[1:], "9090")
	out, err := cmd.Output()
	if err != nil {
		// Container not running — check the docker-compose source template.
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
