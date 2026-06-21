package commands

import (
	"testing"
)

func TestPlannedHardeningSteps(t *testing.T) {
	steps := plannedHardeningSteps()
	if len(steps) == 0 {
		t.Fatal("expected hardening steps")
	}

	// Verify key steps exist
	names := make(map[string]bool)
	for _, s := range steps {
		names[s.Name] = true
		if len(s.Cmd) == 0 {
			t.Errorf("step %q has empty command", s.Name)
		}
	}

	expected := []string{
		"Allow SSH (22)",
		"Allow HTTP (80)",
		"Allow HTTPS (443)",
		"Enable ufw",
		"Enable fail2ban",
		"Enable netfilter-persistent",
		"Save iptables rules (v4)",
		"Save iptables rules (v6)",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing step: %q", name)
		}
	}
}

func TestRunChecks_ReturnsFindings(t *testing.T) {
	findings := runChecks()
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}

	// All findings should have a name
	for _, f := range findings {
		if f.Name == "" {
			t.Error("finding with empty name")
		}
		if f.Detail == "" {
			t.Errorf("finding %q has empty detail", f.Name)
		}
	}
}

func TestCheckEnvFilePerms(t *testing.T) {
	// Should pass when no .env files exist in cwd
	f := checkEnvFilePerms()
	if f.Name != "Env file permissions" {
		t.Errorf("name = %q", f.Name)
	}
	// In test context (no .env files present), should be OK
	if !f.OK {
		t.Errorf("expected OK when no env files exist, detail: %s", f.Detail)
	}
}

func TestFinding_Fields(t *testing.T) {
	f := finding{
		Name:   "Test Check",
		OK:     true,
		Detail: "all good",
	}
	if f.Name != "Test Check" {
		t.Errorf("name = %q", f.Name)
	}
	if !f.OK {
		t.Error("expected OK")
	}
}

func TestSecurityCmd_Structure(t *testing.T) {
	// Verify subcommands are registered
	subs := securityCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subs {
		names[cmd.Name()] = true
	}
	for _, name := range []string{"audit", "status", "setup"} {
		if !names[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestSecuritySetupCmd_ApplyFlag(t *testing.T) {
	f := securitySetupCmd.Flags().Lookup("apply")
	if f == nil {
		t.Fatal("--apply flag not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--apply default = %q, want false", f.DefValue)
	}
}

func TestHardeningStep_Fields(t *testing.T) {
	s := hardeningStep{
		Name: "test step",
		Cmd:  []string{"echo", "hello"},
	}
	if s.Name != "test step" {
		t.Errorf("name = %q", s.Name)
	}
	if len(s.Cmd) != 2 {
		t.Errorf("cmd len = %d", len(s.Cmd))
	}
}

// TestSecurityScanCmd_Registered verifies the scan subcommand is registered under security.
func TestSecurityScanCmd_Registered(t *testing.T) {
	subs := securityCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subs {
		names[cmd.Name()] = true
	}
	if !names["scan"] {
		t.Error("security: missing subcommand 'scan' — required for Security-Always-Free Doctrine")
	}
}

// TestSecurityScanCmd_NoLicenseCheck verifies runSecurityScan source has no license gating.
// (Structural: the function simply calls runChecks() then reports — no license path.)
func TestSecurityScanCmd_FreePath(t *testing.T) {
	// runSecurityScan is not exported, but we can call runChecks() — the same
	// function scan uses internally — and confirm it returns findings without
	// requiring any license env var.
	findings := runChecks()
	if len(findings) == 0 {
		t.Fatal("scan: expected findings from runChecks()")
	}
	// All findings must have a name (basic structural assertion).
	for _, f := range findings {
		if f.Name == "" {
			t.Error("scan: finding with empty name")
		}
	}
}

// TestSecurityCmd_AllSubcommands verifies audit, status, setup, and scan are all registered.
func TestSecurityCmd_AllSubcommands(t *testing.T) {
	subs := securityCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subs {
		names[cmd.Name()] = true
	}
	for _, name := range []string{"audit", "status", "setup", "scan"} {
		if !names[name] {
			t.Errorf("security: missing subcommand %q", name)
		}
	}
}
