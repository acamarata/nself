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

// SanitizeIdentifier validates a SQL identifier and returns a double-quoted,
// escape-safe form suitable for direct interpolation into SQL. Returns an
// error if the input does not match the safe identifier pattern.
//
// Use this for database names, schema names, table names, column names, and
// any other SQL identifier that must appear in a statement string. NEVER
// interpolate a raw string into SQL without validation.
func SanitizeIdentifier(s string) (string, error) {
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
