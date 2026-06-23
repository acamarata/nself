package commands

import (
	"testing"
)

// TestSSLCmd_Structure verifies ssl subcommands are registered.
func TestSSLCmd_Structure(t *testing.T) {
	subs := sslCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subs {
		names[cmd.Name()] = true
	}
	for _, name := range []string{"status", "renew"} {
		if !names[name] {
			t.Errorf("missing ssl subcommand: %s", name)
		}
	}
}

// TestSSLCmd_Use verifies the command Use field.
func TestSSLCmd_Use(t *testing.T) {
	if sslCmd.Use != "ssl" {
		t.Errorf("Use = %q, want ssl", sslCmd.Use)
	}
}

// TestSSLRenewCmd_MaxArgs verifies renew accepts 0 or 1 arg (no more).
func TestSSLRenewCmd_MaxArgs(t *testing.T) {
	if sslRenewCmd.Args == nil {
		t.Fatal("ssl renew should enforce MaximumNArgs(1)")
	}
	// Valid: 0 args
	if err := sslRenewCmd.Args(sslRenewCmd, []string{}); err != nil {
		t.Errorf("unexpected error for 0 args: %v", err)
	}
	// Valid: 1 arg
	if err := sslRenewCmd.Args(sslRenewCmd, []string{"example.com"}); err != nil {
		t.Errorf("unexpected error for 1 arg: %v", err)
	}
	// Invalid: 2 args
	if err := sslRenewCmd.Args(sslRenewCmd, []string{"a.com", "b.com"}); err == nil {
		t.Error("expected error for 2 args (MaximumNArgs(1))")
	}
}

// TestSSLStatusCmd_HasRunE verifies ssl status has a RunE handler.
func TestSSLStatusCmd_HasRunE(t *testing.T) {
	if sslStatusCmd.RunE == nil {
		t.Error("ssl status command has no RunE handler")
	}
}

// TestSSLRenewCmd_HasRunE verifies ssl renew has a RunE handler.
func TestSSLRenewCmd_HasRunE(t *testing.T) {
	if sslRenewCmd.RunE == nil {
		t.Error("ssl renew command has no RunE handler")
	}
}

// TestSSLCmd_Short verifies descriptive short text.
func TestSSLCmd_Short(t *testing.T) {
	if sslCmd.Short == "" {
		t.Error("ssl command has empty Short description")
	}
}
