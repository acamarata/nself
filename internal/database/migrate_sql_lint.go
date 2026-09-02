package database

import (
	"fmt"
	"regexp"
	"strings"
)

// Postgres supports IF NOT EXISTS on some CREATE/ALTER forms and not others.
// The two below read as though they should work, are accepted by no Postgres
// version, and fail at apply time rather than at review time:
//
//	ALTER TABLE t ADD CONSTRAINT IF NOT EXISTS c ...   -- not valid
//	CREATE POLICY IF NOT EXISTS p ON t ...             -- not valid
//
// Both are natural things to write when making a migration re-runnable, which
// is exactly why they get written. ADD COLUMN IF NOT EXISTS, CREATE TABLE /
// SCHEMA / INDEX IF NOT EXISTS are all valid and must not be flagged.
//
// Reported by msg-2026-07-02-nself-migration-ledger-pk-bug.md as a secondary
// finding: these passed nself's migration handling and reached the database.
var unsupportedIfNotExists = []struct {
	pattern *regexp.Regexp
	what    string
	advice  string
}{
	{
		regexp.MustCompile(`(?i)\bADD\s+CONSTRAINT\s+IF\s+NOT\s+EXISTS\b`),
		"ADD CONSTRAINT IF NOT EXISTS",
		"Postgres has no IF NOT EXISTS for ADD CONSTRAINT. Guard it with a DO block that checks pg_constraint, or drop the constraint first.",
	},
	{
		regexp.MustCompile(`(?i)\bCREATE\s+POLICY\s+IF\s+NOT\s+EXISTS\b`),
		"CREATE POLICY IF NOT EXISTS",
		"Postgres has no IF NOT EXISTS for CREATE POLICY. Use DROP POLICY IF EXISTS followed by CREATE POLICY.",
	},
}

var (
	lineCommentRe  = regexp.MustCompile(`--[^\n]*`)
	blockCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// stripSQLComments blanks out comment bodies while preserving newlines, so
// line numbers still line up with the original file. Without this, a migration
// that documents the pitfall in a comment would be reported as containing it.
func stripSQLComments(sql string) string {
	blank := func(m string) string {
		return strings.Repeat(" ", len(m)-strings.Count(m, "\n")) + strings.Repeat("\n", strings.Count(m, "\n"))
	}
	sql = blockCommentRe.ReplaceAllStringFunc(sql, blank)
	sql = lineCommentRe.ReplaceAllStringFunc(sql, blank)
	return sql
}

// ValidateMigrationSQL reports constructs that no Postgres version accepts, so
// they surface before the migration is applied rather than as a syntax error
// partway through a batch.
//
// It is deliberately narrow. It flags only forms that are invalid in every
// supported Postgres, never merely unusual ones: a lint that cries wolf on
// valid SQL would be worked around rather than fixed.
func ValidateMigrationSQL(name, sql string) error {
	stripped := stripSQLComments(sql)
	var problems []string

	for _, rule := range unsupportedIfNotExists {
		for _, loc := range rule.pattern.FindAllStringIndex(stripped, -1) {
			line := 1 + strings.Count(stripped[:loc[0]], "\n")
			problems = append(problems,
				fmt.Sprintf("  %s:%d: %s is not valid Postgres.\n    %s",
					name, line, rule.what, rule.advice))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("migration %s contains SQL Postgres will reject:\n%s",
		name, strings.Join(problems, "\n"))
}
