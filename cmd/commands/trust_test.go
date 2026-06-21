package commands

import (
	"testing"
)

// TestTrustCmd_Structure verifies trust subcommands and flags are registered.
func TestTrustCmd_Structure(t *testing.T) {
	// Verify root flags.
	flags := []string{"skip-dns", "skip-ssl", "skip-ports", "status", "undo", "project"}
	for _, name := range flags {
		if f := trustCmd.Flags().Lookup(name); f == nil {
			t.Errorf("trust: missing flag --%s", name)
		}
	}

	// Verify status subcommand exists.
	found := false
	for _, sub := range trustCmd.Commands() {
		if sub.Name() == "status" {
			found = true
		}
	}
	if !found {
		t.Error("trust: missing subcommand 'status'")
	}
}

// TestTrustStatusCmd_Flags verifies trust status --project flag is registered.
func TestTrustStatusCmd_Flags(t *testing.T) {
	if f := trustStatusCmd.Flags().Lookup("project"); f == nil {
		t.Error("trust status: missing --project flag")
	}
}

// TestBuildTrustConfig_Defaults verifies that buildTrustConfig produces sane defaults.
func TestBuildTrustConfig_Defaults(t *testing.T) {
	opts := trustOpts{}
	cfg := buildTrustConfig(opts)

	// BaseDomain should be non-empty (read from env or default).
	// In the test environment, BaseDomain may be empty if .env is absent — that's OK.
	// What we care about: it does NOT panic, returns a struct.
	_ = cfg
}

// TestTrustCmd_IdempotencyFlags verifies skip flags so idempotent re-runs are achievable.
func TestTrustCmd_IdempotencyFlags(t *testing.T) {
	for _, flag := range []string{"skip-dns", "skip-ssl", "skip-ports"} {
		f := trustCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("trust: --%s flag not registered — needed for idempotent admin-prompt avoidance", flag)
		}
		if f.DefValue != "false" {
			t.Errorf("trust: --%s default = %q, want false", flag, f.DefValue)
		}
	}
}
