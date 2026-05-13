package claw

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMigration creates a .sql file in dir with the given name and content.
func writeMigration(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
		t.Fatalf("writeMigration: %v", err)
	}
}

// TestScanClawMigrations verifies that scanClawMigrations returns only .sql files
// (not .down.sql) in lexicographic order.
func TestScanClawMigrations(t *testing.T) {
	dir := t.TempDir()

	writeMigration(t, dir, "001_init.sql", "-- init")
	writeMigration(t, dir, "003_add_index.sql", "-- index")
	writeMigration(t, dir, "002_add_topics.sql", "-- topics")
	writeMigration(t, dir, "002_add_topics.down.sql", "-- rollback topics") // must be excluded
	writeMigration(t, dir, "readme.txt", "not sql")                         // must be excluded

	names, err := scanClawMigrations(dir)
	if err != nil {
		t.Fatalf("scanClawMigrations: %v", err)
	}

	want := []string{"001_init.sql", "002_add_topics.sql", "003_add_index.sql"}
	if len(names) != len(want) {
		t.Fatalf("got %d migrations, want %d: %v", len(names), len(want), names)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, want[i])
		}
	}
}

// TestScanClawMigrationsEmpty verifies that an empty directory returns an empty
// slice (not an error).
func TestScanClawMigrationsEmpty(t *testing.T) {
	dir := t.TempDir()

	names, err := scanClawMigrations(dir)
	if err != nil {
		t.Fatalf("scanClawMigrations on empty dir: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty slice, got %v", names)
	}
}

// TestScanClawMigrationsNotExist verifies the error message when the directory
// does not exist.
func TestScanClawMigrationsNotExist(t *testing.T) {
	_, err := scanClawMigrations("/nonexistent/path/to/migrations")
	if err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
}

// TestMigrateOrderLogic exercises the filter/order logic of Migrate using a
// mock that records which SQL strings would have been applied, without a real DB.
//
// Strategy: stub scanClawMigrations by pre-populating a temp dir, then call
// the exported applyOrdered helper that accepts a pre-built list. Since Migrate
// itself calls ensureClawSchemaVersions (which needs a live DB), we test the
// lower-level filter via applyOrdered directly.
func TestApplyOrdered(t *testing.T) {
	tests := []struct {
		name        string
		all         []string
		applied     map[string]bool
		fromVersion string
		toVersion   string
		wantApplied []string
		wantSkipped []string
	}{
		{
			name:        "apply all pending",
			all:         []string{"001.sql", "002.sql", "003.sql"},
			applied:     map[string]bool{},
			fromVersion: "",
			toVersion:   "",
			wantApplied: []string{"001.sql", "002.sql", "003.sql"},
			wantSkipped: nil,
		},
		{
			name:        "skip already applied",
			all:         []string{"001.sql", "002.sql", "003.sql"},
			applied:     map[string]bool{"001.sql": true, "002.sql": true},
			fromVersion: "",
			toVersion:   "",
			wantApplied: []string{"003.sql"},
			wantSkipped: []string{"001.sql", "002.sql"},
		},
		{
			name:        "from version constraint",
			all:         []string{"001.sql", "002.sql", "003.sql"},
			applied:     map[string]bool{},
			fromVersion: "001.sql",
			toVersion:   "",
			wantApplied: []string{"002.sql", "003.sql"},
			wantSkipped: []string{"001.sql"},
		},
		{
			name:        "to version constraint",
			all:         []string{"001.sql", "002.sql", "003.sql"},
			applied:     map[string]bool{},
			fromVersion: "",
			toVersion:   "002.sql",
			wantApplied: []string{"001.sql", "002.sql"},
			wantSkipped: nil,
		},
		{
			name:        "from and to version constraint",
			all:         []string{"001.sql", "002.sql", "003.sql", "004.sql"},
			applied:     map[string]bool{},
			fromVersion: "001.sql",
			toVersion:   "003.sql",
			wantApplied: []string{"002.sql", "003.sql"},
			wantSkipped: []string{"001.sql"},
		},
		{
			name:        "all already applied",
			all:         []string{"001.sql", "002.sql"},
			applied:     map[string]bool{"001.sql": true, "002.sql": true},
			fromVersion: "",
			toVersion:   "",
			wantApplied: nil,
			wantSkipped: []string{"001.sql", "002.sql"},
		},
		{
			name:        "no migrations",
			all:         []string{},
			applied:     map[string]bool{},
			fromVersion: "",
			toVersion:   "",
			wantApplied: nil,
			wantSkipped: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			applied, skipped := applyOrdered(tc.all, tc.applied, tc.fromVersion, tc.toVersion)

			assertStringSlice(t, "applied", applied, tc.wantApplied)
			assertStringSlice(t, "skipped", skipped, tc.wantSkipped)
		})
	}
}

// TestMigrationDirection verifies that forward migrations are included and all
// rollback/down variants are excluded from the scan result.
//
// This is the regression test for the HG-12 data-loss bug where _rollback.sql
// and _down.sql (underscore) files were applied as forward migrations.
func TestMigrationDirection(t *testing.T) {
	dir := t.TempDir()

	// Forward migrations — must be included.
	writeMigration(t, dir, "001_init.sql", "CREATE TABLE np_claw_foo (id bigserial PRIMARY KEY);")
	writeMigration(t, dir, "002_add_col.sql", "ALTER TABLE np_claw_foo ADD COLUMN name TEXT;")

	// Rollback variants — must ALL be excluded regardless of naming convention.
	writeMigration(t, dir, "001_init.down.sql", "DROP TABLE IF EXISTS np_claw_foo;")
	writeMigration(t, dir, "002_add_col_rollback.sql", "ALTER TABLE np_claw_foo DROP COLUMN IF EXISTS name;")
	writeMigration(t, dir, "002_add_col_down.sql", "ALTER TABLE np_claw_foo DROP COLUMN IF EXISTS name;")
	writeMigration(t, dir, "000_rollback_all.sql", "DROP SCHEMA IF EXISTS np_claw CASCADE;")

	names, err := scanClawMigrations(dir)
	if err != nil {
		t.Fatalf("scanClawMigrations: %v", err)
	}

	want := []string{"001_init.sql", "002_add_col.sql"}
	if len(names) != len(want) {
		t.Fatalf("got %d migration(s) %v, want only forward migrations %v", len(names), names, want)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, want[i])
		}
	}

	// Verify isRollbackSQL explicitly for each convention.
	cases := []struct {
		file     string
		rollback bool
	}{
		{"001_init.sql", false},
		{"002_add_col.sql", false},
		{"001_init.down.sql", true},
		{"002_add_col_rollback.sql", true},
		{"002_add_col_down.sql", true},
		{"000_rollback_all.sql", true},
	}
	for _, c := range cases {
		got := isRollbackSQL(c.file)
		if got != c.rollback {
			t.Errorf("isRollbackSQL(%q) = %v, want %v", c.file, got, c.rollback)
		}
	}
}

// assertStringSlice compares two string slices, treating nil and empty as equal.
func assertStringSlice(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v (len=%d), want %v (len=%d)", label, got, len(got), want, len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q", label, i, got[i], want[i])
		}
	}
}
