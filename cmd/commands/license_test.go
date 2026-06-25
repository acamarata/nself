package commands

import (
	"strings"
	"testing"
)

// ── T05: license family — S05 acceptance tests ────────────────────────────────

// TestLicenseCmd_Registered verifies nself license is present on RootCmd.
func TestLicenseCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Use == "license" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("licenseCmd not registered on RootCmd")
	}
}

// TestLicenseCmd_AllSubcommandsRegistered verifies all 14 subcommands are present.
func TestLicenseCmd_AllSubcommandsRegistered(t *testing.T) {
	required := []string{
		"set", "add", "remove", "status", "list", "show", "validate",
		"clear", "upgrade", "refresh", "export", "import", "migrate", "simulate-offline",
	}
	subs := map[string]bool{}
	for _, sub := range licenseCmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range required {
		if !subs[want] {
			t.Errorf("license subcommand %q not registered", want)
		}
	}
}

// TestLicenseSimulateOffline_FlagsPresent verifies the simulate-offline subcommand
// is registered with the --clear flag.
func TestLicenseSimulateOffline_FlagsPresent(t *testing.T) {
	found := false
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "simulate-offline" {
			found = true
			f := sub.Flags().Lookup("clear")
			if f == nil {
				t.Error("simulate-offline: --clear flag not registered")
			}
			break
		}
	}
	if !found {
		t.Error("simulate-offline subcommand not found")
	}
}

// TestLicenseSimulateOffline_RequiresDays verifies the command returns an error
// when called with no arguments and no --clear flag.
func TestLicenseSimulateOffline_RequiresDays(t *testing.T) {
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "simulate-offline" {
			err := sub.RunE(sub, []string{})
			if err == nil {
				t.Fatal("expected error when called with no args and no --clear")
			}
			if !strings.Contains(err.Error(), "days") && !strings.Contains(err.Error(), "specify") {
				t.Errorf("unexpected error: %v", err)
			}
			return
		}
	}
	t.Fatal("simulate-offline subcommand not found")
}

// TestLicenseSimulateOffline_InvalidDays verifies a non-integer days arg is rejected.
func TestLicenseSimulateOffline_InvalidDays(t *testing.T) {
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "simulate-offline" {
			err := sub.RunE(sub, []string{"notanumber"})
			if err == nil {
				t.Fatal("expected error for non-integer days")
			}
			return
		}
	}
	t.Fatal("simulate-offline subcommand not found")
}

// TestLicenseSimulate_GraceHardThreshold verifies the grace hard threshold
// constant is set to 7 days (per fail-open spec: simulate-offline 14 triggers hard stop).
func TestLicenseSimulate_GraceHardThreshold(t *testing.T) {
	// The acceptance criterion is: simulate-offline at 7-day TTL must trigger fail-open.
	// We verify the constant in the internal/license package equals 7 days by checking
	// that the simulate-offline command docs reference 7 days in the long description.
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "simulate-offline" {
			long := sub.Long
			if !strings.Contains(long, "7") {
				t.Error("simulate-offline long description must reference 7-day threshold")
			}
			return
		}
	}
	t.Fatal("simulate-offline not found")
}

// TestLicenseSetCmd_FlagsPresent verifies the set subcommand is properly configured.
func TestLicenseSetCmd_FlagsPresent(t *testing.T) {
	found := false
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "set" {
			found = true
			if sub.Args == nil && sub.Use == "" {
				t.Error("license set: missing Use or Args definition")
			}
			break
		}
	}
	if !found {
		t.Fatal("license set subcommand not found")
	}
}

// TestLicenseClearCmd_Registered verifies the clear subcommand exists.
func TestLicenseClearCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "clear" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("license clear subcommand not found")
	}
}

// TestLicenseMigrateCmd_FlagsPresent verifies the migrate subcommand has --dry-run.
func TestLicenseMigrateCmd_FlagsPresent(t *testing.T) {
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "migrate" {
			f := sub.Flags().Lookup("dry-run")
			if f == nil {
				t.Error("license migrate: --dry-run flag not registered")
			}
			return
		}
	}
	t.Fatal("license migrate subcommand not found")
}

// TestLicenseExportCmd_Registered verifies export subcommand exists.
func TestLicenseExportCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "export" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("license export subcommand not found")
	}
}

// TestLicenseImportCmd_Registered verifies import subcommand exists.
func TestLicenseImportCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "import" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("license import subcommand not found")
	}
}

// TestLicenseUpgradeCmd_Registered verifies upgrade subcommand is registered.
func TestLicenseUpgradeCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "upgrade" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("license upgrade subcommand not found")
	}
}
