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
	// Core subcommands per ticket scope.
	for _, name := range []string{"status", "renew"} {
		if !names[name] {
			t.Errorf("ssl: missing subcommand %q", name)
		}
	}
}

// TestSSLCmd_HasSetupAndAdd verifies ssl setup and add subcommands (from ssl_setup.go).
func TestSSLCmd_HasSetupAndAdd(t *testing.T) {
	subs := sslCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subs {
		names[cmd.Name()] = true
	}
	for _, name := range []string{"setup", "add"} {
		if !names[name] {
			t.Errorf("ssl: missing subcommand %q", name)
		}
	}
}

// TestIsLocalDomain verifies .local and localhost detection.
func TestIsLocalDomain(t *testing.T) {
	cases := []struct {
		domain string
		want   bool
	}{
		{"localhost", true},
		{"ummat.local", true},
		{"api.ummat.local", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"nself.org", false},
		{"api.nself.org", false},
	}
	for _, c := range cases {
		got := isLocalDomain(c.domain)
		if got != c.want {
			t.Errorf("isLocalDomain(%q) = %v, want %v", c.domain, got, c.want)
		}
	}
}

// TestSSLRenewCmd_Args verifies ssl renew accepts zero or one domain argument.
func TestSSLRenewCmd_Args(t *testing.T) {
	if sslRenewCmd.Args == nil {
		t.Fatal("ssl renew: Args validator not set")
	}
	// MaximumNArgs(1) — should accept 0 and 1 args.
	if err := sslRenewCmd.Args(sslRenewCmd, []string{}); err != nil {
		t.Errorf("ssl renew: 0 args should be accepted, got %v", err)
	}
	if err := sslRenewCmd.Args(sslRenewCmd, []string{"example.com"}); err != nil {
		t.Errorf("ssl renew: 1 arg should be accepted, got %v", err)
	}
	if err := sslRenewCmd.Args(sslRenewCmd, []string{"a.com", "b.com"}); err == nil {
		t.Error("ssl renew: 2 args should be rejected")
	}
}

// TestSSLSetupCmd_Flags verifies ssl setup flags.
func TestSSLSetupCmd_Flags(t *testing.T) {
	for _, flag := range []string{"provider", "wildcard", "email", "staging", "install-cron"} {
		if f := sslSetupCmd.Flags().Lookup(flag); f == nil {
			t.Errorf("ssl setup: missing flag --%s", flag)
		}
	}
}
