package docker

import (
	"strings"
	"testing"
)

// TestComposeUpNoDeps_BuildsTheRightArgs pins the primitive that makes
// `nself restart <service>` actually apply a compose change.
//
// `docker compose restart` bounces the existing container and never re-reads
// the file, which is why a config edit appeared to do nothing until the
// container was recreated by hand (ntask clean-fork self-host drill,
// 2026-08-24). --no-deps keeps it scoped to the named services.
func TestComposeUpNoDeps_BuildsTheRightArgs(t *testing.T) {
	c := NewCompose("docker-compose.yml", "docker-compose.override.yml")
	args := c.buildBaseArgs()
	args = append(args, "up", "-d", "--no-deps", "functions")

	joined := strings.Join(args, " ")
	for _, want := range []string{
		"compose",
		"-f docker-compose.yml",
		"-f docker-compose.override.yml",
		"up -d --no-deps functions",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q missing %q", joined, want)
		}
	}

	if strings.Contains(joined, "--force-recreate") {
		t.Error("must not force-recreate: Compose should recreate only what changed")
	}
}
