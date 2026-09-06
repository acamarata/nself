package database

import "testing"

func TestLintUnguardedBulkCreateTables_FiresOnBulkUnguarded(t *testing.T) {
	sql := `
CREATE TABLE tasks (id uuid PRIMARY KEY);
CREATE TABLE projects (id uuid PRIMARY KEY);
CREATE TABLE labels (id uuid PRIMARY KEY);
`
	finding := LintUnguardedBulkCreateTables(sql)
	if finding == nil {
		t.Fatal("expected a finding for 3 unguarded CREATE TABLE statements")
	}
	if finding.Count != 3 {
		t.Errorf("Count = %d, want 3", finding.Count)
	}
	if finding.Rule != "bulk-unguarded-create-table" {
		t.Errorf("Rule = %q", finding.Rule)
	}
}

func TestLintUnguardedBulkCreateTables_OKWhenGuarded(t *testing.T) {
	sql := `
CREATE TABLE IF NOT EXISTS tasks (id uuid PRIMARY KEY);
CREATE TABLE IF NOT EXISTS projects (id uuid PRIMARY KEY);
CREATE TABLE IF NOT EXISTS labels (id uuid PRIMARY KEY);
`
	if finding := LintUnguardedBulkCreateTables(sql); finding != nil {
		t.Fatalf("expected no finding for guarded statements, got %+v", finding)
	}
}

func TestLintUnguardedBulkCreateTables_BelowThreshold(t *testing.T) {
	sql := `
CREATE TABLE tasks (id uuid PRIMARY KEY);
CREATE TABLE projects (id uuid PRIMARY KEY);
`
	if finding := LintUnguardedBulkCreateTables(sql); finding != nil {
		t.Fatalf("expected no finding below threshold, got %+v", finding)
	}
}

func TestLintUnguardedBulkCreateTables_MixedGuardedStillCounted(t *testing.T) {
	sql := `
CREATE TABLE IF NOT EXISTS tasks (id uuid PRIMARY KEY);
CREATE TABLE projects (id uuid PRIMARY KEY);
CREATE TABLE labels (id uuid PRIMARY KEY);
CREATE TABLE comments (id uuid PRIMARY KEY);
`
	finding := LintUnguardedBulkCreateTables(sql)
	if finding == nil {
		t.Fatal("expected a finding: 3 unguarded creates even with one guarded present")
	}
	if finding.Count != 3 {
		t.Errorf("Count = %d, want 3 (the guarded statement must not be counted)", finding.Count)
	}
}

func TestLintMigrationFile(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, tmp, "tasks_schema.sql", `
CREATE TABLE tasks (id uuid PRIMARY KEY);
CREATE TABLE projects (id uuid PRIMARY KEY);
CREATE TABLE labels (id uuid PRIMARY KEY);
`)

	finding, err := LintMigrationFile(tmp + "/tasks_schema.sql")
	if err != nil {
		t.Fatalf("LintMigrationFile: %v", err)
	}
	if finding == nil {
		t.Fatal("expected a finding for the tasks_schema-style fixture")
	}
}

func TestListMigrationFiles(t *testing.T) {
	tmp := t.TempDir()
	withCwd(t, tmp)
	writeFile(t, tmp, "migrations/20260101_a.sql", "CREATE TABLE a (id int);\n")
	writeFile(t, tmp, "migrations/20260102_b.sql", "CREATE TABLE b (id int);\n")

	files, err := ListMigrationFiles(nil, "migrations")
	if err != nil {
		t.Fatalf("ListMigrationFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
}
