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

// CheckMigrationIdempotency analyzes SQL content for idempotent patterns.
// Returns true if the migration appears safe to re-run.
// Idempotent indicators: IF NOT EXISTS, IF EXISTS, CREATE OR REPLACE
// Non-idempotent: bare CREATE TABLE, ALTER TABLE ... ADD COLUMN without IF NOT EXISTS
func CheckMigrationIdempotency(sqlContent string) (bool, []string) {
	upper := strings.ToUpper(sqlContent)

	// Non-idempotent patterns (case-insensitive).
	nonIdempotentPatterns := []struct {
		re      *regexp.Regexp
		message string
	}{
		{
			re:      regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(?!IF\s+NOT\s+EXISTS)`),
			message: "Non-idempotent: CREATE TABLE (missing IF NOT EXISTS)",
		},
		{
			re:      regexp.MustCompile(`(?i)\bCREATE\s+(?:UNIQUE\s+)?INDEX\s+(?!IF\s+NOT\s+EXISTS)(?!CONCURRENTLY\s+IF\s+NOT\s+EXISTS)`),
			message: "Non-idempotent: CREATE INDEX (missing IF NOT EXISTS)",
		},
		{
			re:      regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+\S+\s+ADD\s+COLUMN\s+(?!IF\s+NOT\s+EXISTS)`),
			message: "Non-idempotent: ALTER TABLE ... ADD COLUMN (missing IF NOT EXISTS)",
		},
		{
			re:      regexp.MustCompile(`(?i)\bDROP\s+TABLE\s+(?!IF\s+EXISTS)`),
			message: "Non-idempotent: DROP TABLE (missing IF EXISTS)",
		},
		{
			re:      regexp.MustCompile(`(?i)\bDROP\s+(?:INDEX|COLUMN)\s+(?!IF\s+EXISTS)`),
			message: "Non-idempotent: DROP INDEX/COLUMN (missing IF EXISTS)",
		},
	}

	// Check whether any idempotent patterns are present.
	hasIdempotentGuard := strings.Contains(upper, "IF NOT EXISTS") ||
		strings.Contains(upper, "IF EXISTS") ||
		strings.Contains(upper, "CREATE OR REPLACE")

	var issues []string
	for _, p := range nonIdempotentPatterns {
		if p.re.MatchString(sqlContent) {
			issues = append(issues, p.message)
		}
	}

	// Considered idempotent only if no non-idempotent patterns are found.
	idempotent := len(issues) == 0 || hasIdempotentGuard && len(issues) == 0
	if len(issues) == 0 {
		idempotent = true
	}

	return idempotent, issues
}

// GenerateIdempotentVersion takes migration SQL and attempts to convert it to
// an idempotent form by adding IF NOT EXISTS clauses where missing.
// Returns the converted SQL and a list of conversions made.
func GenerateIdempotentVersion(sqlContent string) (converted string, changes []string) {
	converted = sqlContent

	// CREATE TABLE -> CREATE TABLE IF NOT EXISTS
	reCreateTable := regexp.MustCompile(`(?i)(CREATE\s+TABLE\s+)(?i:IF\s+NOT\s+EXISTS\s+)?`)
	if reCreateTable.MatchString(converted) {
		newSQL := regexp.MustCompile(`(?i)(CREATE\s+TABLE\s+)(?!IF\s+NOT\s+EXISTS)`).
			ReplaceAllStringFunc(converted, func(m string) string {
				return regexp.MustCompile(`(?i)(CREATE\s+TABLE\s+)`).
					ReplaceAllString(m, "${1}IF NOT EXISTS ")
			})
		if newSQL != converted {
			converted = newSQL
			changes = append(changes, "Added IF NOT EXISTS to CREATE TABLE")
		}
	}

	// CREATE INDEX -> CREATE INDEX IF NOT EXISTS
	{
		re := regexp.MustCompile(`(?i)(CREATE\s+(?:UNIQUE\s+)?INDEX\s+)(?!IF\s+NOT\s+EXISTS)`)
		newSQL := re.ReplaceAllStringFunc(converted, func(m string) string {
			re2 := regexp.MustCompile(`(?i)(CREATE\s+(?:UNIQUE\s+)?INDEX\s+)`)
			return re2.ReplaceAllString(m, "${1}IF NOT EXISTS ")
		})
		if newSQL != converted {
			converted = newSQL
			changes = append(changes, "Added IF NOT EXISTS to CREATE INDEX")
		}
	}

	// DROP TABLE -> DROP TABLE IF EXISTS
	{
		re := regexp.MustCompile(`(?i)(DROP\s+TABLE\s+)(?!IF\s+EXISTS)`)
		newSQL := re.ReplaceAllStringFunc(converted, func(m string) string {
			re2 := regexp.MustCompile(`(?i)(DROP\s+TABLE\s+)`)
			return re2.ReplaceAllString(m, "${1}IF EXISTS ")
		})
		if newSQL != converted {
			converted = newSQL
			changes = append(changes, "Added IF EXISTS to DROP TABLE")
		}
	}

	// DROP INDEX -> DROP INDEX IF EXISTS
	{
		re := regexp.MustCompile(`(?i)(DROP\s+INDEX\s+)(?!IF\s+EXISTS)`)
		newSQL := re.ReplaceAllStringFunc(converted, func(m string) string {
			re2 := regexp.MustCompile(`(?i)(DROP\s+INDEX\s+)`)
			return re2.ReplaceAllString(m, "${1}IF EXISTS ")
		})
		if newSQL != converted {
			converted = newSQL
			changes = append(changes, "Added IF EXISTS to DROP INDEX")
		}
	}

	// ALTER TABLE ... ADD COLUMN -> ADD COLUMN IF NOT EXISTS (best effort)
	{
		re := regexp.MustCompile(`(?i)(ADD\s+COLUMN\s+)(?!IF\s+NOT\s+EXISTS)`)
		newSQL := re.ReplaceAllStringFunc(converted, func(m string) string {
			re2 := regexp.MustCompile(`(?i)(ADD\s+COLUMN\s+)`)
			return re2.ReplaceAllString(m, "${1}IF NOT EXISTS ")
		})
		if newSQL != converted {
			converted = newSQL
			changes = append(changes, "Added IF NOT EXISTS to ADD COLUMN")
		}
	}

	return converted, changes
}
