package commands

import (
	"testing"
)

// TestDNSSetupCmd_Structure verifies dns-setup flags are registered.
func TestDNSSetupCmd_Structure(t *testing.T) {
	for _, flag := range []string{"dry-run", "project"} {
		if f := dnsSetupCmd.Flags().Lookup(flag); f == nil {
			t.Errorf("dns-setup: missing flag --%s", flag)
		}
	}
}

// TestDNSSetupCmd_DryRunDefault verifies --dry-run defaults to false.
func TestDNSSetupCmd_DryRunDefault(t *testing.T) {
	f := dnsSetupCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Fatal("dns-setup: --dry-run flag not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", f.DefValue)
	}
}

// TestDNSSetupCmd_ProjectFlag verifies --project flag is registered with empty default.
func TestDNSSetupCmd_ProjectFlag(t *testing.T) {
	f := dnsSetupCmd.Flags().Lookup("project")
	if f == nil {
		t.Fatal("dns-setup: --project flag not registered")
	}
	if f.DefValue != "" {
		t.Errorf("--project default = %q, want empty", f.DefValue)
	}
}

// TestDNSSetupCmd_UseName verifies command Use field.
func TestDNSSetupCmd_UseName(t *testing.T) {
	if dnsSetupCmd.Use != "dns-setup" {
		t.Errorf("dns-setup Use = %q, want dns-setup", dnsSetupCmd.Use)
	}
}
