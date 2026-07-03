package docker

import (
	"strings"
	"testing"
)

// TestBuildBaseArgs_EnvFiles verifies --env-file flags are emitted for each
// configured env file (secret templating: interpolation via .nself/compose.env).
func TestBuildBaseArgs_EnvFiles(t *testing.T) {
	c := NewCompose("/p/docker-compose.yml")
	c.EnvFiles = []string{"/p/.env", "/p/.nself/compose.env"}

	got := strings.Join(c.buildBaseArgs(), " ")
	want := "compose -f /p/docker-compose.yml --env-file /p/.env --env-file /p/.nself/compose.env"
	if got != want {
		t.Errorf("buildBaseArgs() = %q, want %q", got, want)
	}
}

// TestBuildBaseArgs_NoEnvFiles keeps legacy behavior: no --env-file flags.
func TestBuildBaseArgs_NoEnvFiles(t *testing.T) {
	c := NewCompose("/p/docker-compose.yml")
	got := strings.Join(c.buildBaseArgs(), " ")
	if strings.Contains(got, "--env-file") {
		t.Errorf("unexpected --env-file in %q", got)
	}
}
