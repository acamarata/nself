package deploy

import (
	"strings"
	"testing"
)

// TestResolveRemoteTargetFromEnv_NoHostConfigured verifies ok=false is
// returned when no NSELF_DEPLOY_HOST_<TARGET> (or legacy <TARGET>_DEPLOY_HOST)
// env var is set, so callers fall back to local execution — the default,
// back-compat path for every project that hasn't configured remote deploy.
func TestResolveRemoteTargetFromEnv_NoHostConfigured(t *testing.T) {
	t.Setenv("NSELF_DEPLOY_HOST_STAGING", "")
	t.Setenv("STAGING_DEPLOY_HOST", "")

	_, ok := ResolveRemoteTargetFromEnv("staging")
	if ok {
		t.Error("expected ok=false when no host env var is configured")
	}
}

// TestResolveRemoteTargetFromEnv_SplitsHostAndPath verifies a configured
// "user@host:/remote/path" value is split into SSHTarget/RemotePath.
func TestResolveRemoteTargetFromEnv_SplitsHostAndPath(t *testing.T) {
	t.Setenv("NSELF_DEPLOY_HOST_STAGING", "deploy@staging.example.com:/opt/nself")

	rt, ok := ResolveRemoteTargetFromEnv("staging")
	if !ok {
		t.Fatal("expected ok=true for a configured host")
	}
	if rt.SSHTarget != "deploy@staging.example.com" {
		t.Errorf("SSHTarget: got %q, want %q", rt.SSHTarget, "deploy@staging.example.com")
	}
	if rt.RemotePath != "/opt/nself" {
		t.Errorf("RemotePath: got %q, want %q", rt.RemotePath, "/opt/nself")
	}
}

// TestRunRemoteCommand_EmptySSHTarget verifies a RemoteTarget with no
// SSHTarget fails fast with a descriptive error instead of shelling out to
// `ssh` with an empty destination argument.
func TestRunRemoteCommand_EmptySSHTarget(t *testing.T) {
	_, err := RunRemoteCommand(nil, RemoteTarget{}, "echo hi") //nolint:staticcheck // nil ctx is fine; guard fires before any ctx use
	if err == nil {
		t.Fatal("expected an error for an empty SSH target")
	}
	if !strings.Contains(err.Error(), "no SSH host") {
		t.Errorf("expected 'no SSH host' error, got: %v", err)
	}
}

// TestRemoteDockerExecCommand_QuotesArguments verifies the built command
// string shell-quotes the container name and each argument, so a container
// name or argument containing spaces/metacharacters can't break the remote
// command out of its intended shape.
func TestRemoteDockerExecCommand_QuotesArguments(t *testing.T) {
	got := RemoteDockerExecCommand("myproject_postgres", "psql", "-U", "postgres", "-c", "SELECT 1;")
	want := `docker exec 'myproject_postgres' 'psql' '-U' 'postgres' '-c' 'SELECT 1;'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestShellQuote_EscapesEmbeddedSingleQuotes verifies POSIX-style escaping
// of a single quote embedded in an argument.
func TestShellQuote_EscapesEmbeddedSingleQuotes(t *testing.T) {
	got := shellQuote("O'Brien")
	want := `'O'\''Brien'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
