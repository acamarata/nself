package admin

import "github.com/nself-org/cli/internal/alerts"

// AlertRules returns the two admin-specific Prometheus alert rules.
func AlertRules() []alerts.AlertRule {
	return []alerts.AlertRule{
		{
			Name:     "AdminAuthFailures",
			Severity: alerts.SeverityP2,
			For:      "0m",
			Expr:     `rate(nself_admin_auth_failure_total[10m]) > 0.05`,
			Summary:  "Admin auth failure rate elevated (>0.05/s over 10m)",
		},
		{
			Name:     "AdminPortExternallyReachable",
			Severity: alerts.SeverityP1,
			For:      "5m",
			Expr:     `nself_admin_external_reachable == 1`,
			Summary:  "Admin port 3021 is externally reachable (must be 127.0.0.1 only)",
		},
	}
}
