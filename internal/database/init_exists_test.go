package database

// init_exists_test.go — Guards CREATE DATABASE being genuinely idempotent.
//
// Purpose: createDatabase checks for the database, then creates it. Those two
//   steps can see two different servers: the postgres image runs a TEMPORARY
//   server for its entrypoint scripts, then replaces it with the real one. The
//   check lands on the temporary server (absent), POSTGRES_DB has already
//   created it on the real server, and CREATE fails:
//
//     ERROR: database "testproject" already exists
//
//   That killed `nself start` at step 5 of the golden path on a clean run.
// Inputs:  errors as they surface from `docker exec ... psql`.
// Outputs: assertions on isDatabaseAlreadyExists.
// Constraints: pure predicate tests; no docker, no postgres, no network.

import (
	"errors"
	"fmt"
	"testing"
)

// TestIsDatabaseAlreadyExists_RealMessage uses the exact text from the failing
// run rather than a paraphrase, wrapped the way the call site wraps it.
func TestIsDatabaseAlreadyExists_RealMessage(t *testing.T) {
	t.Parallel()

	inner := errors.New(`psql exec failed: ERROR:  database "testproject" already exists: exit status 1`)
	if !isDatabaseAlreadyExists(inner) {
		t.Error("the real postgres message must be recognised — this is the exact string that broke the golden path")
	}

	// The call site wraps before this predicate sees it in some paths.
	if !isDatabaseAlreadyExists(fmt.Errorf("create database testproject: %w", inner)) {
		t.Error("must still match through a wrap")
	}
}

// TestIsDatabaseAlreadyExists_DoesNotSwallowRealFailures is the important half.
// A predicate that is too broad turns this fix into a database-init step that
// cannot fail, which is worse than the bug.
func TestIsDatabaseAlreadyExists_DoesNotSwallowRealFailures(t *testing.T) {
	t.Parallel()

	mustNotMatch := []string{
		`ERROR:  permission denied to create database`,
		`psql: error: connection to server failed: Connection refused`,
		`FATAL:  password authentication failed for user "postgres"`,
		`ERROR:  role "postgres" does not exist`,
		`FATAL:  the database system is starting up`,
		`ERROR:  relation "users" already exists`, // a TABLE, not a database
		`ERROR:  schema "auth" already exists`,    // a SCHEMA, not a database
		`Error: no such container: myproj_postgres`,
	}
	for _, m := range mustNotMatch {
		m := m
		t.Run(m[:min(len(m), 42)], func(t *testing.T) {
			t.Parallel()
			if isDatabaseAlreadyExists(errors.New(m)) {
				t.Errorf("must NOT be treated as success: %q", m)
			}
		})
	}
}

// TestIsDatabaseAlreadyExists_NilIsNotSuccess — nil means CREATE succeeded and
// is handled on the happy path; the predicate must not claim it.
func TestIsDatabaseAlreadyExists_NilIsNotSuccess(t *testing.T) {
	t.Parallel()

	if isDatabaseAlreadyExists(nil) {
		t.Error("nil error must not be reported as already-exists")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
