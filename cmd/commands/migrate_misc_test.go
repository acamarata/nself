package commands

import (
	"testing"
)

// TestMigrateFirebaseCmdRegistered verifies the firebase subcommand is wired
// under the migrate parent command.
func TestMigrateFirebaseCmdRegistered(t *testing.T) {
	t.Parallel()
	found := false
	for _, sub := range migrateCmd.Commands() {
		if sub.Use == "firebase" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'firebase' subcommand under migrateCmd; not found")
	}
}

// TestMigrateFirebaseFlagsPresent verifies all documented flags are registered
// on the firebase subcommand.
func TestMigrateFirebaseFlagsPresent(t *testing.T) {
	t.Parallel()
	required := []string{"export-dir", "auth-export", "output-dir", "project-name", "dry-run"}
	for _, flag := range required {
		if migrateFirebaseCmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s on migrateFirebaseCmd; not found", flag)
		}
	}
}

// TestMigrateSupabaseCmdRegistered verifies the supabase subcommand is wired
// under the migrate parent command.
func TestMigrateSupabaseCmdRegistered(t *testing.T) {
	t.Parallel()
	found := false
	for _, sub := range migrateCmd.Commands() {
		if sub.Use == "supabase" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'supabase' subcommand under migrateCmd; not found")
	}
}

// TestMigrateSupabaseFlagsPresent verifies all documented flags are registered
// on the supabase subcommand.
func TestMigrateSupabaseFlagsPresent(t *testing.T) {
	t.Parallel()
	required := []string{"project-ref", "supabase-key", "out"}
	for _, flag := range required {
		if migrateSupabaseCmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag --%s on migrateSupabaseCmd; not found", flag)
		}
	}
}
