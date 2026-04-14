// Package metrics — license.go emits Prometheus metrics for license
// grace period monitoring. Metrics are written as textfile collector
// .prom files for node_exporter to scrape.
//
// Metrics:
//   - nself_license_grace_days_remaining (gauge)
//   - nself_license_offline_seconds (gauge)
package metrics

import (
	"fmt"
	"strings"
)

// LicenseRecord captures the current license grace state for metric emission.
type LicenseRecord struct {
	Host               string  // hostname label (e.g., "nclaw-prod")
	Env                string  // prod, staging, dev
	GraceDaysRemaining float64 // days until hard stop; negative if past
	OfflineSeconds     float64 // seconds since last successful validation
	GraceState         string  // valid, grace_soft, grace_hard, expired
	Tier               string  // license tier
	TextfileDir        string  // overrides DefaultTextfileDir when non-empty
}

// EmitLicense writes license grace metrics to nself_license.prom.
func EmitLicense(r LicenseRecord) error {
	dir := r.TextfileDir
	if dir == "" {
		dir = DefaultTextfileDir
	}

	labels := labelString(map[string]string{
		"env":  firstNonEmpty(r.Env, "prod"),
		"host": firstNonEmpty(r.Host, "unknown"),
	})

	stateLabels := labelString(map[string]string{
		"env":   firstNonEmpty(r.Env, "prod"),
		"host":  firstNonEmpty(r.Host, "unknown"),
		"state": firstNonEmpty(r.GraceState, "unknown"),
		"tier":  firstNonEmpty(r.Tier, "unknown"),
	})

	var b strings.Builder

	fmt.Fprintln(&b, "# HELP nself_license_grace_days_remaining Days remaining in license grace period before hard stop")
	fmt.Fprintln(&b, "# TYPE nself_license_grace_days_remaining gauge")
	fmt.Fprintf(&b, "nself_license_grace_days_remaining%s %.1f\n", labels, r.GraceDaysRemaining)

	fmt.Fprintln(&b, "# HELP nself_license_offline_seconds Seconds since last successful license validation")
	fmt.Fprintln(&b, "# TYPE nself_license_offline_seconds gauge")
	fmt.Fprintf(&b, "nself_license_offline_seconds%s %.0f\n", labels, r.OfflineSeconds)

	fmt.Fprintln(&b, "# HELP nself_license_grace_state Current license grace state (1 = active for this state)")
	fmt.Fprintln(&b, "# TYPE nself_license_grace_state gauge")
	fmt.Fprintf(&b, "nself_license_grace_state%s 1\n", stateLabels)

	return writeAtomic(dir+"/nself_license.prom", b.String())
}
