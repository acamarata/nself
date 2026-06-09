// Package database provides SQL identifier sanitization helpers to prevent
// SQL injection in contexts where identifiers (database, schema, table, column
// names) must be interpolated into SQL statements (positional parameters do
// not bind identifiers in PostgreSQL).
package database

import (
	"fmt"
	"regexp"
	"strings"
)

// identRegex matches PostgreSQL unquoted identifiers: starts with a letter or
// underscore, followed by letters, digits, or underscores. This is a
// conservative superset; stricter than Postgres allows but safe for all
// practical names used by the CLI (database, schema, migration IDs, etc.).
var identRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// maxIdentLen is the PostgreSQL identifier byte limit (NAMEDATALEN-1 = 63).
// Identifiers longer than this are silently truncated by Postgres, which can
// cause mismatches between the name used to create a table/column and the name
// used to query it. We reject them early instead of letting Postgres truncate.
const maxIdentLen = 63

// SanitizeIdentifier validates a SQL identifier and returns a double-quoted,
// escape-safe form suitable for direct interpolation into SQL. Returns an
// error if the input does not match the safe identifier pattern or exceeds
// the PostgreSQL identifier byte limit (63 bytes).
//
// Use this for database names, schema names, table names, column names, and
// any other SQL identifier that must appear in a statement string. NEVER
// interpolate a raw string into SQL without validation.
func SanitizeIdentifier(s string) (string, error) {
	if len(s) > maxIdentLen {
		return "", fmt.Errorf("SQL identifier too long (%d bytes, max %d): %q", len(s), maxIdentLen, s)
	}
	if !identRegex.MatchString(s) {
		return "", fmt.Errorf("invalid SQL identifier: %q", s)
	}
	// Double quote and escape any embedded quote (defensive; regex already
	// rejects strings containing quotes, but belt-and-suspenders).
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`, nil
}

// MustSanitizeIdentifier is SanitizeIdentifier that panics on invalid input.
// Only use for compile-time-constant identifiers where validation is a
// sanity check, not a user-input boundary.
func MustSanitizeIdentifier(s string) string {
	out, err := SanitizeIdentifier(s)
	if err != nil {
		panic(err)
	}
	return out
}
