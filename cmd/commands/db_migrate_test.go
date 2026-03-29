package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
