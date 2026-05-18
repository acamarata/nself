// Package doctor — tests for OBS-REDACT-01 redaction audit check.
package doctor

import (
	"context"
	"strings"
	"testing"
)

func TestCheckRedactionCoverage_Pass(t *testing.T) {
	got := CheckRedactionCoverage(context.Background())
	if got.Status != "pass" {
		t.Errorf("expected status=pass, got %q (msg=%q)", got.Status, got.Message)
	}
	if got.Section != "observability" {
		t.Errorf("expected section=observability, got %q", got.Section)
	}
	if !strings.Contains(got.Name, "OBS-REDACT-01") {
		t.Errorf("expected check name to mention OBS-REDACT-01, got %q", got.Name)
	}
}

func TestCheckRedactionCoverage_ProbeCount(t *testing.T) {
	// Guard: the probe table must cover at least the 6 documented surfaces
	// (email, ipv4, ipv6, jwt, ssn, stripe_secret). If a future refactor
	// drops one, this guard catches it.
	want := []string{"email", "ipv4", "ipv6", "jwt", "ssn", "stripe_secret"}
	have := map[string]bool{}
	for _, p := range obsRedactProbes {
		have[p.label] = true
	}
	for _, w := range want {
		if !have[w] {
			t.Errorf("probe table missing required label %q", w)
		}
	}
}
