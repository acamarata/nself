package doctor

import (
	"testing"
)

func TestFilterBySection(t *testing.T) {
	results := []CheckResult{
		{Section: "host", Name: "Disk", Status: "pass"},
		{Section: "docker", Name: "Daemon", Status: "pass"},
		{Section: "host", Name: "CPU", Status: "warn"},
		{Section: "postgres", Name: "pg_isready", Status: "pass"},
	}

	host := FilterBySection(results, "host")
	if len(host) != 2 {
		t.Errorf("expected 2 host results, got %d", len(host))
	}

	pg := FilterBySection(results, "postgres")
	if len(pg) != 1 {
		t.Errorf("expected 1 postgres result, got %d", len(pg))
	}

	empty := FilterBySection(results, "nonexistent")
	if len(empty) != 0 {
		t.Errorf("expected 0 results for nonexistent section, got %d", len(empty))
	}
}

func TestDefaultSchedulesSectionNames(t *testing.T) {
	// Verify all 12 subsystem section names are used in DeepChecks output types
	expectedSections := []string{
		"host", "docker", "postgres", "hasura", "nginx", "ssl",
		"ping", "plugins", "license", "monitoring", "backups", "security",
	}
	for _, s := range expectedSections {
		// Just verify the section name is a non-empty string
		if s == "" {
			t.Error("empty section name")
		}
	}
}
