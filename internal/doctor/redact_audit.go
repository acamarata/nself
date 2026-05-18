// Package doctor — redact_audit.go: doctor check OBS-REDACT-01.
//
// Verifies that a synthetic telemetry-shaped payload, after passing through
// observability.Redact, contains no patterns matching the canonical PII
// regexes (email, IPv4 [non-loopback], IPv6 [non-loopback], JWT). Catches
// regressions in redact.go where a pattern is silently removed or weakened.
//
// Severity: WARN. A redaction regression does not break the CLI but does
// degrade the privacy guarantees documented in
// .claude/docs/operations/telemetry-privacy.md.
package doctor

import (
	"context"
	"regexp"

	"github.com/nself-org/cli/internal/observability"
)

// obsRedactName is the public check ID.
const obsRedactName = "Telemetry redaction coverage (OBS-REDACT-01)"

// obsRedactProbes is the synthetic telemetry payload used to exercise the
// redactor. Each entry is a (label, input, mustNotMatchAfterRedact) triple.
var obsRedactProbes = []struct {
	label   string
	input   string
	pattern *regexp.Regexp
}{
	{"email", "user alice@example.com signed in",
		regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)},
	{"ipv4", "connect from 198.51.100.42 port 443",
		regexp.MustCompile(`\b198\.51\.100\.42\b`)},
	{"ipv6", "peer 2001:db8::8a2e:370:7334 joined",
		regexp.MustCompile(`\b2001:db8:`)},
	{"jwt", "auth eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.abcdefghij1234567890",
		regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\b`)},
	{"ssn", "ssn 123-45-6789 on record",
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)},
	// Synthetic test fixture — not a real credential. Built from parts to avoid
	// GitHub Push Protection false-positive on a fake key used as a redaction probe.
	// gitleaks:allow
	{"stripe_secret", "stripe key " + "sk_live_" + "AbCdEfGhIjKlMnOp1234567890",
		regexp.MustCompile(`\bsk_live_[A-Za-z0-9]{10,}`)},
}

// CheckRedactionCoverage runs OBS-REDACT-01.
func CheckRedactionCoverage(_ context.Context) CheckResult {
	const section = "observability"

	var leaks []string
	for _, p := range obsRedactProbes {
		out := observability.Redact(p.input)
		if p.pattern.MatchString(out) {
			leaks = append(leaks, p.label)
		}
	}

	if len(leaks) > 0 {
		msg := "redaction leak detected for: "
		for i, l := range leaks {
			if i > 0 {
				msg += ", "
			}
			msg += l
		}
		return CheckResult{
			Section: section,
			Name:    obsRedactName,
			Status:  "fail",
			Message: msg + " — telemetry payload may expose PII",
			FixCmd:  "review cli/internal/observability/redact.go and re-run go test ./internal/observability/...",
		}
	}

	return CheckResult{
		Section: section,
		Name:    obsRedactName,
		Status:  "pass",
		Message: "all PII probe patterns redacted",
	}
}
