// legacy_bios.go — P97 G0-T02
//
// nSelf was internally referred to as "BIOS" in pre-v1.0 (alpha) builds. A
// handful of long-running self-hosted deployments still ship .env files with
// the legacy BIOS_* prefixes. To avoid silently breaking those operators on
// upgrade, this file provides a one-way mapping from BIOS_* env vars to the
// canonical NSELF_* names. Any caller that loads env vars at startup can
// invoke ApplyLegacyBIOSFallback once and have legacy values transparently
// promoted to the new names.
//
// Policy:
//   - The new (NSELF_*) name always wins. We never overwrite a value that the
//     operator has already set under the canonical name.
//   - When a legacy var is promoted, we emit a one-shot deprecation warning
//     to stderr listing the mapping. Subsequent calls in the same process
//     are silent (so build/start/etc don't spam the warning N times).
//   - Sunset version: BIOS_* support will be removed in v1.2.0. The warning
//     advertises this so operators have a deadline.
package env

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
)

// LegacyBIOSSunsetVersion is the nSelf release after which BIOS_* env vars
// will no longer be honoured. Surfaced in the deprecation warning so
// operators know the cutoff.
const LegacyBIOSSunsetVersion = "v1.2.0"

// legacyBIOSMap maps deprecated BIOS_* env var names to the canonical
// NSELF_* names. Order is irrelevant at runtime; the alphabetical layout
// here is for readability in code review.
var legacyBIOSMap = map[string]string{
	"BIOS_ADMIN_EMAIL":     "NSELF_ADMIN_EMAIL",
	"BIOS_ADMIN_PASSWORD":  "NSELF_ADMIN_PASSWORD",
	"BIOS_BASE_DOMAIN":     "NSELF_BASE_DOMAIN",
	"BIOS_DOMAIN":          "NSELF_BASE_DOMAIN",
	"BIOS_ENV":             "NSELF_ENV",
	"BIOS_ENVIRONMENT":     "NSELF_ENV",
	"BIOS_LICENSE_KEY":     "NSELF_PLUGIN_LICENSE_KEY",
	"BIOS_PLUGIN_DIR":      "NSELF_PLUGIN_DIR",
	"BIOS_PLUGIN_REGISTRY": "NSELF_PLUGIN_REGISTRY",
	"BIOS_PROJECT":         "NSELF_PROJECT_NAME",
	"BIOS_PROJECT_NAME":    "NSELF_PROJECT_NAME",
	"BIOS_TIMEZONE":        "NSELF_TIMEZONE",
}

// legacyWarningOnce ensures the deprecation banner is printed at most once
// per process invocation, even if ApplyLegacyBIOSFallback is called
// repeatedly across multiple commands.
var legacyWarningOnce sync.Once

// ApplyLegacyBIOSFallback promotes any BIOS_* env var to its canonical
// NSELF_* equivalent when the canonical name is unset. Returns the slice of
// (legacy, canonical) pairs that were promoted in this call, primarily so
// tests can assert behaviour deterministically.
//
// The function uses os.Getenv / os.Setenv on the real process environment.
// The (legacy, canonical) pairs that were promoted are returned in a
// deterministic alphabetical order (by legacy name) so callers can rely on
// stable output for assertions and structured logs.
//
// On the first call that promotes at least one variable, a one-shot
// deprecation warning is written to stderr listing the mapping table and
// the sunset version. Subsequent calls in the same process do not re-print
// even if more variables are promoted.
func ApplyLegacyBIOSFallback() []LegacyBIOSPromotion {
	return applyLegacyBIOSFallback(os.Stderr, os.Getenv, os.Setenv)
}

// LegacyBIOSPromotion records a single (legacy, canonical) promotion.
type LegacyBIOSPromotion struct {
	Legacy    string
	Canonical string
	Value     string // value that was copied (redacted upstream when logged)
}

// applyLegacyBIOSFallback is the testable implementation. The real-process
// version (ApplyLegacyBIOSFallback) wires it to os.Getenv / os.Setenv /
// os.Stderr. Tests inject in-memory fakes.
func applyLegacyBIOSFallback(
	stderr io.Writer,
	getenv func(string) string,
	setenv func(string, string) error,
) []LegacyBIOSPromotion {
	var promoted []LegacyBIOSPromotion

	// Iterate the map in stable order so output is deterministic.
	keys := make([]string, 0, len(legacyBIOSMap))
	for k := range legacyBIOSMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, legacy := range keys {
		canonical := legacyBIOSMap[legacy]
		legacyVal := getenv(legacy)
		if legacyVal == "" {
			continue // operator did not set the legacy var
		}
		if getenv(canonical) != "" {
			continue // canonical already set — final mapped value wins, never overwrite
		}
		// Copy legacy value into canonical name. We deliberately do NOT
		// unset the legacy name: third-party scripts may read it, and
		// leaving it in place is a harmless no-op once canonical is set.
		if err := setenv(canonical, legacyVal); err != nil {
			// Never abort the whole CLI on a single env promotion failure —
			// that would brick the operator's command. Log to stderr and
			// continue.
			fmt.Fprintf(stderr, "warning: failed to promote %s -> %s: %v\n", legacy, canonical, err)
			continue
		}
		promoted = append(promoted, LegacyBIOSPromotion{
			Legacy:    legacy,
			Canonical: canonical,
			Value:     legacyVal,
		})
	}

	if len(promoted) > 0 {
		legacyWarningOnce.Do(func() {
			emitLegacyBIOSWarning(stderr, promoted)
		})
	}

	return promoted
}

// emitLegacyBIOSWarning writes a single human-readable deprecation banner
// to stderr describing the mapping table and the sunset version.
func emitLegacyBIOSWarning(w io.Writer, promoted []LegacyBIOSPromotion) {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("[DEPRECATED] Detected legacy BIOS_* environment variables.\n")
	b.WriteString("            BIOS_* support will be REMOVED in nSelf ")
	b.WriteString(LegacyBIOSSunsetVersion)
	b.WriteString(".\n")
	b.WriteString("            Migrate to the canonical NSELF_* names below:\n")
	b.WriteString("\n")
	for _, p := range promoted {
		fmt.Fprintf(&b, "              %s  ->  %s\n", p.Legacy, p.Canonical)
	}
	b.WriteString("\n")
	fmt.Fprint(w, b.String())
}

// resetLegacyWarningOnce is a test-only helper that resets the
// once-warning state so a single test can verify the warning banner
// appears on the first call within that test.
//
// Not exported; used via the same-package test file.
func resetLegacyWarningOnce() {
	legacyWarningOnce = sync.Once{}
}
