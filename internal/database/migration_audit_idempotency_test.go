package database

// migration_audit_idempotency_test.go — Coverage for CheckMigrationIdempotency.
//
// Purpose: this function shipped with five regexes using `(?!...)` negative
//          lookahead, which Go's RE2 engine does not support. They were built
//          with regexp.MustCompile inside the function body, so every call
//          panicked at runtime — `nself db audit` crashed on every invocation.
//          Nothing caught it because no test called this function at all.
//          These tests exist so that never recurs.
// Inputs:  SQL strings, both guarded and unguarded.
// Outputs: assertions on the (idempotent, issues) pair.
// Constraints: pure string analysis — no database, no filesystem.

import (
	"strings"
	"testing"
)

// TestCheckMigrationIdempotency_DoesNotPanic is the direct regression guard for
// the shipped crash. Before the fix this call panicked with "invalid or
// unsupported Perl syntax: (?!" rather than returning.
func TestCheckMigrationIdempotency_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CheckMigrationIdempotency panicked: %v", r)
		}
	}()
	CheckMigrationIdempotency("CREATE TABLE users (id int);")
}

func TestCheckMigrationIdempotency_FlagsUnguardedStatements(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"bare create table", "CREATE TABLE users (id int);", "CREATE TABLE"},
		{"bare create index", "CREATE INDEX idx_users ON users(id);", "CREATE INDEX"},
		{"bare unique index", "CREATE UNIQUE INDEX idx_u ON users(email);", "CREATE INDEX"},
		{"bare add column", "ALTER TABLE users ADD COLUMN age int;", "ADD COLUMN"},
		{"bare drop table", "DROP TABLE users;", "DROP TABLE"},
		{"bare drop index", "DROP INDEX idx_users;", "DROP INDEX"},
	}
	for _, c := range cases {
		ok, issues := CheckMigrationIdempotency(c.sql)
		if ok {
			t.Errorf("%s: reported idempotent, want non-idempotent", c.name)
			continue
		}
		var found bool
		for _, is := range issues {
			if strings.Contains(is, c.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: issues %v, want one mentioning %q", c.name, issues, c.want)
		}
	}
}

func TestCheckMigrationIdempotency_AcceptsGuardedStatements(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"create table guarded", "CREATE TABLE IF NOT EXISTS users (id int);"},
		{"create index guarded", "CREATE INDEX IF NOT EXISTS idx_users ON users(id);"},
		{"index concurrently guarded", "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_u ON users(id);"},
		{"add column guarded", "ALTER TABLE users ADD COLUMN IF NOT EXISTS age int;"},
		{"drop table guarded", "DROP TABLE IF EXISTS users;"},
		{"drop index guarded", "DROP INDEX IF EXISTS idx_users;"},
		{"lowercase guarded", "create table if not exists users (id int);"},
		{"extra whitespace", "CREATE   TABLE\n  IF NOT EXISTS users (id int);"},
	}
	for _, c := range cases {
		ok, issues := CheckMigrationIdempotency(c.sql)
		if !ok {
			t.Errorf("%s: reported non-idempotent with issues %v, want idempotent", c.name, issues)
		}
	}
}

// TestCheckMigrationIdempotency_GuardedElsewhereStillFlagged pins the behavior
// that matters most in a real migration file: one guarded statement must not
// launder an unguarded one sitting beside it. The pre-fix code carried a
// `hasIdempotentGuard` flag computed over the WHOLE file, which would have done
// exactly that laundering had the regexes ever compiled.
func TestCheckMigrationIdempotency_GuardedElsewhereStillFlagged(t *testing.T) {
	sql := "CREATE TABLE IF NOT EXISTS a (id int);\nCREATE TABLE b (id int);"
	ok, issues := CheckMigrationIdempotency(sql)
	if ok {
		t.Fatal("a guarded statement laundered an unguarded one; want non-idempotent")
	}
	if len(issues) != 1 {
		t.Errorf("issues = %v, want exactly one", issues)
	}
}

// TestGenerateIdempotentVersion_DoesNotPanic covers the SECOND set of lookahead
// regexes in this file. The first fix (cli#317) repaired
// CheckMigrationIdempotency's five rules and missed the five in
// GenerateIdempotentVersion, which panicked identically — same file, same
// defect, same commit that claimed to fix it. staticcheck kept reporting SA1000
// here after that PR merged, which is how it was caught.
func TestGenerateIdempotentVersion_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GenerateIdempotentVersion panicked: %v", r)
		}
	}()
	GenerateIdempotentVersion("CREATE TABLE users (id int);")
}

func TestGenerateIdempotentVersion_AddsMissingGuards(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"create table", "CREATE TABLE users (id int);", "CREATE TABLE IF NOT EXISTS users (id int);"},
		{"create index", "CREATE INDEX i ON t(c);", "CREATE INDEX IF NOT EXISTS i ON t(c);"},
		{"unique index", "CREATE UNIQUE INDEX i ON t(c);", "CREATE UNIQUE INDEX IF NOT EXISTS i ON t(c);"},
		{"drop table", "DROP TABLE users;", "DROP TABLE IF EXISTS users;"},
		{"drop index", "DROP INDEX i;", "DROP INDEX IF EXISTS i;"},
		{"add column", "ALTER TABLE t ADD COLUMN c int;", "ALTER TABLE t ADD COLUMN IF NOT EXISTS c int;"},
	}
	for _, c := range cases {
		got, changes := GenerateIdempotentVersion(c.in)
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if len(changes) == 0 {
			t.Errorf("%s: no change recorded", c.name)
		}
	}
}

// TestGenerateIdempotentVersion_LeavesGuardedStatementsAlone is the one that
// matters for correctness: double-inserting a guard produces invalid SQL, and
// avoiding that is exactly what the lookahead was there to do.
func TestGenerateIdempotentVersion_LeavesGuardedStatementsAlone(t *testing.T) {
	for _, sql := range []string{
		"CREATE TABLE IF NOT EXISTS users (id int);",
		"CREATE INDEX IF NOT EXISTS i ON t(c);",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS i ON t(c);",
		"DROP TABLE IF EXISTS users;",
		"DROP INDEX IF EXISTS i;",
		"ALTER TABLE t ADD COLUMN IF NOT EXISTS c int;",
	} {
		got, changes := GenerateIdempotentVersion(sql)
		if got != sql {
			t.Errorf("rewrote an already-guarded statement:\n  in:  %s\n  out: %s", sql, got)
		}
		if len(changes) != 0 {
			t.Errorf("%s: reported changes %v on already-guarded SQL", sql, changes)
		}
	}
}
