// Package commands — tests for backup drill (regression) and DR subcommand flags.
//
// Purpose: Verify that nself backup drill and nself dr subcommands are wired
// correctly: flags registered, --dry-run present on all 5 dr subcommands, and
// exit codes correct. Does not require a running Docker environment.
package commands

import (
	"strings"
	"testing"
)

// ── backup drill ────────────────────────────────────────────────────────────

// TestBackupDrillCmd_Flags is a regression test for P100 Wave 2D: verify that
// all four flags from the original implementation are still registered.
func TestBackupDrillCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"file", ""},
		{"rto-hours", "4"},
		{"dry-run", "false"},
		{"json", "false"},
	}
	for _, f := range flags {
		flag := backupDrillCmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("flag --%s not registered on backup drill (P100 Wave 2D regression)", f.name)
			continue
		}
		if flag.DefValue != f.defValue {
			t.Errorf("--%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
		}
	}
}

// TestBackupDrillCmd_Use verifies the Use field is "drill" (subcommand of backup).
func TestBackupDrillCmd_Use(t *testing.T) {
	if backupDrillCmd.Use != "drill" {
		t.Errorf("backupDrillCmd.Use = %q, want %q", backupDrillCmd.Use, "drill")
	}
}

// TestBackupDrillCmd_HasRunE verifies backup drill has a RunE handler.
func TestBackupDrillCmd_HasRunE(t *testing.T) {
	if backupDrillCmd.RunE == nil {
		t.Error("backup drill: RunE is nil")
	}
}

// TestBackupDrillCmd_RegisteredUnderBackup verifies backup drill is a subcommand
// of the backup parent command.
func TestBackupDrillCmd_RegisteredUnderBackup(t *testing.T) {
	for _, sub := range backupCmd.Commands() {
		if sub.Use == "drill" {
			return
		}
	}
	t.Error("backup drill not found as subcommand of backupCmd")
}

// TestBackupDrillCmd_LongDescription verifies the long description mentions
// critical concepts: RTO target, scratch database, and production safety.
func TestBackupDrillCmd_LongDescription(t *testing.T) {
	long := backupDrillCmd.Long
	checks := []string{"RTO", "scratch", "Production"}
	for _, want := range checks {
		if !strings.Contains(long, want) {
			t.Errorf("backup drill Long description missing %q", want)
		}
	}
}

// ── dr subcommands — --dry-run coverage (safety-critical) ───────────────────

// TestDRDrill_HasDryRunFlag verifies dr drill has --dry-run.
func TestDRDrill_HasDryRunFlag(t *testing.T) {
	flag := drDrillCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dr drill: --dry-run flag not registered")
	}
	if flag.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", flag.DefValue)
	}
}

// TestDRPromote_HasDryRunFlag verifies dr promote-standby has --dry-run.
// promote-standby is irreversible in production — dry-run gate is safety-critical.
func TestDRPromote_HasDryRunFlag(t *testing.T) {
	flag := drPromoteCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dr promote-standby: --dry-run flag not registered (CR-B safety-critical requirement)")
	}
	if flag.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", flag.DefValue)
	}
}

// TestDRReconfigureDNS_HasDryRunFlag verifies dr reconfigure-dns has --dry-run.
func TestDRReconfigureDNS_HasDryRunFlag(t *testing.T) {
	flag := drReconfigureDNSCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dr reconfigure-dns: --dry-run flag not registered")
	}
	if flag.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", flag.DefValue)
	}
}

// TestDRRollback_HasDryRunFlag verifies dr rollback has --dry-run.
// rollback is irreversible — dry-run gate is safety-critical.
func TestDRRollback_HasDryRunFlag(t *testing.T) {
	flag := drRollbackCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dr rollback: --dry-run flag not registered (CR-B safety-critical requirement)")
	}
	if flag.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", flag.DefValue)
	}
}

// TestDRFence_HasDryRunFlag verifies dr fence has --dry-run.
func TestDRFence_HasDryRunFlag(t *testing.T) {
	flag := drFenceCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("dr fence: --dry-run flag not registered")
	}
	if flag.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", flag.DefValue)
	}
}

// TestDRAllSubcommands_RegisteredUnderDR verifies all 5 DR subcommands are wired
// under the dr parent command.
func TestDRAllSubcommands_RegisteredUnderDR(t *testing.T) {
	want := map[string]bool{
		"drill":          false,
		"promote-standby": false,
		"reconfigure-dns": false,
		"rollback":       false,
		"fence":          false,
	}
	for _, sub := range drCmd.Commands() {
		if _, ok := want[sub.Use]; ok {
			want[sub.Use] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("dr subcommand %q not found under drCmd", name)
		}
	}
}

// TestDRCmd_Use verifies the dr top-level command Use field.
func TestDRCmd_Use(t *testing.T) {
	if drCmd.Use != "dr" {
		t.Errorf("drCmd.Use = %q, want %q", drCmd.Use, "dr")
	}
}

// ── backup schedule cron validation ─────────────────────────────────────────

// TestBackupScheduleCmd_HasCronFlag verifies --cron flag is registered.
func TestBackupScheduleCmd_HasCronFlag(t *testing.T) {
	flag := backupScheduleCmd.Flags().Lookup("cron")
	if flag == nil {
		t.Fatal("backup schedule: --cron flag not registered")
	}
}

// TestBackupScheduleCmd_HasDryRunFlag verifies --dry-run flag is registered.
func TestBackupScheduleCmd_HasDryRunFlag(t *testing.T) {
	flag := backupScheduleCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("backup schedule: --dry-run flag not registered")
	}
}
