package config

import "strings"

// isJSONValue reports whether v is a JSON object or array value (starts with
// '{' or '['). These values must be quoted when written to .env files because
// the equals sign, braces, and inner quotes would otherwise be misinterpreted
// by naive line-by-line readers that do not implement the dotenv quoting spec.
func isJSONValue(v string) bool {
	s := strings.TrimSpace(v)
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}

// QuoteEnvValue wraps v in single quotes when v is a JSON object or array
// value. Non-JSON values are returned unchanged. Single-quoted values are also
// returned unchanged (idempotent).
//
// Example:
//
//	QuoteEnvValue(`{"type":"HS256","key":"abc"}`) → `'{"type":"HS256","key":"abc"}'`
//	QuoteEnvValue("plain-value")                  → "plain-value"
//	QuoteEnvValue(`'already-quoted'`)             → `'already-quoted'`
func QuoteEnvValue(v string) string {
	if isSingleQuoted(v) {
		return v
	}
	if isJSONValue(v) {
		return "'" + v + "'"
	}
	return v
}

// UnquoteEnvValue strips a single layer of wrapping single quotes from v when
// the entire value is single-quoted. It is the inverse of QuoteEnvValue.
//
// Example:
//
//	UnquoteEnvValue(`'{"type":"HS256","key":"abc"}'`) → `{"type":"HS256","key":"abc"}`
//	UnquoteEnvValue("plain-value")                    → "plain-value"
func UnquoteEnvValue(v string) string {
	if isSingleQuoted(v) {
		return v[1 : len(v)-1]
	}
	return v
}

// isSingleQuoted reports whether v is wrapped in a matching pair of single
// quotes with at least one character between them.
func isSingleQuoted(v string) bool {
	return len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\''
}
