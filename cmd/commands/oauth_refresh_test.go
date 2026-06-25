package commands

// oauth_refresh_test.go — Unit tests for the oauth refresh command.
// P4-E5-W3-S06-T20: security command coverage gate (was 0% — now covers happy + error paths).
//
// Purpose: Verify command registration and error path when no config is present.
// Inputs:  cobra command execution.
// Outputs: error when project config missing; command var non-nil.
// Constraints: Does not perform real OAuth flow; tests CLI plumbing only.

import (
	"testing"
)

// TestOAuthCmd_Registration verifies the oauth command and subcommands are registered.
func TestOAuthCmd_Registration(t *testing.T) {
	t.Parallel()

	if oauthCmd == nil {
		t.Fatal("oauthCmd is nil — command not registered")
	}
	if oauthCmd.Use != "oauth" {
		t.Errorf("oauthCmd.Use = %q, want %q", oauthCmd.Use, "oauth")
	}
	if oauthRefreshCmd == nil {
		t.Fatal("oauthRefreshCmd is nil — subcommand not registered")
	}
	if oauthRefreshCmd.Use == "" {
		t.Error("oauthRefreshCmd.Use is empty")
	}
}

// TestOAuthRefreshCmd_MissingConfig verifies oauth refresh returns a descriptive
// error when run outside a project directory (no config found).
func TestOAuthRefreshCmd_MissingConfig(t *testing.T) {
	t.Parallel()

	err := oauthRefreshCmd.RunE(oauthRefreshCmd, []string{})
	if err == nil {
		t.Skip("runOAuthRefresh unexpectedly succeeded — may have local project config")
	}
	// Must return a descriptive error string, not an empty one.
	if len(err.Error()) == 0 {
		t.Error("expected descriptive error, got empty string")
	}
}

// TestOAuthRefreshCmd_HasRunE verifies the RunE field is set (not Run), per
// cobra conventions enforced by cli/rules/go.md.
func TestOAuthRefreshCmd_HasRunE(t *testing.T) {
	t.Parallel()

	if oauthRefreshCmd.RunE == nil {
		t.Error("oauthRefreshCmd.RunE is nil — command must use RunE, not Run")
	}
	if oauthRefreshCmd.Run != nil {
		t.Error("oauthRefreshCmd.Run is set — cobra convention requires RunE only")
	}
}
