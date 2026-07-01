package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/controlplane"
	"github.com/spf13/cobra"
)

// newDBRemoteTestCmd returns a minimal cobra.Command with --env/--server
// registered via addDBRemoteFlags, mirroring how dbMigrateUpCmd etc. are
// wired in db.go's init().
func newDBRemoteTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "up"}
	addDBRemoteFlags(cmd)
	cmd.SetContext(context.Background())
	return cmd
}

// TestAddDBRemoteFlags_Registered verifies --env and --server are both
// registered by addDBRemoteFlags (gap #9: these commands had zero flags
// before this change).
func TestAddDBRemoteFlags_Registered(t *testing.T) {
	cmd := newDBRemoteTestCmd()
	if cmd.Flags().Lookup("env") == nil {
		t.Error("expected --env flag to be registered")
	}
	if cmd.Flags().Lookup("server") == nil {
		t.Error("expected --server flag to be registered")
	}
}

// TestResolveDBRemoteTarget_DefaultsLocal verifies that omitting --env/--server
// entirely resolves to Local=true, preserving today's default behavior for
// every existing caller that doesn't pass these new flags.
func TestResolveDBRemoteTarget_DefaultsLocal(t *testing.T) {
	cmd := newDBRemoteTestCmd()
	target, err := resolveDBRemoteTarget(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !target.Local {
		t.Error("expected Local=true when --env/--server are unset")
	}
}

// TestResolveDBRemoteTarget_EnvLocal verifies --env=local also resolves to
// Local=true without touching control-plane inventory.
func TestResolveDBRemoteTarget_EnvLocal(t *testing.T) {
	cmd := newDBRemoteTestCmd()
	if err := cmd.Flags().Set("env", "local"); err != nil {
		t.Fatalf("set --env: %v", err)
	}
	target, err := resolveDBRemoteTarget(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !target.Local {
		t.Error("expected Local=true for --env=local")
	}
}

// TestResolveDBRemoteTarget_UnknownEnvironment verifies that requesting a
// remote environment with no matching control-plane inventory entry and no
// legacy env var returns a descriptive error rather than silently running
// locally (which would silently apply migrations to the wrong database).
func TestResolveDBRemoteTarget_UnknownEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cmd := newDBRemoteTestCmd()
	if err := cmd.Flags().Set("env", "staging"); err != nil {
		t.Fatalf("set --env: %v", err)
	}
	_, err := resolveDBRemoteTarget(cmd)
	if err == nil {
		t.Fatal("expected an error when --env=staging has no configured inventory/host")
	}
	if !strings.Contains(err.Error(), "staging") {
		t.Errorf("expected error to mention 'staging', got: %v", err)
	}
}

// TestResolveDBRemoteTarget_RemoteServer verifies that a control-plane
// inventory entry with a real Host resolves to a non-local target carrying
// the expected SSH target and remote path.
func TestResolveDBRemoteTarget_RemoteServer(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	inv := &controlplane.Inventory{
		SchemaVersion: 1,
		Project:       "test",
		Environments: map[string]controlplane.Environment{
			"staging": {
				Name: "staging",
				Kind: "remote",
				Servers: []controlplane.Server{
					{
						Name:       "staging-app",
						Role:       controlplane.RoleApp,
						Host:       "deploy@staging.example.com",
						RemotePath: "/opt/nself",
						Primary:    true,
					},
				},
			},
		},
	}
	if err := controlplane.Write(dir, inv); err != nil {
		t.Fatalf("write inventory: %v", err)
	}

	cmd := newDBRemoteTestCmd()
	if err := cmd.Flags().Set("env", "staging"); err != nil {
		t.Fatalf("set --env: %v", err)
	}
	target, err := resolveDBRemoteTarget(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Local {
		t.Fatal("expected Local=false for a staging environment with a real Host")
	}
	if target.SSHTarget != "deploy@staging.example.com" {
		t.Errorf("SSHTarget: got %q, want %q", target.SSHTarget, "deploy@staging.example.com")
	}
	if target.RemotePath != "/opt/nself" {
		t.Errorf("RemotePath: got %q, want %q", target.RemotePath, "/opt/nself")
	}
	if target.EnvName != "staging" {
		t.Errorf("EnvName: got %q, want %q", target.EnvName, "staging")
	}
}

// TestResolveDBRemoteTarget_ServerFlagOverridesPrimary verifies that
// --server picks a named non-primary server instead of the environment's
// primary server.
func TestResolveDBRemoteTarget_ServerFlagOverridesPrimary(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	inv := &controlplane.Inventory{
		SchemaVersion: 1,
		Project:       "test",
		Environments: map[string]controlplane.Environment{
			"staging": {
				Name: "staging",
				Kind: "remote",
				Servers: []controlplane.Server{
					{Name: "staging-app", Role: controlplane.RoleApp, Host: "deploy@app.example.com", Primary: true},
					{Name: "staging-db", Role: controlplane.RoleDB, Host: "deploy@db.example.com"},
				},
			},
		},
	}
	if err := controlplane.Write(dir, inv); err != nil {
		t.Fatalf("write inventory: %v", err)
	}

	cmd := newDBRemoteTestCmd()
	if err := cmd.Flags().Set("env", "staging"); err != nil {
		t.Fatalf("set --env: %v", err)
	}
	if err := cmd.Flags().Set("server", "staging-db"); err != nil {
		t.Fatalf("set --server: %v", err)
	}
	target, err := resolveDBRemoteTarget(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.SSHTarget != "deploy@db.example.com" {
		t.Errorf("expected --server to override primary selection; got SSHTarget=%q", target.SSHTarget)
	}
}

// TestFindDBTargetServer_PrimaryPreferred verifies the Primary server is
// picked when no explicit server name is requested.
func TestFindDBTargetServer_PrimaryPreferred(t *testing.T) {
	env := controlplane.Environment{
		Servers: []controlplane.Server{
			{Name: "a", Primary: false},
			{Name: "b", Primary: true},
			{Name: "c", Primary: false},
		},
	}
	srv, ok := findDBTargetServer(env, "")
	if !ok || srv.Name != "b" {
		t.Errorf("expected primary server 'b', got %+v (ok=%v)", srv, ok)
	}
}

// TestFindDBTargetServer_FallsBackToFirst verifies the first server is used
// when no server is marked Primary.
func TestFindDBTargetServer_FallsBackToFirst(t *testing.T) {
	env := controlplane.Environment{
		Servers: []controlplane.Server{
			{Name: "a"},
			{Name: "b"},
		},
	}
	srv, ok := findDBTargetServer(env, "")
	if !ok || srv.Name != "a" {
		t.Errorf("expected fallback to first server 'a', got %+v (ok=%v)", srv, ok)
	}
}

// TestFindDBTargetServer_NotFound verifies a named-but-missing server
// returns ok=false rather than a zero-value match.
func TestFindDBTargetServer_NotFound(t *testing.T) {
	env := controlplane.Environment{Servers: []controlplane.Server{{Name: "a"}}}
	if _, ok := findDBTargetServer(env, "does-not-exist"); ok {
		t.Error("expected ok=false for a server name not present in the environment")
	}
}

// TestShellQuoteArg_EscapesSingleQuotes verifies embedded single quotes are
// escaped POSIX-style so remote command strings can't break out of the
// quoted argument.
func TestShellQuoteArg_EscapesSingleQuotes(t *testing.T) {
	got := shellQuoteArg("it's a test")
	want := `'it'\''s a test'`
	if got != want {
		t.Errorf("shellQuoteArg: got %q, want %q", got, want)
	}
}

// TestShellQuoteArg_Empty verifies an empty string quotes to an empty pair
// of quotes rather than an unquoted empty argument (which a shell would drop).
func TestShellQuoteArg_Empty(t *testing.T) {
	if got := shellQuoteArg(""); got != "''" {
		t.Errorf("shellQuoteArg(\"\"): got %q, want \"''\"", got)
	}
}

// TestWrapRemoteVersionDriftError_DetectsCommandNotFound verifies gap #16:
// a remote "command not found"/"unknown command" style failure is turned
// into an actionable version-drift message instead of a bare wrapped error.
func TestWrapRemoteVersionDriftError_DetectsCommandNotFound(t *testing.T) {
	rt := dbRemoteTarget{SSHTarget: "deploy@staging.example.com", EnvName: "staging"}
	err := wrapRemoteVersionDriftError(rt, []string{"db", "migrate", "up"}, os.ErrDeadlineExceeded, "bash: nself: command not found")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if !strings.Contains(err.Error(), "older version") {
		t.Errorf("expected version-drift guidance in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--version") {
		t.Errorf("expected a suggestion to check --version, got: %v", err)
	}
}

// TestWrapRemoteVersionDriftError_PassesThroughOtherErrors verifies that an
// unrelated remote failure (e.g. a real migration SQL error) is still
// wrapped with context but is NOT misreported as a version-drift issue.
func TestWrapRemoteVersionDriftError_PassesThroughOtherErrors(t *testing.T) {
	rt := dbRemoteTarget{SSHTarget: "deploy@staging.example.com", EnvName: "staging"}
	err := wrapRemoteVersionDriftError(rt, []string{"db", "migrate", "up"}, os.ErrDeadlineExceeded, "ERROR: relation \"foo\" does not exist")
	if err == nil {
		t.Fatal("expected a non-nil error")
	}
	if strings.Contains(err.Error(), "older version") {
		t.Errorf("did not expect version-drift guidance for an unrelated SQL error, got: %v", err)
	}
}

// TestDispatchRemoteIfNeeded_NoFlagsRegistered verifies commands without
// --env/--server registered (e.g. minimal test cobra.Commands elsewhere in
// this package) are treated as local, never erroring on a missing flag.
func TestDispatchRemoteIfNeeded_NoFlagsRegistered(t *testing.T) {
	cmd := &cobra.Command{Use: "status"}
	cmd.SetContext(context.Background())
	handled, err := dispatchRemoteIfNeeded(cmd, "db", "migrate", "status")
	if handled {
		t.Error("expected handled=false when --env/--server are not registered on cmd")
	}
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

// TestDispatchRemoteIfNeeded_LocalIsNotHandled verifies that the default
// (no --env/--server) path returns handled=false so callers proceed with
// their original local implementation unchanged.
func TestDispatchRemoteIfNeeded_LocalIsNotHandled(t *testing.T) {
	cmd := newDBRemoteTestCmd()
	handled, err := dispatchRemoteIfNeeded(cmd, "db", "migrate", "up")
	if handled {
		t.Error("expected handled=false for the default local target")
	}
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

// TestMigrateUpCmd_RemoteFlagsRegistered verifies the real dbMigrateUpCmd,
// dbMigrateStatusCmd, and dbHasuraMetadataApplyCmd (wired in db.go's init())
// carry the new --env/--server flags, not just a throwaway test command.
func TestMigrateUpCmd_RemoteFlagsRegistered(t *testing.T) {
	for _, cmd := range []*cobra.Command{dbMigrateUpCmd, dbMigrateStatusCmd, dbHasuraMetadataApplyCmd} {
		if cmd.Flags().Lookup("env") == nil {
			t.Errorf("%s: expected --env flag to be registered", cmd.Use)
		}
		if cmd.Flags().Lookup("server") == nil {
			t.Errorf("%s: expected --server flag to be registered", cmd.Use)
		}
	}
}

// TestResolveDBRemoteTarget_ProjectRootIsRespected verifies resolution reads
// control-plane inventory relative to the resolved project root (via
// projectRoot()), not just the raw cwd — regression guard for running these
// commands from a subdirectory.
func TestResolveDBRemoteTarget_ProjectRootIsRespected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("PROJECT_NAME=test\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	inv := &controlplane.Inventory{
		SchemaVersion: 1,
		Project:       "test",
		Environments: map[string]controlplane.Environment{
			"staging": {
				Name:    "staging",
				Kind:    "remote",
				Servers: []controlplane.Server{{Name: "staging-app", Host: "deploy@example.com", Primary: true}},
			},
		},
	}
	if err := controlplane.Write(dir, inv); err != nil {
		t.Fatalf("write inventory: %v", err)
	}
	t.Chdir(dir)

	cmd := newDBRemoteTestCmd()
	if err := cmd.Flags().Set("env", "staging"); err != nil {
		t.Fatalf("set --env: %v", err)
	}
	target, err := resolveDBRemoteTarget(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.SSHTarget != "deploy@example.com" {
		t.Errorf("expected resolution from project root inventory, got SSHTarget=%q", target.SSHTarget)
	}
}
