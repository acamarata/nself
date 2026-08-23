package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Purpose: monitoring-stack reachability checks (Prometheus, Grafana, Loki)
// and the Loki retention_period verification (MON-LOKI-RETENTION-01), plus
// its small duration-parsing helpers.
// Inputs: a context; the retention helpers take config text or a duration string.
// Outputs: []CheckResult, or a single CheckResult from checkLokiRetention.
// Constraints: split out of system.go (CLI-R12) as a pure move; no behavior
// changed.

// MonitoringChecks verifies monitoring stack health.
func MonitoringChecks(ctx context.Context) []CheckResult {
	var results []CheckResult

	// Check Prometheus targets
	cmd := exec.CommandContext(ctx, "curl", "-sf", "http://127.0.0.1:9090/-/healthy")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "monitoring", Name: "Prometheus", Status: "warn", Message: "not reachable"})
	} else {
		results = append(results, CheckResult{Section: "monitoring", Name: "Prometheus", Status: "pass", Message: "healthy"})
	}

	// Check Grafana
	cmd = exec.CommandContext(ctx, "curl", "-sf", "http://127.0.0.1:3000/api/health")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "monitoring", Name: "Grafana", Status: "warn", Message: "not reachable"})
	} else {
		results = append(results, CheckResult{Section: "monitoring", Name: "Grafana", Status: "pass", Message: "healthy"})
	}

	// Check Loki reachability
	cmd = exec.CommandContext(ctx, "curl", "-sf", "http://127.0.0.1:3100/ready")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "monitoring", Name: "Loki", Status: "warn", Message: "not reachable"})
	} else {
		results = append(results, CheckResult{Section: "monitoring", Name: "Loki", Status: "pass", Message: "ingesting"})
	}

	// MON-LOKI-RETENTION-01: verify Loki retention_period is configured.
	// The standard nSelf Loki config sets retention_period: 720h (30 days).
	// Missing or short retention means logs may roll off before DR drills can
	// access historical data needed for incident investigation.
	results = append(results, checkLokiRetention(ctx))

	return results
}

// checkLokiRetention verifies that Loki is configured with a retention_period
// of at least 720h (30 days). Checks the running Loki config via the /config
// endpoint, then falls back to reading the config file from the Docker container.
func checkLokiRetention(ctx context.Context) CheckResult {
	const checkID = "MON-LOKI-RETENTION-01"
	const minRetentionHours = 720 // 30 days

	// Attempt 1: query Loki /config endpoint (YAML response).
	configCmd := exec.CommandContext(ctx, "curl", "-sf", "http://127.0.0.1:3100/config")
	configOut, configErr := configCmd.Output()
	if configErr == nil {
		retentionFound, retentionOK := parseLokiRetention(string(configOut), minRetentionHours)
		if retentionFound && retentionOK {
			return CheckResult{
				Section: "monitoring",
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("MON-LOKI-RETENTION-01: Loki retention_period >= %dh (30 days)", minRetentionHours),
			}
		}
		if retentionFound && !retentionOK {
			return CheckResult{
				Section: "monitoring",
				Name:    checkID,
				Status:  "warn",
				Message: fmt.Sprintf("MON-LOKI-RETENTION-01: Loki retention_period is set but less than %dh — increase to 720h for 30-day log history", minRetentionHours),
				FixCmd:  "nself doctor --fix-loki-retention",
			}
		}
	}

	// Attempt 2: read config file from nself_loki container.
	catCmd := exec.CommandContext(ctx,
		"docker", "exec", "nself_loki",
		"cat", "/etc/loki/loki.yaml",
	)
	catOut, catErr := catCmd.Output()
	if catErr == nil {
		retentionFound, retentionOK := parseLokiRetention(string(catOut), minRetentionHours)
		if retentionFound && retentionOK {
			return CheckResult{
				Section: "monitoring",
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("MON-LOKI-RETENTION-01: Loki config file has retention_period >= %dh", minRetentionHours),
			}
		}
		if retentionFound && !retentionOK {
			return CheckResult{
				Section: "monitoring",
				Name:    checkID,
				Status:  "warn",
				Message: fmt.Sprintf("MON-LOKI-RETENTION-01: Loki retention_period < %dh — add 'retention_period: 720h' under compactor in loki.yaml", minRetentionHours),
				FixCmd:  "nself doctor --fix-loki-retention",
			}
		}
		// Config readable but retention_period not set.
		return CheckResult{
			Section: "monitoring",
			Name:    checkID,
			Status:  "warn",
			Message: "MON-LOKI-RETENTION-01: retention_period not set in Loki config — logs may roll off without warning",
			FixCmd:  "nself doctor --fix-loki-retention",
		}
	}

	// Could not reach Loki or its container — skip retention check.
	return CheckResult{
		Section: "monitoring",
		Name:    checkID,
		Status:  "warn",
		Message: "MON-LOKI-RETENTION-01: Loki not reachable — cannot verify retention_period",
	}
}

// parseLokiRetention scans a Loki YAML config string for a retention_period
// field and returns (found bool, meetsMinimum bool).
// Accepted formats: "720h", "30d", "43200m".
func parseLokiRetention(config string, minHours int) (found bool, meetsMinimum bool) {
	for _, line := range strings.Split(config, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "retention_period:") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		val := strings.TrimSpace(parts[1])
		hours := parseRetentionToHours(val)
		if hours < 0 {
			return true, false // found but unparseable — treat as too short
		}
		return true, hours >= minHours
	}
	return false, false
}

// parseRetentionToHours converts a Prometheus duration string (720h, 30d, 43200m)
// to integer hours. Returns -1 on parse failure.
func parseRetentionToHours(val string) int {
	if len(val) < 2 {
		return -1
	}
	unit := val[len(val)-1]
	numStr := val[:len(val)-1]
	var num int
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil || num < 0 {
		return -1
	}
	switch unit {
	case 'h':
		return num
	case 'd':
		return num * 24
	case 'm':
		return num / 60
	case 'w':
		return num * 24 * 7
	}
	return -1
}
