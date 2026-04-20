package nginx

import (
	"strings"
	"testing"
)

// TestConflictErrorNamesBothPlugins is the S01.T02 regression test.
// When two plugins both claim the same route, HasDomainConflict must emit
// an error message that names both competing plugins, not just "conflict detected".
// Error format: "route conflict: /api claimed by <pluginA> and <pluginB>"
func TestConflictErrorNamesBothPlugins(t *testing.T) {
	routes := []NginxRoute{
		{ServerName: "api", Location: "/", PluginName: "plugin-a"},
		{ServerName: "api", Location: "/", PluginName: "plugin-b"},
	}

	conflict, details := HasDomainConflict(routes)
	if !conflict {
		t.Fatal("S01.T02 regression: HasDomainConflict() = false, expected true for duplicate route")
	}
	if len(details) == 0 {
		t.Fatal("S01.T02 regression: HasDomainConflict() returned no conflict details")
	}

	msg := details[0]

	// Both plugin names must appear in the conflict message.
	if !strings.Contains(msg, "plugin-a") {
		t.Errorf("S01.T02 regression: conflict message must name plugin-a; got: %q", msg)
	}
	if !strings.Contains(msg, "plugin-b") {
		t.Errorf("S01.T02 regression: conflict message must name plugin-b; got: %q", msg)
	}

	// Message must not be the old generic format without plugin names.
	if !strings.Contains(msg, "claimed by") {
		t.Errorf("S01.T02 regression: conflict message must contain 'claimed by'; got: %q", msg)
	}

	t.Logf("conflict message: %s", msg)
}

// TestConflictErrorNamesBothPlugins_NoPluginNames verifies that when PluginName
// is empty (legacy callers), HasDomainConflict still reports the conflict but
// falls back to the server-name-based format instead of panicking.
func TestConflictErrorNamesBothPlugins_NoPluginNames(t *testing.T) {
	routes := []NginxRoute{
		{ServerName: "api.example.com", Location: "/"},
		{ServerName: "api.example.com", Location: "/"},
	}

	conflict, details := HasDomainConflict(routes)
	if !conflict {
		t.Fatal("expected conflict for duplicate server_name without plugin names")
	}
	if len(details) == 0 {
		t.Fatal("expected at least one conflict detail string")
	}

	t.Logf("fallback conflict message: %s", details[0])
}
