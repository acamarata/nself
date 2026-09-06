package database

import (
	"fmt"
	"os"
	"regexp"

	"github.com/nself-org/cli/internal/config"
)

// Purpose: detect the tasks_schema-style authoring mistake — a single
// migration file issuing many bare CREATE TABLE statements with no
// IF NOT EXISTS guard anywhere in the file. cli#386 traced staging's
// migration backlog to exactly this shape: fine against an empty database,
// but a hard failure (and a possible mixed apply/fail state) against one
// where some of the tables already exist.
// Inputs: raw migration SQL text, or a file/directory path.
// Outputs: nil when clean; a LintFinding describing the violation otherwise.
// Constraints: static, text-only — never touches a live database, so the
// same check runs inside `db migrate lint` and inside --detect.

// bulkUnguardedCreateTableThreshold is "many": one or two ungated CREATE
// TABLE statements in an otherwise-guarded file is a plausible oversight;
// three or more in a single file is the bulk-dump pattern this lint exists
// to catch and refuse.
const bulkUnguardedCreateTableThreshold = 3

// createTableAnyRe matches every CREATE TABLE occurrence, guarded or not.
// Go's RE2 regexp engine has no negative lookahead, so "not followed by IF
// NOT EXISTS" is checked separately (see countUnguardedCreateTables) against
// the text immediately after each match instead of being expressed in one
// pattern.
var createTableAnyRe = regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+`)

// ifNotExistsAtStartRe matches "IF NOT EXISTS" anchored at the start of the
// string it's tested against — used to check the text right after a
// CREATE TABLE match.
var ifNotExistsAtStartRe = regexp.MustCompile(`(?i)^IF\s+NOT\s+EXISTS\b`)

// countUnguardedCreateTables returns how many CREATE TABLE statements in
// sqlContent are NOT immediately followed by IF NOT EXISTS.
func countUnguardedCreateTables(sqlContent string) int {
	count := 0
	for _, loc := range createTableAnyRe.FindAllStringIndex(sqlContent, -1) {
		if !ifNotExistsAtStartRe.MatchString(sqlContent[loc[1]:]) {
			count++
		}
	}
	return count
}

// LintFinding describes one migration-file lint violation.
type LintFinding struct {
	Rule    string
	Message string
	Count   int
}

// LintUnguardedBulkCreateTables returns nil when sqlContent has fewer than
// bulkUnguardedCreateTableThreshold unguarded CREATE TABLE statements.
// Callers (the `db migrate lint` command, --detect) treat a non-nil finding
// as a refusal, never a warning to silently route around.
func LintUnguardedBulkCreateTables(sqlContent string) *LintFinding {
	count := countUnguardedCreateTables(sqlContent)
	if count < bulkUnguardedCreateTableThreshold {
		return nil
	}
	return &LintFinding{
		Rule: "bulk-unguarded-create-table",
		Message: fmt.Sprintf(
			"%d CREATE TABLE statement(s) without IF NOT EXISTS in one migration — "+
				"fails hard on a database where any of them already exists. "+
				"Add IF NOT EXISTS to each, or split into per-object migrations.",
			count),
		Count: count,
	}
}

// LintMigrationFile reads a single migration file and lints it.
func LintMigrationFile(path string) (*LintFinding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LintUnguardedBulkCreateTables(string(data)), nil
}

// ListMigrationFiles returns the sorted on-disk migration files for dir (or
// the auto-detected directory when dir is ""). Exported so commands that
// need the raw file list (lint) don't have to reach into the DB-backed
// status/apply paths.
func ListMigrationFiles(cfg *config.Config, dir string) ([]string, error) {
	if dir == "" {
		dir = migrationsDir(cfg, "")
	}
	return scanMigrations(dir)
}
