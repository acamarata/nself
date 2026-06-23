package commands

// oauth_refresh_test.go — unit tests for the oauth refresh command.
//
// Purpose: verify command registration, argument validation, and flag wiring
// for nself oauth refresh. Tests run offline — no network, no OAuth provider.

import (
	"testing"
)

// TestOAuthRefreshCmd_MaxArgs verifies that the command accepts 0 or 1 provider
// arg and rejects 2 or more.
func TestOAuthRefreshCmd_MaxArgs(t *testing.T) {
	if oauthRefreshCmd.Args == nil {
		t.Fatal("oauth refresh should enforce MaximumNArgs(1) — Args validator is nil")
	}
	// 0 args: valid (refresh all or error from flag).
	if err := oauthRefreshCmd.Args(oauthRefreshCmd, []string{}); err != nil {
		t.Errorf("unexpected error for 0 args: %v", err)
	}
	// 1 arg: valid (named provider).
	if err := oauthRefreshCmd.Args(oauthRefreshCmd, []string{"google"}); err != nil {
		t.Errorf("unexpected error for 1 arg: %v", err)
	}
	// 2 args: invalid.
	if err := oauthRefreshCmd.Args(oauthRefreshCmd, []string{"google", "github"}); err == nil {
		t.Error("expected error for 2 args (MaximumNArgs(1))")
	}
}

// TestOAuthCmd_SubcommandsRegistered verifies the refresh subcommand is
// registered under the oauth parent command.
func TestOAuthCmd_SubcommandsRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range oauthCmd.Commands() {
		names[sub.Name()] = true
	}
	if !names["refresh"] {
		t.Error("oauth subcommand 'refresh' not registered")
	}
}

// TestOAuthRefreshCmd_FlagsRegistered verifies that the --all, --account-id,
// and --base-url flags are registered on the refresh subcommand.
func TestOAuthRefreshCmd_FlagsRegistered(t *testing.T) {
	required := []string{"all", "account-id", "base-url"}
	for _, flag := range required {
		if oauthRefreshCmd.Flags().Lookup(flag) == nil {
			t.Errorf("oauth refresh flag --%s not registered", flag)
		}
	}
}

// TestOAuthRefreshCmd_HasRunE verifies the command has a RunE handler.
func TestOAuthRefreshCmd_HasRunE(t *testing.T) {
	if oauthRefreshCmd.RunE == nil {
		t.Error("oauth refresh has no RunE handler")
	}
}
