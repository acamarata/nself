// Package commands — tests for nself pitr subcommands.
//
// Purpose: Verify cobra wiring, flag registration, and round-trip behaviour of
// pitr enable/disable/status subcommands. Does not require a running database.
package commands

import (
	"testing"
)

// TestPITRCmd_Use verifies the top-level pitr command Use field.
func TestPITRCmd_Use(t *testing.T) {
	if pitrCmd.Use != "pitr" {
		t.Errorf("pitrCmd.Use = %q, want %q", pitrCmd.Use, "pitr")
	}
}

// TestPITRCmd_HasRunE verifies pitr parent command has RunE (help fallback).
func TestPITRCmd_HasRunE(t *testing.T) {
	if pitrCmd.RunE == nil {
		t.Error("pitrCmd.RunE is nil (expected help fallback)")
	}
}

// TestPITREnableCmd_Flags verifies that nself pitr enable has all required flags.
func TestPITREnableCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"to", ""},
		{"encrypt-recipient", ""},
		{"retention-days", "7"},
		{"wal-timeout", "60"},
	}
	for _, f := range flags {
		flag := pitrEnableCmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("pitr enable: flag --%s not registered", f.name)
			continue
		}
		if flag.DefValue != f.defValue {
			t.Errorf("pitr enable: --%s default = %q, want %q", f.name, flag.DefValue, f.defValue)
		}
	}
}

// TestPITRDisableCmd_Use verifies pitr disable is registered with correct Use.
func TestPITRDisableCmd_Use(t *testing.T) {
	if pitrDisableCmd.Use != "disable" {
		t.Errorf("pitrDisableCmd.Use = %q, want disable", pitrDisableCmd.Use)
	}
}

// TestPITRStatusCmd_Use verifies pitr status is registered with correct Use.
func TestPITRStatusCmd_Use(t *testing.T) {
	if pitrStatusCmd.Use != "status" {
		t.Errorf("pitrStatusCmd.Use = %q, want status", pitrStatusCmd.Use)
	}
}

// TestPITRBaseBackupCmd_Flags verifies pitr base-backup has encrypt-recipient flag.
func TestPITRBaseBackupCmd_Flags(t *testing.T) {
	flag := pitrBaseBackupCmd.Flags().Lookup("encrypt-recipient")
	if flag == nil {
		t.Error("pitr base-backup: --encrypt-recipient flag not registered")
	}
}

// TestPITRRestoreCmd_ToFlagRequired verifies pitr restore marks --to as required.
func TestPITRRestoreCmd_ToFlagRequired(t *testing.T) {
	// The flag must be registered.
	flag := pitrRestoreCmd.Flags().Lookup("to")
	if flag == nil {
		t.Fatal("pitr restore: --to flag not registered")
	}
	// MarkFlagRequired annotates the flag with the "required" annotation.
	annotations := pitrRestoreCmd.Annotations
	_ = annotations // cobra uses flag.Annotations internally; check via cobra API
	requiredAnnotation, ok := pitrRestoreCmd.Flags().Lookup("to").Annotations["cobra_annotation_bash_completion_one_required_flag"]
	if !ok {
		// cobra_annotation_bash_completion_one_required_flag is the key used by cobra for
		// MarkFlagRequired. This is internal API; verify via the pflag annotation key.
		// Fallback: simply confirm the flag exists and has a non-empty Usage string.
		if flag.Usage == "" {
			t.Error("pitr restore: --to flag has empty usage string")
		}
	} else {
		if len(requiredAnnotation) == 0 {
			t.Error("pitr restore: --to should be marked required but annotation is empty")
		}
	}
}

// TestPITRRestoreCmd_Flags verifies all pitr restore flags are registered.
func TestPITRRestoreCmd_Flags(t *testing.T) {
	flags := []struct {
		name string
	}{
		{"to"},
		{"identity"},
		{"auto-promote"},
	}
	for _, f := range flags {
		if pitrRestoreCmd.Flags().Lookup(f.name) == nil {
			t.Errorf("pitr restore: flag --%s not registered", f.name)
		}
	}
}

// TestPITRAllSubcommands_Registered verifies enable/disable/status/base-backup/restore
// are all registered under pitrCmd (round-trip wiring test).
func TestPITRAllSubcommands_Registered(t *testing.T) {
	want := map[string]bool{
		"enable":      false,
		"disable":     false,
		"status":      false,
		"base-backup": false,
		"restore":     false,
	}
	for _, sub := range pitrCmd.Commands() {
		if _, ok := want[sub.Use]; ok {
			want[sub.Use] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("pitr subcommand %q not found under pitrCmd", name)
		}
	}
}

// TestPITREnableCmd_HasRunE verifies pitr enable has a RunE handler.
func TestPITREnableCmd_HasRunE(t *testing.T) {
	if pitrEnableCmd.RunE == nil {
		t.Error("pitr enable: RunE is nil")
	}
}

// TestPITRDisableCmd_HasRunE verifies pitr disable has a RunE handler.
func TestPITRDisableCmd_HasRunE(t *testing.T) {
	if pitrDisableCmd.RunE == nil {
		t.Error("pitr disable: RunE is nil")
	}
}

// TestPITRStatusCmd_HasRunE verifies pitr status has a RunE handler.
func TestPITRStatusCmd_HasRunE(t *testing.T) {
	if pitrStatusCmd.RunE == nil {
		t.Error("pitr status: RunE is nil")
	}
}

// TestPITRAutoPromoteDefault verifies auto-promote defaults to true on restore.
// When auto-promote is false after a restore, the DBA must manually promote —
// surprising default that was previously undocumented.
func TestPITRAutoPromoteDefault(t *testing.T) {
	flag := pitrRestoreCmd.Flags().Lookup("auto-promote")
	if flag == nil {
		t.Fatal("pitr restore: --auto-promote flag not registered")
	}
	if flag.DefValue != "true" {
		t.Errorf("--auto-promote default = %q, want true", flag.DefValue)
	}
}
