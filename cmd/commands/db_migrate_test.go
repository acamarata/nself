package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestDBMigrateCreate_ValidName verifies that a valid migration name creates
// both the up and down SQL files inside a "migrations" subdirectory.
func TestDBMigrateCreate_ValidName(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := runDBMigrateCreate(nil, []string{"my_migration"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("expected migrations directory to exist: %v", err)
	}

	var upFound, downFound bool
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, "_my_migration.sql") {
			upFound = true
		}
		if strings.HasSuffix(name, "_my_migration.down.sql") {
			downFound = true
		}
	}

	if !upFound {
		t.Error("expected an up migration file (*_my_migration.sql) but none found")
	}
	if !downFound {
		t.Error("expected a down migration file (*_my_migration.down.sql) but none found")
	}
}

// TestDBMigrateCreate_PathTraversal verifies that a name containing path
// traversal sequences is rejected.
func TestDBMigrateCreate_PathTraversal(t *testing.T) {
	if err := runDBMigrateCreate(nil, []string{"../../evil"}); err == nil {
		t.Fatal("expected an error for path traversal name, got nil")
	}
}

// TestDBMigrateCreate_SpaceInName verifies that a name containing a space
// is rejected.
func TestDBMigrateCreate_SpaceInName(t *testing.T) {
	if err := runDBMigrateCreate(nil, []string{"foo bar"}); err == nil {
		t.Fatal("expected an error for name with space, got nil")
	}
}

// TestDBMigrateCreate_EmptyName verifies that an empty name is rejected.
func TestDBMigrateCreate_EmptyName(t *testing.T) {
	if err := runDBMigrateCreate(nil, []string{""}); err == nil {
		t.Fatal("expected an error for empty name, got nil")
	}
}

// TestDBMigrateCreate_UppercaseName verifies that uppercase letters are rejected
// since the allowlist is restricted to [a-z0-9_-].
func TestDBMigrateCreate_UppercaseName(t *testing.T) {
	if err := runDBMigrateCreate(nil, []string{"MyMigration"}); err == nil {
		t.Fatal("expected an error for name with uppercase letters, got nil")
	}
}

// TestDBMigrateCreate_AllInvalidChars verifies that a name consisting entirely
// of disallowed characters (which would reduce to empty after sanitization) is
// rejected.
func TestDBMigrateCreate_AllInvalidChars(t *testing.T) {
	if err := runDBMigrateCreate(nil, []string{"@@@"}); err == nil {
		t.Fatal("expected an error for name with only invalid characters, got nil")
	}
}

// TestDBMigrateCreate_ValidWithHyphen verifies that hyphens are allowed in
// migration names and that both files are created.
func TestDBMigrateCreate_ValidWithHyphen(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := runDBMigrateCreate(nil, []string{"my-migration-v2"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("expected migrations directory to exist: %v", err)
	}

	var upFound, downFound bool
	for _, e := range entries {
		name := filepath.Base(e.Name())
		if strings.HasSuffix(name, "_my-migration-v2.sql") {
			upFound = true
		}
		if strings.HasSuffix(name, "_my-migration-v2.down.sql") {
			downFound = true
		}
	}

	if !upFound {
		t.Error("expected an up migration file (*_my-migration-v2.sql) but none found")
	}
	if !downFound {
		t.Error("expected a down migration file (*_my-migration-v2.down.sql) but none found")
	}
}

// newApplyCmd returns a minimal cobra.Command with the --file flag registered
// and a background context set, mirroring the real dbMigrateApplyCmd setup.
// Used for unit tests that need to exercise runDBMigrateApply without a live
// nSelf project.
func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "apply", RunE: runDBMigrateApply}
	cmd.Flags().String("file", "", "Path to the SQL migration file to apply")
	cmd.SetContext(context.Background())
	return cmd
}

// TestDBMigrateApply_FileNotFound verifies that runDBMigrateApply returns an
// error when --file points to a non-existent path. This exercises the early
// os.Stat guard, which fires before any project config or DB interaction (G-008
// error path).
func TestDBMigrateApply_FileNotFound(t *testing.T) {
	cmd := newApplyCmd()
	if err := cmd.Flags().Set("file", "/nonexistent/path/99999_no_such.sql"); err != nil {
		t.Fatalf("setting --file flag: %v", err)
	}
	err := runDBMigrateApply(cmd, nil)
	if err == nil {
		t.Fatal("expected an error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error message, got: %v", err)
	}
}

// TestDBMigrateApply_ExistingFile_ReachesConfigLoad verifies that when a valid
// SQL file exists, runDBMigrateApply passes the file-validation guard and
// proceeds to load project config (G-008 happy path up to config boundary).
// In a unit test environment without a .nself/ project directory the config
// load returns an error; that error confirms the file check was satisfied and
// control advanced past the os.Stat guard.
func TestDBMigrateApply_ExistingFile_ReachesConfigLoad(t *testing.T) {
	dir := t.TempDir()
	migFile := filepath.Join(dir, "20260101000000_create_users.sql")
	if err := os.WriteFile(migFile, []byte("CREATE TABLE users (id BIGSERIAL PRIMARY KEY);"), 0o600); err != nil {
		t.Fatalf("writing temp migration file: %v", err)
	}

	cmd := newApplyCmd()
	if err := cmd.Flags().Set("file", migFile); err != nil {
		t.Fatalf("setting --file flag: %v", err)
	}
	err := runDBMigrateApply(cmd, nil)
	// A config-load error means the file guard passed and we reached DB territory.
	// A "not found" error means the guard failed — that would be a regression.
	if err != nil && strings.Contains(err.Error(), "not found") {
		t.Errorf("file guard failed for a file that exists: %v", err)
	}
}
