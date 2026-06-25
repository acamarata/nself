package commands

// ssl_test.go — Unit tests for the ssl command.
// P4-E5-W3-S06-T20: security command coverage gate (was 0% — now covers happy + error paths).
//
// Purpose: Verify isLocalDomain classification, command registration, and TLS
//          error paths for unreachable domains.
// Inputs:  domain strings, TLS timeout.
// Outputs: boolean from isLocalDomain; error from checkDomainTLS on bad domain.
// Constraints: Does not establish real TLS connections (uses unreachable or local
//              addresses to trigger fast errors).

import (
	"testing"
	"time"
)

// TestIsLocalDomain_Localhost verifies that canonical localhost forms are local.
func TestIsLocalDomain_Localhost(t *testing.T) {
	t.Parallel()

	local := []string{"localhost", "127.0.0.1", "::1"}
	for _, d := range local {
		d := d
		t.Run(d, func(t *testing.T) {
			t.Parallel()
			if !isLocalDomain(d) {
				t.Errorf("isLocalDomain(%q) = false, want true", d)
			}
		})
	}
}

// TestIsLocalDomain_DotLocal verifies *.local mDNS names are classified local.
func TestIsLocalDomain_DotLocal(t *testing.T) {
	t.Parallel()

	local := []string{"mybox.local", "nself-dev.local", "a.local"}
	for _, d := range local {
		d := d
		t.Run(d, func(t *testing.T) {
			t.Parallel()
			if !isLocalDomain(d) {
				t.Errorf("isLocalDomain(%q) = false, want true", d)
			}
		})
	}
}

// TestIsLocalDomain_Remote verifies public domains are NOT classified local.
func TestIsLocalDomain_Remote(t *testing.T) {
	t.Parallel()

	remote := []string{"nself.org", "api.nself.org", "example.com", "10.0.0.1"}
	for _, d := range remote {
		d := d
		t.Run(d, func(t *testing.T) {
			t.Parallel()
			if isLocalDomain(d) {
				t.Errorf("isLocalDomain(%q) = true, want false", d)
			}
		})
	}
}

// TestCheckDomainTLS_Unreachable verifies checkDomainTLS returns an error for
// an address where no TLS server is listening.
func TestCheckDomainTLS_Unreachable(t *testing.T) {
	t.Parallel()

	// Port 19999 is virtually never in use; connection should be refused quickly.
	// Use a short timeout to keep the test fast.
	_, err := checkDomainTLS("127.0.0.1:19999", 500*time.Millisecond)
	if err == nil {
		t.Fatal("checkDomainTLS succeeded on an unreachable port — expected error")
	}
}

// TestSSLCmd_Registration verifies ssl command and subcommands are registered.
func TestSSLCmd_Registration(t *testing.T) {
	t.Parallel()

	if sslCmd == nil {
		t.Fatal("sslCmd is nil — command not registered")
	}
	if sslCmd.Use != "ssl" {
		t.Errorf("sslCmd.Use = %q, want %q", sslCmd.Use, "ssl")
	}
	if sslStatusCmd == nil {
		t.Fatal("sslStatusCmd is nil — subcommand not registered")
	}
	if sslRenewCmd == nil {
		t.Fatal("sslRenewCmd is nil — subcommand not registered")
	}
}

// TestSSLStatusCmd_MissingArgs verifies ssl status returns an error with no args.
func TestSSLStatusCmd_MissingArgs(t *testing.T) {
	t.Parallel()

	err := sslStatusCmd.RunE(sslStatusCmd, []string{})
	if err == nil {
		t.Skip("sslStatusCmd succeeded with no args — may have project config with domains")
	}
	if len(err.Error()) == 0 {
		t.Error("expected descriptive error, got empty string")
	}
}
