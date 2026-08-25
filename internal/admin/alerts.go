package admin

// AlertRule is a minimal Prometheus alert rule descriptor.
//
// Purpose: describe the two admin-specific alert rules without depending on
// internal/alerts, which was extracted to a standalone plugin under
// CLI-R11 (see cli/cmd/commands/alerts.go). internal/admin's usage was
// limited to two static rule literals plus the SeverityP1/SeverityP2
// constants — no shared logic, no shared state file — so this local type
// mirrors internal/alerts.AlertRule's fields exactly rather than forking any
// real behavior.
//
// Inputs: none — the two rules below are static.
//
// Outputs: []AlertRule for the two admin-specific rules.
//
// Constraints: keep field names identical to internal/alerts.AlertRule so a
// future caller that wants to merge this with the core rule set can convert
// between the two without surprises.
type AlertRule struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Expr     string `json:"expr"`
	For      string `json:"for"`
	Summary  string `json:"summary"`
}

// Severity levels, copied from internal/alerts so callers see identical
// values (P1 = page immediately, P2 = elevated but not urgent).
const (
	SeverityP1 = "P1"
	SeverityP2 = "P2"
)

// AlertRules returns the two admin-specific Prometheus alert rules.
func AlertRules() []AlertRule {
	return []AlertRule{
		{
			Name:     "AdminAuthFailures",
			Severity: SeverityP2,
			For:      "0m",
			Expr:     `rate(nself_admin_auth_failure_total[10m]) > 0.05`,
			Summary:  "Admin auth failure rate elevated (>0.05/s over 10m)",
		},
		{
			Name:     "AdminPortExternallyReachable",
			Severity: SeverityP1,
			For:      "5m",
			Expr:     `nself_admin_external_reachable == 1`,
			Summary:  "Admin port 3021 is externally reachable (must be 127.0.0.1 only)",
		},
	}
}
