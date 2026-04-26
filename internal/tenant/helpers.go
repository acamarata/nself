package tenant

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// hashTenantID returns the first 8 hex characters of the SHA-256 of the raw
// tenant UUID. This is used in slog fields so that the full UUID (PII) never
// appears in structured logs while still providing a correlatable identifier.
func hashTenantID(id string) string {
	h := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", h[:4]) // 8 hex chars = 4 bytes
}

// sanitize escapes single quotes for safe SQL interpolation via psql -c.
// This is NOT a substitute for parameterized queries; callers must also
// validate inputs against a strict whitelist (see assertSafeValue) before
// interpolation. Kept as a belt-and-suspenders defence.
func sanitize(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// quoteIdent double-quotes a SQL identifier for safe DDL interpolation.
// Embedded double-quotes are escaped as "" per the SQL standard.
// Use for schema names, table names, column names, role names — any identifier
// that appears in an unparameterized DDL statement. SEC-SQL-01.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// trimOutput trims whitespace and newlines from command output.
func trimOutput(b []byte) string {
	return strings.TrimSpace(string(b))
}

// uuidRegex matches canonical UUIDs (8-4-4-4-12 hex).
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// slugRegex matches safe tenant slugs: letters, digits, underscores, hyphens.
var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// dateRegex matches YYYY-MM-DD dates only.
var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// monthRegex matches YYYY-MM month specifiers only.
var monthRegex = regexp.MustCompile(`^\d{4}-\d{2}$`)

// eventIDRegex matches stripe/idempotency-style event identifiers.
var eventIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,128}$`)

// reasonRegex matches a printable ASCII subset for suspend/free-text reasons.
// Quotes, backslashes, semicolons, and control characters are rejected so no
// escape combination can break out of the SQL literal.
var reasonRegex = regexp.MustCompile(`^[a-zA-Z0-9 ._,()\[\]+/@#!?:-]{0,256}$`)

// durationRegex matches values fed into parseDuration (digits + unit).
var durationRegex = regexp.MustCompile(`^\d{1,6}[smhd]$`)

// validateUUID returns an error if s is not a canonical UUID.
func validateUUID(s string) error {
	if !uuidRegex.MatchString(s) {
		return fmt.Errorf("invalid tenant id %q (expected UUID)", s)
	}
	return nil
}

// validateSlug returns an error if s is not a safe tenant slug.
func validateSlug(s string) error {
	if !slugRegex.MatchString(s) {
		return fmt.Errorf("invalid tenant slug %q", s)
	}
	return nil
}

// validateDate returns an error if s is not YYYY-MM-DD.
func validateDate(s string) error {
	if !dateRegex.MatchString(s) {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return nil
}

// validateMonth returns an error if s is not YYYY-MM.
func validateMonth(s string) error {
	if !monthRegex.MatchString(s) {
		return fmt.Errorf("invalid month %q (expected YYYY-MM)", s)
	}
	return nil
}

// validateEventID returns an error if s is not a safe event identifier.
func validateEventID(s string) error {
	if !eventIDRegex.MatchString(s) {
		return fmt.Errorf("invalid event id %q", s)
	}
	return nil
}

// validateReason returns an error if the free-text reason contains any
// character outside the safe set.
func validateReason(s string) error {
	if !reasonRegex.MatchString(s) {
		return fmt.Errorf("invalid reason (contains unsafe characters)")
	}
	return nil
}

// validateDuration returns an error if s is not a digits+unit duration.
func validateDuration(s string) error {
	if !durationRegex.MatchString(s) {
		return fmt.Errorf("invalid duration %q (expected e.g. 24h, 7d)", s)
	}
	return nil
}
