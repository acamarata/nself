package bundle

import (
	"testing"
)

func TestResolveVersionConflictsFromPins_MaxWins(t *testing.T) {
	bundlePins := map[string]VersionPins{
		"nclaw":  {"ai": "1.0.0", "mux": "2.0.0"},
		"clawde": {"ai": "1.2.0", "mux": "2.0.0"},
	}
	resolved, conflicts, err := ResolveVersionConflictsFromPins(bundlePins, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved["ai"] != "1.2.0" {
		t.Errorf("ai resolved to %q; want 1.2.0", resolved["ai"])
	}
	if resolved["mux"] != "2.0.0" {
		t.Errorf("mux resolved to %q; want 2.0.0", resolved["mux"])
	}
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict (ai); got %d: %v", len(conflicts), conflicts)
	}
	if len(conflicts) > 0 && conflicts[0].PluginName != "ai" {
		t.Errorf("conflict plugin = %q; want ai", conflicts[0].PluginName)
	}
	if len(conflicts) > 0 && conflicts[0].Resolved != "1.2.0" {
		t.Errorf("conflict Resolved = %q; want 1.2.0", conflicts[0].Resolved)
	}
}

func TestResolveVersionConflictsFromPins_NoConflict(t *testing.T) {
	bundlePins := map[string]VersionPins{
		"nclaw":  {"ai": "1.2.0"},
		"clawde": {"ai": "1.2.0"},
	}
	_, conflicts, err := ResolveVersionConflictsFromPins(bundlePins, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts; got %d", len(conflicts))
	}
}

func TestResolveVersionConflictsFromPins_DowngradeSkip(t *testing.T) {
	// Plugin "ai" installed at 2.0.0; both bundles want 1.x — should be skipped.
	bundlePins := map[string]VersionPins{
		"nclaw":  {"ai": "1.0.0"},
		"clawde": {"ai": "1.2.0"},
	}
	installed := map[string]string{"ai": "2.0.0"}
	resolved, conflicts, err := ResolveVersionConflictsFromPins(bundlePins, false, installed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "ai" should be absent from resolved (downgrade skip).
	if _, ok := resolved["ai"]; ok {
		t.Errorf("ai should be skipped (downgrade); got resolved[ai]=%q", resolved["ai"])
	}
	// Should have a conflict with Resolved == "" to signal skip.
	found := false
	for _, c := range conflicts {
		if c.PluginName == "ai" && c.Resolved == "" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected downgrade-skip conflict for ai; got conflicts=%v", conflicts)
	}
}

func TestResolveVersionConflictsFromPins_StrictWarnings(t *testing.T) {
	// strict=true: conflicts are still resolved (MAX wins), but bundle names are surfaced.
	bundlePins := map[string]VersionPins{
		"nclaw":  {"ai": "1.0.0"},
		"clawde": {"ai": "1.5.0"},
	}
	resolved, conflicts, err := ResolveVersionConflictsFromPins(bundlePins, true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved["ai"] != "1.5.0" {
		t.Errorf("ai resolved to %q; want 1.5.0", resolved["ai"])
	}
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict; got %d", len(conflicts))
	}
	// In strict mode the install is NOT blocked — conflict is informational.
}

func TestResolveVersionConflictsFromPins_EmptyBundles(t *testing.T) {
	resolved, conflicts, err := ResolveVersionConflictsFromPins(nil, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected empty resolved map; got %d entries", len(resolved))
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts; got %d", len(conflicts))
	}
}

func TestResolveVersionConflictsFromPins_SingleBundle(t *testing.T) {
	bundlePins := map[string]VersionPins{
		"nclaw": {"ai": "1.0.0", "mux": "2.1.0"},
	}
	resolved, conflicts, err := ResolveVersionConflictsFromPins(bundlePins, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved["ai"] != "1.0.0" {
		t.Errorf("ai resolved to %q; want 1.0.0", resolved["ai"])
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for single bundle; got %d", len(conflicts))
	}
}

func TestResolveVersionConflictsFromPins_ManyBundles(t *testing.T) {
	bundlePins := map[string]VersionPins{
		"nclaw":  {"ai": "1.0.0", "auth": "3.0.0"},
		"nchat":  {"ai": "1.3.0", "auth": "3.0.0"},
		"clawde": {"ai": "1.1.0", "auth": "3.1.0"},
	}
	resolved, _, err := ResolveVersionConflictsFromPins(bundlePins, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved["ai"] != "1.3.0" {
		t.Errorf("ai resolved to %q; want 1.3.0", resolved["ai"])
	}
	if resolved["auth"] != "3.1.0" {
		t.Errorf("auth resolved to %q; want 3.1.0", resolved["auth"])
	}
}

func TestHasDistinctVersions(t *testing.T) {
	cases := []struct {
		in   []string
		want bool
	}{
		{[]string{"1.0.0"}, false},
		{[]string{"1.0.0", "1.0.0"}, false},
		{[]string{"v1.0.0", "1.0.0"}, false}, // v-prefix normalised
		{[]string{"1.0.0", "1.0.1"}, true},
		{[]string{"1.0.0", "1.0.0", "2.0.0"}, true},
	}
	for _, tc := range cases {
		got := hasDistinctVersions(tc.in)
		if got != tc.want {
			t.Errorf("hasDistinctVersions(%v) = %v; want %v", tc.in, got, tc.want)
		}
	}
}
