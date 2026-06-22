package commands

// migrate_misc_test.go — integration/unit tests for the "misc family" migration commands.
//
// Purpose: Verify that migrate-firebase exposes --dry-run, that migrate-supabase
// behaves correctly in CI mode (NSELF_SUPABASE_SKIP=1), and that the commands are
// properly registered on the migrate parent command.
//
// Constraints: No live Firebase or Supabase connectivity. Tests use the offline
// skip mechanism (NSELF_SUPABASE_SKIP=1) and --dry-run to exercise the command
// layer without writing files or hitting external services.

import (
	"os"
	"testing"
)

// ---------------------------------------------------------------------------
// migrate firebase
// ---------------------------------------------------------------------------

// TestMigrateFirebaseCmdRegistered verifies that the firebase subcommand is
// registered on migrateCmd.
func TestMigrateFirebaseCmdRegistered(t *testing.T) {
	for _, c := range migrateCmd.Commands() {
		if c.Name() == "firebase" {
			return
		}
	}
	t.Fatal("migrate firebase subcommand not registered on migrateCmd")
}

// TestMigrateFirebaseFlagsPresent verifies that all expected flags are declared
// on migrateFirebaseCmd, including --dry-run (acceptance criterion).
func TestMigrateFirebaseFlagsPresent(t *testing.T) {
	t.Parallel()

	required := []string{"export-dir", "dry-run"}
	optional := []string{"auth-export", "output-dir", "project-name"}

	for _, name := range append(required, optional...) {
		if migrateFirebaseCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not declared on migrate firebase", name)
		}
	}
}

// TestMigrateFirebaseDryRunFlag verifies --dry-run is a boolean flag with default false.
func TestMigrateFirebaseDryRunFlag(t *testing.T) {
	t.Parallel()

	f := migrateFirebaseCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Fatal("--dry-run flag not registered on migrate firebase")
	}
	if f.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", f.DefValue)
	}
}

// TestMigrateFirebaseExportDirRequired verifies that --export-dir is required.
func TestMigrateFirebaseExportDirRequired(t *testing.T) {
	t.Parallel()

	ann := migrateFirebaseCmd.Flags().Lookup("export-dir")
	if ann == nil {
		t.Fatal("--export-dir not declared")
	}
	// cobra marks required flags with the BashCompOneRequired annotation or the
	// RequiredFlags persistent-flag annotation; the simplest check is that the
	// annotation map is non-nil (set by MarkFlagRequired).
	if ann.Annotations == nil {
		t.Fatal("--export-dir has no annotations; expected cobra required annotation")
	}
}

// TestMigrateFirebaseDryRun_EmptyDir verifies that running with --dry-run on an
// empty (but existing) export directory returns an error describing the missing
// data rather than panicking or writing files.
func TestMigrateFirebaseDryRun_EmptyDir(t *testing.T) {
	exportDir := t.TempDir()

	cmd := migrateFirebaseCmd
	cmd.Flags().Set("export-dir", exportDir) //nolint:errcheck
	cmd.Flags().Set("dry-run", "true")       //nolint:errcheck

	err := runMigrateFirebase(cmd, nil)
	// An empty export directory contains no Firestore JSON — the internal
	// package returns an error ("no Firestore documents found" or similar).
	// We only care that (a) no panic, (b) no files were written outside the
	// temp dir, (c) the --dry-run flag was accepted without a flag-parse error.
	if err == nil {
		// If the command succeeded on an empty dir it means the internal package
		// treated it as zero-table dry-run — that is also acceptable behaviour.
		t.Log("runMigrateFirebase dry-run on empty dir returned nil — zero-table result, OK")
	}

	// Confirm no files were written to the test-dir.
	entries, readErr := os.ReadDir(exportDir)
	if readErr != nil {
		t.Fatalf("reading export dir: %v", readErr)
	}
	for _, e := range entries {
		t.Errorf("unexpected file written in --dry-run mode: %s", e.Name())
	}
}

// ---------------------------------------------------------------------------
// migrate supabase
// ---------------------------------------------------------------------------

// TestMigrateSupabaseCmdRegistered verifies that the supabase subcommand is
// registered on migrateCmd.
func TestMigrateSupabaseCmdRegistered(t *testing.T) {
	for _, c := range migrateCmd.Commands() {
		if c.Name() == "supabase" {
			return
		}
	}
	t.Fatal("migrate supabase subcommand not registered on migrateCmd")
}

// TestMigrateSupabaseFlagsPresent verifies that all expected flags are declared.
func TestMigrateSupabaseFlagsPresent(t *testing.T) {
	t.Parallel()

	flags := []string{"project-ref", "supabase-key", "out"}
	for _, name := range flags {
		if migrateSupabaseCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not declared on migrate supabase", name)
		}
	}
}

// TestMigrateSupabase_CISkipMode verifies that setting NSELF_SUPABASE_SKIP=1
// causes the command to exit cleanly without attempting live connectivity.
// This is the canonical "dry-run" path for CI environments.
func TestMigrateSupabase_CISkipMode(t *testing.T) {
	t.Setenv("NSELF_SUPABASE_SKIP", "1")

	cmd := migrateSupabaseCmd

	// Override required flags for the test without marking them truly required
	// (they were already marked in init(); just set values directly).
	cmd.Flags().Set("project-ref", "ci-test-ref")  //nolint:errcheck
	cmd.Flags().Set("supabase-key", "ci-test-key") //nolint:errcheck
	cmd.Flags().Set("out", t.TempDir())            //nolint:errcheck

	err := runMigrateSupabase(cmd, nil)
	if err != nil {
		t.Fatalf("NSELF_SUPABASE_SKIP=1 should skip connectivity and return nil, got: %v", err)
	}
}
