package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// MigrationAuditResult describes the audit result for a single migration.
type MigrationAuditResult struct {
	Name          string
	Applied       bool
	Idempotent    bool     // uses IF NOT EXISTS, IF EXISTS, CREATE OR REPLACE
	HasRollback   bool     // a down.sql exists alongside up.sql
	ChecksumMatch bool     // file checksum matches stored checksum in nself_ops.migrations
	Issues        []string // list of idempotency problems found
}

// AuditMigrations audits all migration files for idempotency and drift.
// It checks:
//   - All applied migrations have matching checksums (no file edits after apply)
//   - Pending migrations use idempotent SQL patterns (IF NOT EXISTS, etc.)
//   - Each migration has a corresponding down/rollback file
func AuditMigrations(ctx context.Context, cfg *config.Config) ([]MigrationAuditResult, error) {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return nil, fmt.Errorf("ensure schema_versions: %w", err)
	}
	if err := ensureMigrationsTable(ctx, cfg); err != nil {
		return nil, fmt.Errorf("ensure migrations table: %w", err)
	}

	dir := migrationsDir(cfg, "")
	files, err := scanMigrations(dir)
	if err != nil {
		return nil, fmt.Errorf("scan migrations: %w", err)
	}

	if err := upgradeLedger(ctx, cfg, files); err != nil {
		return nil, fmt.Errorf("upgrade migration ledger: %w", err)
	}

	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("check applied migrations: %w", err)
	}

	opsApplied, err := appliedMigrationsOps(ctx, cfg)
	if err != nil {
		// Non-fatal: ops table may not exist on older projects.
		opsApplied = make(map[string]MigrationRecord)
	}

	var results []MigrationAuditResult

	for _, f := range files {
		name := migrationKey(f)
		migID := extractMigrationID(f)

		result := MigrationAuditResult{
			Name:          name,
			ChecksumMatch: true, // optimistic; corrected below if applied
		}

		// Applied check via np_common.schema_versions.
		_, isApplied := applied[name]
		result.Applied = isApplied

		// Read SQL content for idempotency check.
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("cannot read file: %v", readErr))
			results = append(results, result)
			continue
		}

		idempotent, issues := CheckMigrationIdempotency(string(data))
		result.Idempotent = idempotent
		result.Issues = append(result.Issues, issues...)

		// Rollback check: look for down.sql alongside up.sql (nested layout)
		// or <name_without_.sql>.down.sql (flat layout).
		var downPath string
		if filepath.Base(f) == "up.sql" {
			downPath = filepath.Join(filepath.Dir(f), "down.sql")
		} else {
			downPath = strings.TrimSuffix(f, ".sql") + ".down.sql"
		}
		if _, statErr := os.Stat(downPath); statErr == nil {
			result.HasRollback = true
		}

		// Checksum drift check: compare on-disk checksum vs stored in nself_ops.migrations.
		if isApplied {
			if rec, ok := opsApplied[migID]; ok && rec.Checksum != "" {
				diskCS, csErr := checksumBytes(data)
				if csErr == nil && diskCS != rec.Checksum {
					result.ChecksumMatch = false
					result.Issues = append(result.Issues,
						fmt.Sprintf("checksum drift: file has %s, recorded %s", diskCS, rec.Checksum))
				}
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// idempotencyRule describes one non-idempotent SQL shape.
//
// Purpose: flag a statement that would fail on re-run because it lacks its
//
//	IF [NOT] EXISTS guard.
//
// Inputs:  trigger matches the statement head and captures the text that
//
//	follows it; guard is applied to that captured text.
//
// Outputs: a match with a guard that does NOT fire is reported as an issue.
// Constraints: Go's regexp is RE2 and has NO lookahead. These rules previously
//
//	used `(?!...)`, which regexp.MustCompile rejects — and because the
//	patterns were compiled inside the function body rather than at
//	package scope, every call to CheckMigrationIdempotency panicked at
//	runtime instead of failing to build. `nself db audit` therefore
//	crashed on every invocation, and no test covered it. Compiling at
//	package scope means a malformed pattern now panics at init, where
//	any test run catches it immediately.
type idempotencyRule struct {
	trigger *regexp.Regexp
	guard   *regexp.Regexp
	message string
}

var nonIdempotentRules = []idempotencyRule{
	{
		trigger: regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(.*)`),
		guard:   regexp.MustCompile(`(?i)^IF\s+NOT\s+EXISTS\b`),
		message: "Non-idempotent: CREATE TABLE (missing IF NOT EXISTS)",
	},
	{
		trigger: regexp.MustCompile(`(?i)\bCREATE\s+(?:UNIQUE\s+)?INDEX\s+(.*)`),
		guard:   regexp.MustCompile(`(?i)^(?:CONCURRENTLY\s+)?IF\s+NOT\s+EXISTS\b`),
		message: "Non-idempotent: CREATE INDEX (missing IF NOT EXISTS)",
	},
	{
		trigger: regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+\S+\s+ADD\s+COLUMN\s+(.*)`),
		guard:   regexp.MustCompile(`(?i)^IF\s+NOT\s+EXISTS\b`),
		message: "Non-idempotent: ALTER TABLE ... ADD COLUMN (missing IF NOT EXISTS)",
	},
	{
		trigger: regexp.MustCompile(`(?i)\bDROP\s+TABLE\s+(.*)`),
		guard:   regexp.MustCompile(`(?i)^IF\s+EXISTS\b`),
		message: "Non-idempotent: DROP TABLE (missing IF EXISTS)",
	},
	{
		trigger: regexp.MustCompile(`(?i)\bDROP\s+(?:INDEX|COLUMN)\s+(.*)`),
		guard:   regexp.MustCompile(`(?i)^IF\s+EXISTS\b`),
		message: "Non-idempotent: DROP INDEX/COLUMN (missing IF EXISTS)",
	},
}

// CheckMigrationIdempotency analyzes SQL content for idempotent patterns.
// Returns true if the migration appears safe to re-run.
// Idempotent indicators: IF NOT EXISTS, IF EXISTS, CREATE OR REPLACE
// Non-idempotent: bare CREATE TABLE, ALTER TABLE ... ADD COLUMN without IF NOT EXISTS
func CheckMigrationIdempotency(sqlContent string) (bool, []string) {
	var issues []string
	for _, rule := range nonIdempotentRules {
		for _, m := range rule.trigger.FindAllStringSubmatch(sqlContent, -1) {
			if !rule.guard.MatchString(strings.TrimLeft(m[1], " \t")) {
				issues = append(issues, rule.message)
				break
			}
		}
	}
	return len(issues) == 0, issues
}

// idempotencyRewrite describes one guard insertion performed by
// GenerateIdempotentVersion.
//
// Purpose: add a missing IF [NOT] EXISTS guard to a statement.
// Inputs:  head matches the statement keyword up to the point the guard would
//
//	go; guard tests the text that follows; insert is the guard text.
//
// Outputs: rewritten SQL plus a human-readable change description.
// Constraints: Go's regexp is RE2 and has NO lookahead. These rewrites
//
//	previously used `(?!...)` inside regexp.MustCompile calls in the
//	function body, so every call to GenerateIdempotentVersion panicked
//	at runtime rather than failing to build — the same defect already
//	fixed in CheckMigrationIdempotency, in the same file, which was
//	repaired without noticing this second set. Compiling at package
//	scope means a malformed pattern now panics at init, where any test
//	run catches it.
type idempotencyRewrite struct {
	head   *regexp.Regexp
	guard  *regexp.Regexp
	insert string
	change string
}

var idempotencyRewrites = []idempotencyRewrite{
	{
		head:   regexp.MustCompile(`(?i)CREATE\s+TABLE\s+`),
		guard:  regexp.MustCompile(`(?i)^IF\s+NOT\s+EXISTS\b`),
		insert: "IF NOT EXISTS ",
		change: "Added IF NOT EXISTS to CREATE TABLE",
	},
	{
		head:   regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+`),
		guard:  regexp.MustCompile(`(?i)^(?:CONCURRENTLY\s+)?IF\s+NOT\s+EXISTS\b`),
		insert: "IF NOT EXISTS ",
		change: "Added IF NOT EXISTS to CREATE INDEX",
	},
	{
		head:   regexp.MustCompile(`(?i)DROP\s+TABLE\s+`),
		guard:  regexp.MustCompile(`(?i)^IF\s+EXISTS\b`),
		insert: "IF EXISTS ",
		change: "Added IF EXISTS to DROP TABLE",
	},
	{
		head:   regexp.MustCompile(`(?i)DROP\s+INDEX\s+`),
		guard:  regexp.MustCompile(`(?i)^IF\s+EXISTS\b`),
		insert: "IF EXISTS ",
		change: "Added IF EXISTS to DROP INDEX",
	},
	{
		head:   regexp.MustCompile(`(?i)ADD\s+COLUMN\s+`),
		guard:  regexp.MustCompile(`(?i)^IF\s+NOT\s+EXISTS\b`),
		insert: "IF NOT EXISTS ",
		change: "Added IF NOT EXISTS to ADD COLUMN",
	},
}

// GenerateIdempotentVersion takes migration SQL and attempts to convert it to
// an idempotent form by adding IF NOT EXISTS clauses where missing.
// Returns the converted SQL and a list of conversions made.
func GenerateIdempotentVersion(sqlContent string) (converted string, changes []string) {
	converted = sqlContent
	for _, r := range idempotencyRewrites {
		newSQL := r.head.ReplaceAllStringFunc(converted, func(m string) string {
			// The head match ends just before whatever follows the statement
			// keyword. Only insert the guard when it is not already there.
			rest := converted[strings.Index(converted, m)+len(m):]
			if r.guard.MatchString(strings.TrimLeft(rest, " \t")) {
				return m
			}
			return m + r.insert
		})
		if newSQL != converted {
			converted = newSQL
			changes = append(changes, r.change)
		}
	}
	return converted, changes
}
