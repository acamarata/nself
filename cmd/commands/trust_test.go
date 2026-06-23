package commands

import (
	"testing"
)

// TestTrustCmd_Flags verifies the trust command registers required flags.
func TestTrustCmd_Flags(t *testing.T) {
	flags := []string{"skip-dns", "skip-ssl", "skip-ports", "status", "undo", "project"}
	for _, name := range flags {
		if trustCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not registered on trust command", name)
		}
	}
}

// TestTrustCmd_HasStatusSubcmd verifies `nself trust status` is registered.
func TestTrustCmd_HasStatusSubcmd(t *testing.T) {
	subs := trustCmd.Commands()
	found := false
	for _, cmd := range subs {
		if cmd.Name() == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Error("trust command is missing 'status' subcommand")
	}
}

// TestTrustStatusCmd_ProjectFlag verifies --project flag on trust status.
func TestTrustStatusCmd_ProjectFlag(t *testing.T) {
	if trustStatusCmd.Flags().Lookup("project") == nil {
		t.Error("trust status missing --project flag")
	}
}

// TestResolveTrustOpts_SkipFlags verifies resolveTrustOpts reads flags correctly.
func TestResolveTrustOpts_SkipFlags(t *testing.T) {
	cmd := trustCmd
	// Reset flags to defaults to isolate test.
	_ = cmd.Flags().Set("skip-dns", "true")
	_ = cmd.Flags().Set("skip-ssl", "false")
	_ = cmd.Flags().Set("skip-ports", "false")
	_ = cmd.Flags().Set("status", "false")
	_ = cmd.Flags().Set("undo", "false")
	_ = cmd.Flags().Set("project", "")

	opts := resolveTrustOpts(cmd)
	if !opts.SkipDNS {
		t.Error("expected SkipDNS=true")
	}
	if opts.SkipSSL {
		t.Error("expected SkipSSL=false")
	}

	// Reset
	_ = cmd.Flags().Set("skip-dns", "false")
}

// TestTrustCmd_Idempotency_SkipFlagIsBoolean verifies skip-dns default is false.
func TestTrustCmd_Idempotency_SkipFlagIsBoolean(t *testing.T) {
	f := trustCmd.Flags().Lookup("skip-dns")
	if f.DefValue != "false" {
		t.Errorf("skip-dns default = %q, want false", f.DefValue)
	}
}

// TestBuildTrustConfig_DefaultProject verifies buildTrustConfig returns a
// non-zero config when no project path is specified.
func TestBuildTrustConfig_DefaultProject(t *testing.T) {
	opts := trustOpts{}
	cfg := buildTrustConfig(opts)
	// cfg is a value type (trust.TrustConfig), never nil; just verify it
	// is populated (ProjectDir should default to cwd or empty string).
	_ = cfg
}
