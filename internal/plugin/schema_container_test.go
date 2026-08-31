package plugin

// schema_container_test.go — Pins the Postgres container name this package uses.
//
// Purpose: `nself plugin install` shells out to `docker exec <container> psql`
//          to create a plugin's schema. Both call sites in schema.go used
//          "<project>-postgres-1", Docker Compose's DEFAULT naming — but
//          `nself build` writes `container_name: ${PROJECT_NAME}_postgres`,
//          which overrides that default. Every exec therefore failed with
//          "No such container", and plugin install was broken for EVERY
//          schema-bearing plugin on EVERY project. A live smoke test of
//          ntask's 7 declared plugins installed 0 of 7 for this reason.
// Inputs:  a config with a project name.
// Outputs: assertions on the container name string.
// Constraints: pure string construction — no docker, no network.

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

func TestPostgresContainer_MatchesGeneratedComposeName(t *testing.T) {
	got := postgresContainer(&config.Config{ProjectName: "ntask"})

	// This is the literal container_name nself build emits for ntask.
	if want := "ntask_postgres"; got != want {
		t.Errorf("postgresContainer = %q, want %q", got, want)
	}

	// The specific wrong answer that shipped. Compose's default naming is
	// <project>-<service>-<index>; nself build overrides it, so this form
	// addresses a container that does not exist.
	if strings.Contains(got, "-postgres-1") {
		t.Error("container name regressed to Docker Compose default naming; " +
			"nself build sets container_name explicitly and this will 'No such container'")
	}
}

// TestPostgresContainer_AgreesWithRestOfCLI guards the convention rather than
// one spelling. internal/database/helpers.go, internal/database/backup.go and
// six files under internal/tenant all build this name as
// ProjectName + "_postgres". This package was the only one that disagreed.
func TestPostgresContainer_AgreesWithRestOfCLI(t *testing.T) {
	for _, project := range []string{"ntask", "nself-web", "my_app", "a"} {
		got := postgresContainer(&config.Config{ProjectName: project})
		if want := project + "_postgres"; got != want {
			t.Errorf("project %q: got %q, want %q", project, got, want)
		}
	}
}
