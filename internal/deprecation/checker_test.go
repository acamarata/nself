package deprecation_test

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/deprecation"
)

// calendarRegistry is a helper that builds a PluginRegistry from inline YAML.
func calendarRegistry(t *testing.T, yaml string) *deprecation.PluginRegistry {
	t.Helper()
	path := writeRegistry(t, yaml)
	reg, err := deprecation.LoadPluginRegistry(path)
	if err != nil {
		t.Fatalf("LoadPluginRegistry: %v", err)
	}
	return reg
}

// ---------------------------------------------------------------------------
// versionGTE-derived tests (via CheckPluginCompat)
// ---------------------------------------------------------------------------

func TestCheckPluginCompat_NoWarningsBelowDeprecatedIn(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints:
      - path: /v1/complete
        deprecated_in: "1.2.0"
        removed_in: "2.0.0"
        replacement: /v1/chat/completions
        reason: "Align with OpenAI-compatible path"
`
	reg := calendarRegistry(t, yaml)

	// Plugin at v1.1.0 — below deprecated_in 1.2.0 — no warnings expected.
	installed := []deprecation.InstalledPlugin{{Name: "ai", Version: "1.1.0"}}
	warnings := deprecation.CheckPluginCompat(installed, reg)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for v1.1.0 < deprecated_in 1.2.0, got %d: %v", len(warnings), warnings)
	}
}

func TestCheckPluginCompat_WarningAtDeprecatedIn(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints:
      - path: /v1/complete
        deprecated_in: "1.2.0"
        removed_in: "2.0.0"
        replacement: /v1/chat/completions
        reason: "Align with OpenAI-compatible path"
`
	reg := calendarRegistry(t, yaml)

	// Plugin exactly at deprecated_in — should warn.
	installed := []deprecation.InstalledPlugin{{Name: "ai", Version: "1.2.0"}}
	warnings := deprecation.CheckPluginCompat(installed, reg)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for v1.2.0 == deprecated_in 1.2.0, got %d", len(warnings))
	}
	w := warnings[0]
	if w.Plugin != "ai" {
		t.Errorf("expected plugin 'ai', got %q", w.Plugin)
	}
	if w.Path != "/v1/complete" {
		t.Errorf("expected path '/v1/complete', got %q", w.Path)
	}
	if w.RemovedIn != "2.0.0" {
		t.Errorf("expected removed_in '2.0.0', got %q", w.RemovedIn)
	}
}

func TestCheckPluginCompat_WarningAboveDeprecatedIn(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints:
      - path: /v1/complete
        deprecated_in: "1.2.0"
        removed_in: "2.0.0"
        replacement: /v1/chat/completions
        reason: ""
`
	reg := calendarRegistry(t, yaml)

	// Plugin above deprecated_in — should warn.
	installed := []deprecation.InstalledPlugin{{Name: "ai", Version: "1.3.0"}}
	warnings := deprecation.CheckPluginCompat(installed, reg)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for v1.3.0 > deprecated_in 1.2.0, got %d", len(warnings))
	}
}

func TestCheckPluginCompat_UnknownPlugin(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints: []
`
	reg := calendarRegistry(t, yaml)

	// A plugin not in the registry should produce no warnings.
	installed := []deprecation.InstalledPlugin{{Name: "mux", Version: "1.1.0"}}
	warnings := deprecation.CheckPluginCompat(installed, reg)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for unknown plugin, got %d", len(warnings))
	}
}

func TestCheckPluginCompat_EmptyInstalled(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints:
      - path: /v1/complete
        deprecated_in: "1.2.0"
        removed_in: "2.0.0"
        replacement: /v1/chat/completions
        reason: ""
`
	reg := calendarRegistry(t, yaml)
	warnings := deprecation.CheckPluginCompat(nil, reg)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for nil installed list, got %d", len(warnings))
	}
}

func TestCheckPluginCompat_MultipleEndpoints(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints:
      - path: /v1/complete
        deprecated_in: "1.2.0"
        removed_in: "2.0.0"
        replacement: /v1/chat/completions
        reason: ""
      - path: /v1/embed
        deprecated_in: "1.3.0"
        removed_in: "2.0.0"
        replacement: /v1/embeddings
        reason: ""
`
	reg := calendarRegistry(t, yaml)

	// Plugin at 1.3.0 — both endpoints are past their deprecated_in.
	installed := []deprecation.InstalledPlugin{{Name: "ai", Version: "1.3.0"}}
	warnings := deprecation.CheckPluginCompat(installed, reg)
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings for v1.3.0 with 2 deprecated endpoints, got %d", len(warnings))
	}
}

// ---------------------------------------------------------------------------
// Warning.String() format
// ---------------------------------------------------------------------------

func TestWarningString_ContainsAllParts(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints:
      - path: /v1/complete
        deprecated_in: "1.2.0"
        removed_in: "2.0.0"
        replacement: /v1/chat/completions
        reason: ""
`
	reg := calendarRegistry(t, yaml)
	installed := []deprecation.InstalledPlugin{{Name: "ai", Version: "1.2.0"}}
	warnings := deprecation.CheckPluginCompat(installed, reg)
	if len(warnings) == 0 {
		t.Fatal("expected at least 1 warning")
	}
	s := warnings[0].String()
	for _, substr := range []string{"ai", "1.2.0", "/v1/complete", "2.0.0", "/v1/chat/completions"} {
		if !strings.Contains(s, substr) {
			t.Errorf("warning string missing %q: %s", substr, s)
		}
	}
}

// ---------------------------------------------------------------------------
// SunsetDate ordering
// ---------------------------------------------------------------------------

func TestSunsetDate_SortedByRemovedIn(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints:
      - path: /v1/z
        deprecated_in: "1.2.0"
        removed_in: "3.0.0"
        replacement: /v3/z
        reason: ""
      - path: /v1/a
        deprecated_in: "1.2.0"
        removed_in: "1.5.0"
        replacement: /v1/a2
        reason: ""
`
	reg := calendarRegistry(t, yaml)
	entries := reg.SunsetDate("ai")
	if len(entries) != 2 {
		t.Fatalf("expected 2 sunset entries, got %d", len(entries))
	}
	if entries[0].RemovedIn != "1.5.0" {
		t.Errorf("expected first entry removed_in 1.5.0, got %q", entries[0].RemovedIn)
	}
	if entries[1].RemovedIn != "3.0.0" {
		t.Errorf("expected second entry removed_in 3.0.0, got %q", entries[1].RemovedIn)
	}
}

func TestSunsetDate_UnknownPlugin(t *testing.T) {
	yaml := `plugins:
  - name: ai
    api_version: "1.3.0"
    deprecated_endpoints: []
`
	reg := calendarRegistry(t, yaml)
	entries := reg.SunsetDate("unknown-plugin")
	if entries != nil {
		t.Errorf("expected nil for unknown plugin, got %v", entries)
	}
}

// ---------------------------------------------------------------------------
// HTTPSunsetHeader
// ---------------------------------------------------------------------------

func TestHTTPSunsetHeader_KnownVersion(t *testing.T) {
	h := deprecation.HTTPSunsetHeader("2.0.0")
	if !strings.Contains(h, "2027") {
		t.Errorf("expected 2027 in sunset header for 2.0.0, got %q", h)
	}
}

func TestHTTPSunsetHeader_EmptyVersion(t *testing.T) {
	h := deprecation.HTTPSunsetHeader("")
	if h != "" {
		t.Errorf("expected empty header for empty version, got %q", h)
	}
}

func TestHTTPSunsetHeader_UnknownVersion(t *testing.T) {
	// Fallback should still return a non-empty, RFC-1123 formatted date.
	h := deprecation.HTTPSunsetHeader("9.9.9")
	if h == "" {
		t.Error("expected non-empty fallback sunset header")
	}
	if !strings.Contains(h, "GMT") {
		t.Errorf("expected GMT in fallback sunset header, got %q", h)
	}
}
