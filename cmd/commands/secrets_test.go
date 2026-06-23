package commands

// secrets_test.go — unit tests for the secrets command.
//
// Purpose: verify command registration, subcommand wiring, and argument
// constraints for nself secrets. Tests run offline — no age encryption,
// no filesystem secrets store.

import (
	"testing"
)

// TestSecretsCmd_SubcommandsRegistered verifies that the core secrets subcommands
// are registered under the secrets parent command.
func TestSecretsCmd_SubcommandsRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range secretsCmd.Commands() {
		names[sub.Name()] = true
	}
	required := []string{"init", "set", "get", "list", "edit", "audit", "lint"}
	for _, req := range required {
		if !names[req] {
			t.Errorf("secrets subcommand %q not registered", req)
		}
	}
}

// TestSecretsSetCmd_ArgRange verifies that secrets set accepts 1 or 2 args
// (KEY only, or KEY VALUE) and rejects 0 or 3.
func TestSecretsSetCmd_ArgRange(t *testing.T) {
	if secretsSetCmd.Args == nil {
		t.Fatal("secrets set should enforce RangeArgs(1,2) — Args validator is nil")
	}
	// 0 args: invalid.
	if err := secretsSetCmd.Args(secretsSetCmd, []string{}); err == nil {
		t.Error("expected error for 0 args (RangeArgs(1,2))")
	}
	// 1 arg (KEY only): valid.
	if err := secretsSetCmd.Args(secretsSetCmd, []string{"MY_KEY"}); err != nil {
		t.Errorf("unexpected error for 1 arg: %v", err)
	}
	// 2 args (KEY VALUE): valid.
	if err := secretsSetCmd.Args(secretsSetCmd, []string{"MY_KEY", "my_value"}); err != nil {
		t.Errorf("unexpected error for 2 args: %v", err)
	}
	// 3 args: invalid.
	if err := secretsSetCmd.Args(secretsSetCmd, []string{"K", "V", "extra"}); err == nil {
		t.Error("expected error for 3 args (RangeArgs(1,2))")
	}
}

// TestSecretsGetCmd_ExactArgs verifies that secrets get accepts exactly 1 arg.
func TestSecretsGetCmd_ExactArgs(t *testing.T) {
	if secretsGetCmd.Args == nil {
		t.Fatal("secrets get should enforce ExactArgs(1) — Args validator is nil")
	}
	// 0 args: invalid.
	if err := secretsGetCmd.Args(secretsGetCmd, []string{}); err == nil {
		t.Error("expected error for 0 args (ExactArgs(1))")
	}
	// 1 arg: valid.
	if err := secretsGetCmd.Args(secretsGetCmd, []string{"MY_KEY"}); err != nil {
		t.Errorf("unexpected error for 1 arg: %v", err)
	}
	// 2 args: invalid.
	if err := secretsGetCmd.Args(secretsGetCmd, []string{"K1", "K2"}); err == nil {
		t.Error("expected error for 2 args (ExactArgs(1))")
	}
}

// TestSecretsCmd_HasRunE verifies the parent secrets command has a RunE handler
// (delegates to help) so it does not silently no-op.
func TestSecretsCmd_HasRunE(t *testing.T) {
	if secretsCmd.RunE == nil {
		t.Error("secrets command has no RunE handler")
	}
}
