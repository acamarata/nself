package tenant

import (
	"strings"
)

// sanitize escapes single quotes for safe SQL interpolation via psql -c.
// This is not a substitute for parameterized queries but sufficient for
// CLI-driven admin operations where input is controlled.
func sanitize(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// trimOutput trims whitespace and newlines from command output.
func trimOutput(b []byte) string {
	return strings.TrimSpace(string(b))
}
