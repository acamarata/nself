package env

import (
	"bytes"
	"strings"
	"testing"
)

// fakeEnv is a closure-backed in-memory env store used to test the legacy
// BIOS fallback without touching os.Getenv / os.Setenv.
type fakeEnv map[string]string

func (e fakeEnv) get(k string) string   { return e[k] }
func (e fakeEnv) set(k, v string) error { e[k] = v; return nil }

func TestApplyLegacyBIOSFallback_PromotesWhenCanonicalUnset(t *testing.T) {
	resetLegacyWarningOnce()

	env := fakeEnv{
		"BIOS_PROJECT_NAME": "myproject",
		"BIOS_DOMAIN":       "example.com",
	}
	var stderr bytes.Buffer

	got := applyLegacyBIOSFallback(&stderr, env.get, env.set)

	if len(got) != 2 {
		t.Fatalf("expected 2 promotions, got %d: %+v", len(got), got)
	}
	if env["NSELF_PROJECT_NAME"] != "myproject" {
		t.Errorf("NSELF_PROJECT_NAME not promoted; got %q", env["NSELF_PROJECT_NAME"])
	}
	if env["NSELF_BASE_DOMAIN"] != "example.com" {
		t.Errorf("NSELF_BASE_DOMAIN not promoted; got %q", env["NSELF_BASE_DOMAIN"])
	}
	// Deprecation warning must appear and reference the sunset version.
	out := stderr.String()
	if !strings.Contains(out, "DEPRECATED") {
		t.Errorf("warning missing DEPRECATED tag; got:\n%s", out)
	}
	if !strings.Contains(out, LegacyBIOSSunsetVersion) {
		t.Errorf("warning missing sunset version %q; got:\n%s", LegacyBIOSSunsetVersion, out)
	}
}

func TestApplyLegacyBIOSFallback_CanonicalWins(t *testing.T) {
	resetLegacyWarningOnce()

	env := fakeEnv{
		"BIOS_PROJECT_NAME":  "legacyname",
		"NSELF_PROJECT_NAME": "newname",
	}
	var stderr bytes.Buffer

	got := applyLegacyBIOSFallback(&stderr, env.get, env.set)

	if len(got) != 0 {
		t.Errorf("expected 0 promotions when canonical is set, got %d", len(got))
	}
	if env["NSELF_PROJECT_NAME"] != "newname" {
		t.Errorf("canonical clobbered: got %q want %q", env["NSELF_PROJECT_NAME"], "newname")
	}
	if stderr.Len() != 0 {
		t.Errorf("warning should not fire when nothing was promoted; got:\n%s", stderr.String())
	}
}

func TestApplyLegacyBIOSFallback_NoLegacyNoOp(t *testing.T) {
	resetLegacyWarningOnce()

	env := fakeEnv{
		"NSELF_PROJECT_NAME": "myproject",
	}
	var stderr bytes.Buffer

	got := applyLegacyBIOSFallback(&stderr, env.get, env.set)

	if len(got) != 0 {
		t.Errorf("expected 0 promotions, got %d", len(got))
	}
	if stderr.Len() != 0 {
		t.Errorf("warning should not fire when nothing was promoted; got:\n%s", stderr.String())
	}
}

func TestApplyLegacyBIOSFallback_WarningOncePerProcess(t *testing.T) {
	resetLegacyWarningOnce()

	env := fakeEnv{"BIOS_PROJECT_NAME": "myproject"}
	var stderr bytes.Buffer

	_ = applyLegacyBIOSFallback(&stderr, env.get, env.set)
	first := stderr.Len()

	// Second call should NOT print again, even with another legacy promotion.
	env2 := fakeEnv{"BIOS_DOMAIN": "example.com"}
	_ = applyLegacyBIOSFallback(&stderr, env2.get, env2.set)
	second := stderr.Len()

	if first == 0 {
		t.Fatalf("first call should have printed warning")
	}
	if second != first {
		t.Errorf("second call should not re-print warning; first=%d second=%d", first, second)
	}
}

func TestApplyLegacyBIOSFallback_AllMappings(t *testing.T) {
	resetLegacyWarningOnce()

	// Verify every entry in the map is reachable: set every BIOS_* var,
	// confirm every NSELF_* gets populated.
	env := fakeEnv{}
	for legacy := range legacyBIOSMap {
		env[legacy] = "v_" + legacy
	}
	var stderr bytes.Buffer

	_ = applyLegacyBIOSFallback(&stderr, env.get, env.set)

	for legacy, canonical := range legacyBIOSMap {
		want := "v_" + legacy
		// Multiple legacy keys may map to the same canonical (e.g., BIOS_DOMAIN
		// and BIOS_BASE_DOMAIN both -> NSELF_BASE_DOMAIN). The first one wins
		// in alphabetical iteration, so the canonical value should be one of
		// the expected legacy values.
		got := env[canonical]
		if got == "" {
			t.Errorf("canonical %q empty after fallback (legacy %q)", canonical, legacy)
			continue
		}
		// Confirm got matches *some* legacy value mapping to this canonical.
		matched := false
		for l, c := range legacyBIOSMap {
			if c == canonical && got == "v_"+l {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("canonical %q has unexpected value %q (want one of the BIOS_* values, e.g. %q)", canonical, got, want)
		}
	}
}
