package migrationai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Op is the type of schema field change detected.
type Op int

const (
	OpAdd    Op = iota // field added to a struct/interface/class
	OpRemove           // field removed from a struct/interface/class
)

// SchemaFieldDelta describes a single field-level schema change.
type SchemaFieldDelta struct {
	StructName string
	FieldName  string
	FieldType  string
	Tag        string // struct tag if available
	Op         Op
}

// DiffSummary returns a human-readable summary of the detected schema deltas.
func DiffSummary(deltas []SchemaFieldDelta) string {
	if len(deltas) == 0 {
		return "No schema changes detected."
	}
	var sb strings.Builder
	for _, d := range deltas {
		switch d.Op {
		case OpAdd:
			fmt.Fprintf(&sb, "+ %s.%s (%s)\n", d.StructName, d.FieldName, d.FieldType)
		case OpRemove:
			fmt.Fprintf(&sb, "- %s.%s (%s)\n", d.StructName, d.FieldName, d.FieldType)
		}
	}
	return sb.String()
}

// MigrationProposal is the result of an AI-assisted migration generation.
type MigrationProposal struct {
	// Slug is the sanitized natural-language description used in the filename.
	Slug string

	// SQL is the proposed migration SQL (additive only in default mode).
	SQL string

	// WarningSQL contains DROP or dangerous statements the AI suggested but that
	// the additive-only guard blocked. It is surfaced as a comment in the migration file.
	WarningSQL string

	// Confidence is a 0.0–1.0 value returned by the AI model.
	Confidence float64

	// AIModel is the model that generated the proposal.
	AIModel string

	// FilePath is the path to the staged migration file.
	FilePath string
}

// WriteMigrationFile stages the migration file at migrationsDir/YYYYMMDDHHMMSS_auto_<slug>.sql.
// It enforces the additive-only rule: any DROP, TRUNCATE, or DELETE statement in the SQL
// body is converted to a comment warning regardless of AI output.
//
// Returns the absolute path of the written file.
func WriteMigrationFile(migrationsDir, slug, sql, warningSQL string) (string, error) {
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		return "", fmt.Errorf("create migrations directory %s: %w", migrationsDir, err)
	}

	ts := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_auto_%s.sql", ts, sanitizeSlug(slug))
	path := filepath.Join(migrationsDir, filename)

	var sb strings.Builder
	sb.WriteString("-- nself migrate generate: auto-generated migration\n")
	sb.WriteString("-- Review carefully before applying. Run: nself db migrate up\n\n")

	// Additive-only safety gate: scan SQL for destructive statements.
	safeSQL, blocked := filterDestructive(sql)
	sb.WriteString(safeSQL)

	// Combine AI warnings + any blocked statements.
	allWarnings := strings.TrimSpace(warningSQL + "\n" + blocked)
	if allWarnings != "" {
		sb.WriteString("\n\n-- WARNING: The following destructive statements were proposed by the AI\n")
		sb.WriteString("-- and have been blocked by the additive-only guard. Review manually:\n")
		for _, line := range strings.Split(allWarnings, "\n") {
			if strings.TrimSpace(line) != "" {
				sb.WriteString("-- " + line + "\n")
			}
		}
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write migration file %s: %w", path, err)
	}
	return path, nil
}

// DeleteMigrationFile removes a staged migration file (user rejected the proposal).
// It is a no-op if the file does not exist.
func DeleteMigrationFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete migration file %s: %w", path, err)
	}
	return nil
}

// ExportedSanitizeSlug is a test-only export of the sanitizeSlug function.
// Use sanitizeSlug directly in production code.
var ExportedSanitizeSlug = sanitizeSlug

// sanitizeSlug converts a natural-language description to a safe filename slug.
// Only lowercase alphanumerics and underscores are allowed; everything else becomes _.
func sanitizeSlug(s string) string {
	var sb strings.Builder
	s = strings.ToLower(strings.TrimSpace(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	slug := sb.String()
	// Trim leading/trailing underscores and collapse runs.
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}
	slug = strings.Trim(slug, "_")
	if slug == "" {
		slug = "migration"
	}
	if len(slug) > 64 {
		slug = slug[:64]
	}
	return slug
}

// destructiveKeywords lists SQL statement prefixes that are never emitted in additive-only mode.
var destructiveKeywords = []string{
	"DROP ",
	"TRUNCATE ",
	"DELETE ",
	"ALTER TABLE", // only DROP COLUMN variant is blocked; ADD COLUMN is safe
}

// filterDestructive splits sql into safe and blocked sections. Any statement that
// starts with a destructive keyword is moved to the blocked output. ALTER TABLE is
// handled specially: only ALTER TABLE ... DROP ... lines are blocked.
func filterDestructive(sql string) (safe, blocked string) {
	var safeLines, blockedLines []string
	for _, stmt := range splitStatements(sql) {
		upper := strings.ToUpper(strings.TrimSpace(stmt))
		isBlocked := false
		for _, kw := range destructiveKeywords {
			if strings.HasPrefix(upper, kw) {
				// For ALTER TABLE, only block DROP COLUMN / DROP CONSTRAINT lines.
				if kw == "ALTER TABLE" {
					if strings.Contains(upper, "DROP ") {
						isBlocked = true
					}
					// ADD COLUMN is safe — fall through.
					break
				}
				isBlocked = true
				break
			}
		}
		if isBlocked {
			blockedLines = append(blockedLines, stmt)
		} else {
			safeLines = append(safeLines, stmt)
		}
	}
	return strings.Join(safeLines, "\n"), strings.Join(blockedLines, "\n")
}

// splitStatements splits a SQL text into individual statements at semicolons.
// Preserves leading/trailing whitespace per statement for comment friendliness.
func splitStatements(sql string) []string {
	var stmts []string
	for _, s := range strings.Split(sql, ";") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" && trimmed != "--" {
			stmts = append(stmts, trimmed+";")
		}
	}
	return stmts
}
