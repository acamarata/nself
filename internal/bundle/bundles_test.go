package bundle

import (
	"testing"
)

func TestGet_KnownBundles(t *testing.T) {
	cases := []struct {
		slug        string
		wantOK      bool
		wantName    string
		installable bool
	}{
		{"nclaw", true, "ɳClaw", true},
		{"NCLAW", true, "ɳClaw", true},
		{"  nchat  ", true, "ɳChat", true},
		{"nself-plus", true, "ɳSelf+", false}, // meta, not installable
		{"ntask", true, "ɳTask", false},       // free, not installable as a unit
		{"sentry", true, "ɳSentry", true},     // alias -> nsentry
		{"SENTRY ", true, "ɳSentry", true},    // alias case-insensitive + trimmed
		{"bogus", false, "", false},
	}
	for _, tc := range cases {
		b, ok := Get(tc.slug)
		if ok != tc.wantOK {
			t.Errorf("Get(%q) ok = %v; want %v", tc.slug, ok, tc.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if b.Name != tc.wantName {
			t.Errorf("Get(%q).Name = %q; want %q", tc.slug, b.Name, tc.wantName)
		}
		if b.IsInstallable() != tc.installable {
			t.Errorf("Get(%q).IsInstallable() = %v; want %v", tc.slug, b.IsInstallable(), tc.installable)
		}
	}
}

func TestNames_SortedAndComplete(t *testing.T) {
	names := Names()
	if len(names) < 6 {
		t.Errorf("Names() returned %d entries; want at least 6", len(names))
	}
	// Sorted check
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("Names() not sorted at %d: %q > %q", i, names[i-1], names[i])
		}
	}
	// Must include all six paid bundles
	required := []string{"nclaw", "nchat", "nfamily", "ntv", "clawde", "nsentry"}
	have := make(map[string]bool, len(names))
	for _, n := range names {
		have[n] = true
	}
	for _, r := range required {
		if !have[r] {
			t.Errorf("Names() missing required bundle %q", r)
		}
	}
	// Aliases must never leak into Names()
	if have["sentry"] {
		t.Errorf("Names() must not include alias %q", "sentry")
	}
}

func TestAliases_NotInDisplayOrder(t *testing.T) {
	for _, slug := range DisplayOrder {
		if slug == "sentry" {
			t.Errorf("DisplayOrder must not include alias %q", "sentry")
		}
	}
}

func TestAll_DisplayOrder(t *testing.T) {
	all := All()
	if len(all) != len(DisplayOrder) {
		t.Fatalf("All() returned %d; want %d (DisplayOrder length)", len(all), len(DisplayOrder))
	}
	for i, b := range all {
		if b.Slug != DisplayOrder[i] {
			t.Errorf("All()[%d].Slug = %q; want %q", i, b.Slug, DisplayOrder[i])
		}
	}
}

func TestUnknownBundleError_ContainsHints(t *testing.T) {
	err := UnknownBundleError("bogus")
	if err == nil {
		t.Fatal("UnknownBundleError returned nil")
	}
	msg := err.Error()
	for _, want := range []string{"bogus", "nclaw", "nsentry"} {
		if !contains(msg, want) {
			t.Errorf("error message missing %q: %s", want, msg)
		}
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
