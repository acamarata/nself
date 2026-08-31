// Package tenant manages multi-tenant lifecycle operations for a self-hosted nSelf deployment.
//
// # SQL Injection Defense — SEC-SQL-02
//
// SEC-SQL-02 has two execution paths:
//
// Path A — direct database/sql connection (query functions):
//
//	QueryUsage, BillingReport, RetryStripeEvent, and Audit use openTenantDB to
//	obtain a *sql.DB connection to Postgres and execute queries with $1/$2…
//	positional-parameter binding via db.QueryContext/ExecContext. No value from
//	user input is ever interpolated into the SQL string on this path.
//
// Path B — docker exec psql (write/collection functions):
//
//	CollectUsage and upsertUsage drive Postgres via
//	`docker exec <container> psql -c <sql>` because they perform multi-statement
//	INSERTs that reference CURRENT_DATE and Prometheus results unavailable over a
//	direct connection from the host. Values on this path are guarded by a two-layer
//	defence:
//	  1. Strict whitelist validation (validate* functions) — primary defence.
//	  2. sanitize() — belt-and-suspenders quote-doubling for the docker exec path
//	     only. It is intentionally kept for Path B; it is NOT used on Path A.
//
// sanitize() is deprecated for SQL use. It exists only for the docker exec Path B
// functions that cannot migrate to direct connections. Do not add new callers.
//
// Static analysis gate: CI runs the sec-lint.sh SQL-injection rule which fails if
// fmt.Sprintf with SQL keywords appears outside of caller-validated whitelist paths.
// See .github/workflows/sec-sqli-gate.yml.
package tenant

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	// pgx stdlib registers the "pgx" driver for the tenant direct-connection path.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/nself-org/cli/internal/config"
)

// hashTenantID returns the first 8 hex characters of the SHA-256 of the raw
// tenant UUID. This is used in slog fields so that the full UUID (PII) never
// appears in structured logs while still providing a correlatable identifier.
func hashTenantID(id string) string {
	h := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", h[:4]) // 8 hex chars = 4 bytes
}

// openTenantDB opens a direct database/sql connection to the project Postgres
// instance using the connection URL from cfg. The caller must call db.Close().
// This is used by query functions (QueryUsage, BillingReport, RetryStripeEvent,
// Audit) so they can use $N parameterized queries instead of docker exec psql.
func openTenantDB(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	url := cfg.DatabaseURL()
	if url == "" {
		return nil, fmt.Errorf("database URL is empty — check POSTGRES_* env vars in your .env")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("open tenant db: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to tenant db: %w (is the postgres container running?)", err)
	}
	return db, nil
}

// sanitize escapes single quotes for the docker exec psql execution path (Path B).
//
// Deprecated: only use in CollectUsage/upsertUsage (docker exec path). Do NOT
// add new callers. All new query functions must use openTenantDB with $N params.
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
