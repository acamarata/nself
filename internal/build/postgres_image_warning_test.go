package build

import (
	"strings"
	"testing"
)

// TestPostgresImageChangeWarning_NoRunningContainer verifies the advisory
// check never errors or fabricates a warning when the container doesn't
// exist (e.g. Docker isn't running, or this is a first-ever build) — a
// missing baseline must never block `nself build`.
func TestPostgresImageChangeWarning_NoRunningContainer(t *testing.T) {
	got := PostgresImageChangeWarning("nonexistent_project_postgres_container_xyz", "pgvector/pgvector:pg16")
	if got != "" {
		t.Errorf("PostgresImageChangeWarning(no container) = %q, want empty", got)
	}
}

// TestPostgresImageChangeWarning_Mismatch verifies cli#384's core case: a
// running pgvector container against a regen that would produce a plain
// alpine image must produce a WARNING naming both images and the fix.
func TestPostgresImageChangeWarning_Mismatch(t *testing.T) {
	lookup := func(name string) string {
		if name == "myproj_postgres" {
			return "pgvector/pgvector:pg16"
		}
		return ""
	}
	got := postgresImageChangeWarning("myproj_postgres", "postgres:16-alpine", lookup)
	if got == "" {
		t.Fatal("expected a warning, got empty string")
	}
	for _, want := range []string{"myproj_postgres", "pgvector/pgvector:pg16", "postgres:16-alpine", "POSTGRES_IMAGE"} {
		if !strings.Contains(got, want) {
			t.Errorf("warning %q missing expected substring %q", got, want)
		}
	}
}

// TestPostgresImageChangeWarning_Match verifies no warning is raised when the
// running container's image already matches what this build generated.
func TestPostgresImageChangeWarning_Match(t *testing.T) {
	lookup := func(string) string { return "pgvector/pgvector:pg16" }
	got := postgresImageChangeWarning("myproj_postgres", "pgvector/pgvector:pg16", lookup)
	if got != "" {
		t.Errorf("postgresImageChangeWarning(match) = %q, want empty", got)
	}
}

// TestPostgresImageChangeWarning_EmptyBaselineIsSilent verifies an empty
// lookup result (no running container, or Docker unreachable) is treated as
// "nothing to compare", not as a mismatch.
func TestPostgresImageChangeWarning_EmptyBaselineIsSilent(t *testing.T) {
	lookup := func(string) string { return "" }
	got := postgresImageChangeWarning("myproj_postgres", "postgres:16-alpine", lookup)
	if got != "" {
		t.Errorf("postgresImageChangeWarning(empty baseline) = %q, want empty", got)
	}
}
