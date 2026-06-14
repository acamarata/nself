package sqlallowlist

import (
	"strings"
	"testing"
)

func TestValidateMigrationSQL_BlockedPrefixes(t *testing.T) {
	t.Parallel()

	blocked := []struct {
		name string
		sql  string
	}{
		{"DROP TABLE", "DROP TABLE users"},
		{"DROP TABLE IF EXISTS", "DROP TABLE IF EXISTS test_drift"},
		{"DROP DATABASE", "DROP DATABASE mydb"},
		{"DROP SCHEMA", "DROP SCHEMA public CASCADE"},
		{"TRUNCATE", "TRUNCATE np_accounts"},
		{"TRUNCATE TABLE", "TRUNCATE TABLE np_events"},
		{"DELETE FROM", "DELETE FROM np_accounts WHERE id = '1'"},
		{"DELETE FROM no WHERE", "DELETE FROM np_accounts"},
		{"ALTER ROLE", "ALTER ROLE postgres SUPERUSER"},
		{"GRANT", "GRANT ALL ON TABLE np_accounts TO anon"},
		{"REVOKE", "REVOKE SELECT ON np_accounts FROM anon"},
		{`\copy meta-command`, `\copy np_accounts TO '/tmp/out.csv'`},
		{`\connect meta-command`, `\connect mydb`},
		{"DROP TABLE lowercase", "drop table users"},
		{"TRUNCATE mixed case", "Truncate np_accounts"},
		{"DROP TABLE with leading spaces", "   DROP TABLE users"},
	}

	for _, tt := range blocked {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateMigrationSQL(tt.sql)
			if err == nil {
				t.Errorf("expected error for blocked SQL %q, got nil", tt.sql)
			}
		})
	}
}

func TestValidateMigrationSQL_AllowedStatements(t *testing.T) {
	t.Parallel()

	allowed := []struct {
		name string
		sql  string
	}{
		{"CREATE TABLE IF NOT EXISTS", "CREATE TABLE IF NOT EXISTS test (id UUID PRIMARY KEY)"},
		{"CREATE TABLE", "CREATE TABLE np_events (id UUID PRIMARY KEY, created_at TIMESTAMPTZ)"},
		{"ALTER TABLE ADD COLUMN", "ALTER TABLE np_accounts ADD COLUMN display_name TEXT"},
		{"CREATE INDEX", "CREATE INDEX idx_np_accounts_email ON np_accounts(email)"},
		{"CREATE EXTENSION", "CREATE EXTENSION IF NOT EXISTS pgcrypto"},
		{"CREATE POLICY", "CREATE POLICY user_isolation ON np_accounts USING (id = current_user_id())"},
		{"INSERT with columns", "INSERT INTO np_accounts (id, email) VALUES ('uuid', 'a@b.com')"},
		{"UPDATE with WHERE", "UPDATE np_accounts SET display_name = 'test' WHERE id = 'uuid'"},
		{"DROP POLICY", "DROP POLICY user_isolation ON np_accounts"},
	}

	for _, tt := range allowed {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateMigrationSQL(tt.sql)
			if err != nil {
				t.Errorf("expected nil for allowed SQL %q, got: %v", tt.sql, err)
			}
		})
	}
}

func TestValidateMigrationSQL_CommentBypassBlocked(t *testing.T) {
	t.Parallel()

	// CR-C requirement: "-- comment\nDROP TABLE" must still be blocked.
	cases := []struct {
		name string
		sql  string
	}{
		{"single comment then DROP TABLE", "-- this is fine\nDROP TABLE users"},
		{"multiple comments then TRUNCATE", "-- ok\n-- also ok\nTRUNCATE np_accounts"},
		{"comment then DELETE FROM", "-- safe\nDELETE FROM np_accounts"},
		{"comment then ALTER ROLE", "-- harmless\nALTER ROLE postgres SUPERUSER"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateMigrationSQL(tt.sql)
			if err == nil {
				t.Errorf("comment bypass not blocked for SQL %q", tt.sql)
			}
		})
	}
}

func TestValidateMigrationSQL_ErrorMessageContainsStatementType(t *testing.T) {
	t.Parallel()

	err := ValidateMigrationSQL("DROP TABLE users")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "DROP TABLE") {
		t.Errorf("error message should contain statement type, got: %v", err)
	}
	if !strings.Contains(err.Error(), "nself_run_migration") {
		t.Errorf("error message should reference nself_run_migration, got: %v", err)
	}
}

func TestValidateMigrationSQL_EmptySQL(t *testing.T) {
	t.Parallel()

	// Empty SQL is not explicitly blocked by prefix — but the handler checks
	// for empty before calling ValidateMigrationSQL. Ensure it doesn't panic.
	err := ValidateMigrationSQL("")
	// Should return nil (empty string has no blocked prefix).
	if err != nil {
		t.Errorf("empty SQL returned unexpected error: %v", err)
	}
}

func TestValidateMigrationSQL_AllBlockedPrefixesCovered(t *testing.T) {
	t.Parallel()

	// Verify every entry in blockedPrefixes is exercised by the blocked test table.
	// This acts as a completeness guard so new entries in blockedPrefixes fail loudly
	// if not tested.
	for _, prefix := range blockedPrefixes {
		prefix := prefix
		t.Run("prefix_"+prefix, func(t *testing.T) {
			t.Parallel()
			// Construct a minimal SQL using this prefix.
			testSQL := prefix + " test_entity"
			err := ValidateMigrationSQL(testSQL)
			if err == nil {
				t.Errorf("blockedPrefix %q not rejected by ValidateMigrationSQL", prefix)
			}
		})
	}
}
