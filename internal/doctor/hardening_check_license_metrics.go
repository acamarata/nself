package doctor

// hardening_check_license_metrics.go — license-freshness and Prometheus
// port-binding security checks for `nself doctor`, split out of
// hardening_check_infra.go (CLI-R12) as a pure move.
//
// Purpose: SEC-OFFLINE-01 (license cache fetched_at within 24h),
//          SEC-METRICS-01 (Prometheus port 9090 bound to 127.0.0.1 only).
// Inputs:  projectDir string for docker-compose.yml inspection; no args for
//          the license check (reads LICENSE_CACHE_PATH / ~/.cache/nself).
// Outputs: CheckResult for each check — pass/warn/fail with remediation hint.
// Constraints: formatAge is a private helper used only by checkLicenseOffline.
//              Depends on prometheusContainerSuffix/ExpectedPort/WildcardPort
//              and hardeningSection, defined in hardening_check_infra.go.
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
