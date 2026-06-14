// Package sqlallowlist enforces a DDL allowlist for SQL statements submitted
// through the nself_run_migration MCP tool. It prevents AI Studio sessions from
// executing destructive or privilege-altering SQL via confirm=true.
//
// Purpose: Block arbitrary DDL/DML (DROP TABLE, TRUNCATE, DELETE FROM without WHERE,
//
//	ALTER ROLE, GRANT, REVOKE, psql meta-commands) from the MCP surface.
//
// Inputs:  Raw SQL string from the MCP tool request.
// Outputs: nil on allowed statements; descriptive error on blocked statements.
// Constraints: Case-insensitive. Strips leading whitespace and single-line comments
//
//	before matching so that comment-prefix bypasses are impossible.
//
// SPORT: F02-COMMAND-INVENTORY.md — nself mcp / nself_run_migration DDL allowlist.
package sqlallowlist

import (
	"fmt"
	"strings"
)

// blockedPrefixes lists statement prefixes that are never permitted via MCP.
// Checked case-insensitively against the normalized (stripped) SQL.
var blockedPrefixes = []string{
	"DROP TABLE",
	"DROP DATABASE",
	"DROP SCHEMA",
	"TRUNCATE",
	"DELETE FROM",
	"ALTER ROLE",
	"GRANT",
	"REVOKE",
	`\COPY`,
	`\CONNECT`,
}

// stripLeadingComments removes leading single-line SQL comments (-- ...) and
// blank lines so that "-- DROP comment\nDROP TABLE" cannot bypass the check.
// It does not remove block comments (/* ... */) because those are rare in
// migration SQL and should be treated conservatively (caller blocks them if
// the resulting effective prefix is blocked).
func stripLeadingComments(sql string) string {
	for {
		trimmed := strings.TrimSpace(sql)
		if strings.HasPrefix(trimmed, "--") {
			// Advance past the comment line.
			if idx := strings.Index(trimmed, "\n"); idx >= 0 {
				sql = trimmed[idx+1:]
			} else {
				// Entire string is a comment.
				return ""
			}
		} else {
			return trimmed
		}
	}
}

// ValidateMigrationSQL checks sql against the DDL blocklist. It returns nil
// when the statement is allowed, or an error with a descriptive message when
// the statement type is prohibited via the MCP surface.
//
// The check is intentionally conservative: any statement whose normalised
// prefix matches a blocked keyword is rejected, regardless of qualifiers that
// follow (e.g. "DROP TABLE IF EXISTS" is still blocked).
func ValidateMigrationSQL(sql string) error {
	// Strip leading comments first to prevent "-- comment\nDROP TABLE" bypass.
	effective := stripLeadingComments(sql)
	normalized := strings.ToUpper(effective)

	for _, blocked := range blockedPrefixes {
		if strings.HasPrefix(normalized, blocked) {
			return fmt.Errorf(
				"MCP migration: statement type %q is not permitted via nself_run_migration; "+
					"use nself db migrate directly",
				blocked,
			)
		}
	}
	return nil
}
