package flags

import (
	"encoding/json"
	"testing"
	"time"
)

// TestFlagSunsetWarn — flag past sunset_at but before removal_date emits warn.
func TestFlagSunsetWarn(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	sunset := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) // past

	f := Flag{
		Key:      "experimental.alpha",
		Enabled:  true,
		SunsetAt: &sunset,
	}

	warns := CheckFlagLifecycle([]Flag{f}, now)
	if len(warns) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warns))
	}
	if warns[0].Level != "warn" {
		t.Errorf("expected level=warn, got %q", warns[0].Level)
	}
	if warns[0].FlagKey != "experimental.alpha" {
		t.Errorf("unexpected key: %q", warns[0].FlagKey)
	}
}

// TestFlagRemovalError — flag past removal_date emits error + caller can force-disable.
func TestFlagRemovalError(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	sunset := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	removal := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) // past

	f := Flag{
		Key:         "deprecated.feature",
		Enabled:     true,
		SunsetAt:    &sunset,
		RemovalDate: &removal,
	}

	warns := CheckFlagLifecycle([]Flag{f}, now)
	if len(warns) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warns))
	}
	if warns[0].Level != "error" {
		t.Errorf("expected level=error, got %q", warns[0].Level)
	}

	flags := []Flag{f}
	disabled := ApplyLifecycleDisable(flags, now)
	if disabled != 1 {
		t.Errorf("expected 1 disabled, got %d", disabled)
	}
	if flags[0].Enabled {
		t.Error("expected flag to be force-disabled")
	}
}

// TestFlagNoDates — no dates set → no warnings.
func TestFlagNoDates(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f := Flag{Key: "stable.feature", Enabled: true}

	warns := CheckFlagLifecycle([]Flag{f}, now)
	if len(warns) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warns))
	}
}

// TestFlagBeforeSunset — flag with sunset_at in future → no warning.
func TestFlagBeforeSunset(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	sunset := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC) // future

	f := Flag{Key: "soon", Enabled: true, SunsetAt: &sunset}

	warns := CheckFlagLifecycle([]Flag{f}, now)
	if len(warns) != 0 {
		t.Errorf("expected 0 warnings for future sunset, got %d", len(warns))
	}
}

// TestFlagJSONRoundTrip — sunset/removal serialize and deserialize cleanly.
func TestFlagJSONRoundTrip(t *testing.T) {
	sunset := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	removal := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	original := Flag{
		ID:          "abc",
		Key:         "lifecycle.test",
		Type:        "boolean",
		Enabled:     true,
		SunsetAt:    &sunset,
		RemovalDate: &removal,
	}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Flag
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.SunsetAt == nil || !decoded.SunsetAt.Equal(sunset) {
		t.Errorf("sunset_at round-trip failed: got %v want %v", decoded.SunsetAt, sunset)
	}
	if decoded.RemovalDate == nil || !decoded.RemovalDate.Equal(removal) {
		t.Errorf("removal_date round-trip failed: got %v want %v", decoded.RemovalDate, removal)
	}
}

// TestFlagOmitEmpty — flags without sunset/removal don't include those keys in JSON.
func TestFlagOmitEmpty(t *testing.T) {
	f := Flag{ID: "x", Key: "k", Type: "boolean", Enabled: true}
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(raw); contains(got, "sunset_at") || contains(got, "removal_date") {
		t.Errorf("expected sunset_at/removal_date omitted; got: %s", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
