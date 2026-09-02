package database

import (
	"context"
	"fmt"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// Purpose: dry-run SQL parse/plan validation for a single migration file,
// run BEFORE the real (irreversible) apply. Closes the secondary finding
// from msg-2026-07-02-nself-migration-ledger-pk-bug.md: nself's own
// migration lint never parsed SQL against Postgres itself, so invalid
// constructs Postgres rejects at parse/plan time — most notably
// `ADD CONSTRAINT IF NOT EXISTS` and `CREATE POLICY IF NOT EXISTS` (neither
// DDL form supports IF NOT EXISTS in any Postgres version) — passed
// nself's lint silently and only surfaced during the real apply, mid-batch.
// Inputs: a context, the resolved *config.Config, and the migration's raw
// SQL text.
// Outputs: nil if every statement parses/plans cleanly; a wrapped
// errs.ErrMigrationValidationFailed naming the real Postgres error text
// otherwise.
// Constraints: runs the *exact* statements that will be applied for real,
// wrapped in BEGIN; ... ROLLBACK; against the live target (container or
// embedded-PG per pipeSQLToContainer's own dispatch) — this is real Postgres
// parse/plan/constraint validation, not a regex heuristic, so it never
// false-negatives on invalid syntax classes this file's authors didn't
// anticipate. Only called for transactional migrations (see migrate.go): a
// CREATE INDEX CONCURRENTLY (or other isNonTransactional) migration cannot
// run inside BEGIN/ROLLBACK at all — Postgres forbids CONCURRENTLY inside a
// transaction block — so wrapping one here would be a false positive
// unrelated to the SQL's own validity, not a real finding.

// wrapDryRunSQL wraps migration SQL text in an explicit
// BEGIN; ... ROLLBACK; envelope so it can be validated against a live
// Postgres connection without ever committing. Split out as a pure,
// independently-testable helper: validateMigrationSQL itself shells out to
// psql (via pipeSQLToContainer) and cannot be exercised in a unit test
// without a live Postgres target, but the envelope construction — the part
// that must be exactly right for ON_ERROR_STOP to abort on the first
// invalid statement and never commit — can be.
func wrapDryRunSQL(sqlContent string) string {
	return "BEGIN;\n" + sqlContent + "\nROLLBACK;\n"
}

// validateMigrationSQL runs sqlContent's exact statements inside a
// transaction that is always rolled back, surfacing any syntax/semantic
// error (including invalid constructs like `ADD CONSTRAINT IF NOT EXISTS`)
// before the real, irreversible apply. It reuses pipeSQLToContainer's
// existing container/embedded-PG dispatch and ON_ERROR_STOP=1 behavior, so
// the very first statement Postgres rejects aborts the whole dry run with a
// non-zero exit and the real Postgres error text — never a silent pass.
func validateMigrationSQL(ctx context.Context, cfg *config.Config, sqlContent string) error {
	if err := pipeSQLToContainer(ctx, cfg, wrapDryRunSQL(sqlContent)); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrMigrationValidationFailed, err)
	}
	return nil
}
