package commands

// Tests exercise the `nself access` cobra wiring end to end against a
// LocalFileTransport fixture (via the newAccessTransport indirection in
// access_transport.go) — never against a real SSH connection. See
// internal/access/manager_test.go for the deeper grant/revoke/list behavior
// tests; these tests confirm the CLI layer wires flags to that package
// correctly.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/access"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const testKeyLine = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMHXHuK8L4SFSmmpHWBnzPFAcJGYHjABCulfo5ZbKvum alice@laptop"

// withFixtureTransport points newAccessTransport at a LocalFileTransport
// rooted in a temp file for the duration of the test, restoring the real
// (SSH-backed) factory afterward.
func withFixtureTransport(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "authorized_keys")

	restoreAudit := access.SetAuditLogPathForTest(filepath.Join(dir, "audit.log"))
	old := newAccessTransport
	newAccessTransport = func(cmd *cobra.Command) (access.Transport, error) {
		return access.NewLocalFileTransport(path), nil
	}
	t.Cleanup(func() {
		newAccessTransport = old
		restoreAudit()
	})
	return path
}

// resetFlags restores every flag on cmd to its registered default so tests
// calling RunE repeatedly on the shared package-level command vars don't leak
// state between each other.
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

func TestAccessCmd_Structure(t *testing.T) {
	names := map[string]bool{}
	for _, c := range accessCmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"grant", "revoke", "list"} {
		if !names[want] {
			t.Errorf("missing subcommand: %s", want)
		}
	}
}

func TestAccessGrantCmd_RequiredFlags(t *testing.T) {
	for _, name := range []string{"host", "identity", "user", "key", "sudo", "docker", "expires", "dry-run"} {
		if accessGrantCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing --%s flag on access grant", name)
		}
	}
}

func TestAccessRevokeCmd_RequiredFlags(t *testing.T) {
	for _, name := range []string{"host", "identity", "user", "force", "dry-run"} {
		if accessRevokeCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing --%s flag on access revoke", name)
		}
	}
}

func TestAccessListCmd_RequiredFlags(t *testing.T) {
	for _, name := range []string{"host", "identity", "json"} {
		if accessListCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing --%s flag on access list", name)
		}
	}
}

func TestRunAccessGrant_MissingUser(t *testing.T) {
	defer resetFlags(accessGrantCmd)
	withFixtureTransport(t)
	accessGrantCmd.SetContext(context.Background())
	_ = accessGrantCmd.Flags().Set("key", testKeyLine)

	if err := runAccessGrant(accessGrantCmd, nil); err == nil {
		t.Fatal("expected error when --user is missing")
	}
}

func TestRunAccessGrant_MissingKey(t *testing.T) {
	defer resetFlags(accessGrantCmd)
	withFixtureTransport(t)
	accessGrantCmd.SetContext(context.Background())
	_ = accessGrantCmd.Flags().Set("user", "alice")

	if err := runAccessGrant(accessGrantCmd, nil); err == nil {
		t.Fatal("expected error when --key is missing")
	}
}

func TestRunAccessGrant_InvalidExpiry(t *testing.T) {
	defer resetFlags(accessGrantCmd)
	withFixtureTransport(t)
	accessGrantCmd.SetContext(context.Background())
	_ = accessGrantCmd.Flags().Set("user", "alice")
	_ = accessGrantCmd.Flags().Set("key", testKeyLine)
	_ = accessGrantCmd.Flags().Set("expires", "not-a-date")

	if err := runAccessGrant(accessGrantCmd, nil); err == nil {
		t.Fatal("expected error for a malformed --expires value")
	}
}

func TestRunAccessGrant_MissingHost(t *testing.T) {
	defer resetFlags(accessGrantCmd)
	// Deliberately do not call withFixtureTransport: this exercises the real
	// buildAccessTransport path, which must reject an empty --host before
	// ever touching SSH.
	accessGrantCmd.SetContext(context.Background())
	_ = accessGrantCmd.Flags().Set("user", "alice")
	_ = accessGrantCmd.Flags().Set("key", testKeyLine)

	if err := runAccessGrant(accessGrantCmd, nil); err == nil {
		t.Fatal("expected error when --host is missing")
	}
}

func TestAccessGrantListRevoke_EndToEnd(t *testing.T) {
	path := withFixtureTransport(t)
	ctx := context.Background()

	defer resetFlags(accessGrantCmd)
	accessGrantCmd.SetContext(ctx)
	_ = accessGrantCmd.Flags().Set("host", "root@fixture")
	_ = accessGrantCmd.Flags().Set("user", "alice")
	_ = accessGrantCmd.Flags().Set("key", testKeyLine)
	if err := runAccessGrant(accessGrantCmd, nil); err != nil {
		t.Fatalf("runAccessGrant: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected authorized_keys to be created: %v", err)
	}

	defer resetFlags(accessListCmd)
	accessListCmd.SetContext(ctx)
	_ = accessListCmd.Flags().Set("host", "root@fixture")
	if err := runAccessList(accessListCmd, nil); err != nil {
		t.Fatalf("runAccessList: %v", err)
	}

	defer resetFlags(accessRevokeCmd)
	accessRevokeCmd.SetContext(ctx)
	_ = accessRevokeCmd.Flags().Set("host", "root@fixture")
	_ = accessRevokeCmd.Flags().Set("user", "alice")
	_ = accessRevokeCmd.Flags().Set("force", "true")
	if err := runAccessRevoke(accessRevokeCmd, nil); err != nil {
		t.Fatalf("runAccessRevoke: %v", err)
	}
}

func TestRunAccessRevoke_MissingUser(t *testing.T) {
	defer resetFlags(accessRevokeCmd)
	withFixtureTransport(t)
	accessRevokeCmd.SetContext(context.Background())

	if err := runAccessRevoke(accessRevokeCmd, nil); err == nil {
		t.Fatal("expected error when --user is missing")
	}
}

func TestDefaultIdentityPath_NonEmpty(t *testing.T) {
	if defaultIdentityPath() == "" {
		t.Error("defaultIdentityPath() must never be empty")
	}
}

func TestParseExpiry_EmptyIsNil(t *testing.T) {
	got, err := parseExpiry("")
	if err != nil {
		t.Fatalf("parseExpiry(\"\"): %v", err)
	}
	if got != nil {
		t.Errorf("parseExpiry(\"\") = %v, want nil", got)
	}
}

func TestParseExpiry_InvalidFormat(t *testing.T) {
	if _, err := parseExpiry("31-12-2026"); err == nil {
		t.Fatal("expected error for non-ISO date format")
	}
}
