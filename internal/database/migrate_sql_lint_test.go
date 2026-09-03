package database

import (
	"strings"
	"testing"
)

func TestValidateMigrationSQL_RejectsUnsupportedIfNotExists(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"add constraint", "ALTER TABLE np_users ADD CONSTRAINT IF NOT EXISTS uq_email UNIQUE (email);", "ADD CONSTRAINT IF NOT EXISTS"},
		{"create policy", "CREATE POLICY IF NOT EXISTS tenant_isolation ON np_todos USING (true);", "CREATE POLICY IF NOT EXISTS"},
		{"lowercase", "alter table t add constraint if not exists c check (x > 0);", "ADD CONSTRAINT IF NOT EXISTS"},
		{"split over lines", "ALTER TABLE t\n  ADD CONSTRAINT\n  IF NOT EXISTS c CHECK (x);", "ADD CONSTRAINT IF NOT EXISTS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMigrationSQL("001_x.sql", tc.sql)
			if err == nil {
				t.Fatalf("expected %s to be rejected", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error should name the construct %q, got: %v", tc.want, err)
			}
		})
	}
}

// The valid IF NOT EXISTS forms are common and must never be flagged. A lint
// that fires on correct SQL gets worked around rather than fixed.
func TestValidateMigrationSQL_AllowsValidIfNotExists(t *testing.T) {
	valid := []string{
		"ALTER TABLE np_users ADD COLUMN IF NOT EXISTS nickname text;",
		"CREATE TABLE IF NOT EXISTS np_todos (id uuid PRIMARY KEY);",
		"CREATE SCHEMA IF NOT EXISTS nself_ops;",
		"CREATE INDEX IF NOT EXISTS idx_todos_user ON np_todos (user_id);",
		"DROP POLICY IF EXISTS tenant_isolation ON np_todos;",
		"ALTER TABLE t ADD CONSTRAINT uq UNIQUE (a);",
		"CREATE POLICY p ON t USING (true);",
	}
	for _, sql := range valid {
		if err := ValidateMigrationSQL("002_x.sql", sql); err != nil {
			t.Errorf("valid SQL was rejected:\n  %s\n  %v", sql, err)
		}
	}
}

// A migration is entitled to document the pitfall without tripping the check.
func TestValidateMigrationSQL_IgnoresComments(t *testing.T) {
	sql := `-- Do not write ADD CONSTRAINT IF NOT EXISTS here: Postgres rejects it.
/* CREATE POLICY IF NOT EXISTS is likewise invalid. */
ALTER TABLE t ADD COLUMN IF NOT EXISTS x text;`
	if err := ValidateMigrationSQL("003_x.sql", sql); err != nil {
		t.Fatalf("commented-out mentions must not be flagged: %v", err)
	}
}

// Line numbers must survive comment stripping, or the report points at the
// wrong place in a long migration.
func TestValidateMigrationSQL_ReportsCorrectLine(t *testing.T) {
	sql := "-- header\n\n/* block\n   spanning lines */\nALTER TABLE t ADD CONSTRAINT IF NOT EXISTS c CHECK (x);"
	err := ValidateMigrationSQL("004_x.sql", sql)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if !strings.Contains(err.Error(), "004_x.sql:5") {
		t.Fatalf("expected the offence on line 5, got: %v", err)
	}
}

func TestValidateMigrationSQL_ReportsEveryOccurrence(t *testing.T) {
	sql := "ALTER TABLE a ADD CONSTRAINT IF NOT EXISTS c1 CHECK (x);\nCREATE POLICY IF NOT EXISTS p ON b USING (true);"
	err := ValidateMigrationSQL("005_x.sql", sql)
	if err == nil {
		t.Fatal("expected rejection")
	}
	for _, want := range []string{"ADD CONSTRAINT IF NOT EXISTS", "CREATE POLICY IF NOT EXISTS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("both problems should be reported; missing %q in: %v", want, err)
		}
	}
}
