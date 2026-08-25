package database

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// Purpose: one-time upgrade path for ledger rows written before the unique-key
// scheme existed — rewriting old date-prefix ids in nself_ops.migrations and
// resolving the legacy nested-layout 'up.sql' collision to the correct key.
// Inputs: a *config.Config, the on-disk migration file list, and (for the
// resolver) the old-style ops ids recorded alongside a legacy row.
// Outputs: idempotent SQL side effects against the ledger tables.
// Constraints: split out of migrate.go (CLI-R12) as a pure move; no behavior
// changed. Depends on migrate_sql.go (pipeSQLToContainer, querySQL) and
// migrate_ledger.go (migrationKey, validateMigrationName).

// legacyOpsIDBackfillSQL upgrades pre-existing nself_ops.migrations rows whose
// id was the old date/timestamp prefix (extractMigrationID used to truncate at
// the first underscore) to the new unique id (full name minus ".sql").
// Idempotent: the `id <>` guard makes re-runs a no-op; the NOT EXISTS guard
// avoids PK conflicts if a new-style row already exists. Rows named 'up.sql'
// (legacy nested layout) are handled separately by legacyNestedRenameSQL.
const legacyOpsIDBackfillSQL = `UPDATE nself_ops.migrations m
SET id = left(m.name, length(m.name) - 4)
WHERE m.name LIKE '%.sql'
  AND m.name <> 'up.sql'
  AND m.id <> left(m.name, length(m.name) - 4)
  AND NOT EXISTS (
    SELECT 1 FROM nself_ops.migrations m2
    WHERE m2.id = left(m.name, length(m.name) - 4)
  );`

// legacyNestedRenameSQL rewrites the single legacy nested-layout ledger row
// (name = 'up.sql') to the resolved migration key. Idempotent: after the first
// run no 'up.sql' rows remain, so every statement is a no-op. The trailing
// DELETEs clean up only when the target key already exists (never loses the
// applied fact — the UPDATE runs first).
func legacyNestedRenameSQL(key string) string {
	k := strings.ReplaceAll(key, "'", "''")
	return fmt.Sprintf(`BEGIN;
UPDATE np_common.schema_versions SET name = '%s' WHERE name = 'up.sql' AND NOT EXISTS (SELECT 1 FROM np_common.schema_versions sv2 WHERE sv2.name = '%s');
DELETE FROM np_common.schema_versions WHERE name = 'up.sql';
UPDATE nself_ops.migrations SET id = '%s', name = '%s' WHERE name = 'up.sql' AND NOT EXISTS (SELECT 1 FROM nself_ops.migrations m2 WHERE m2.id = '%s');
DELETE FROM nself_ops.migrations WHERE name = 'up.sql';
COMMIT;
`, k, k, k, k, k)
}

// resolveLegacyNestedKey decides which on-disk nested migration a legacy
// 'up.sql' ledger row belongs to. opsIDs are the old prefix-style ids of
// nself_ops.migrations rows named 'up.sql' (recorded in the same transaction
// as the legacy row, so they identify the directory that actually ran).
// Falls back to the first nested migration in sorted order — under the old
// code only the lexicographically first nested migration could ever have been
// applied (all later ones were skipped by the name collision).
// Returns "" when no nested migrations exist on disk.
func resolveLegacyNestedKey(files []string, opsIDs []string) string {
	var nested []string
	for _, f := range files {
		if filepath.Base(f) == "up.sql" {
			nested = append(nested, migrationKey(f))
		}
	}
	if len(nested) == 0 {
		return ""
	}
	sort.Strings(nested)

	for _, id := range opsIDs {
		var matches []string
		for _, k := range nested {
			if k == id || strings.SplitN(k, "_", 2)[0] == id {
				matches = append(matches, k)
			}
		}
		if len(matches) == 1 {
			return matches[0]
		}
	}
	return nested[0]
}

// upgradeLedger migrates legacy ledger rows (written before the unique-key
// scheme) in place so already-deployed boxes upgrade cleanly without
// re-running applied migrations:
//  1. nself_ops.migrations rows keyed by the old date prefix get their id
//     rewritten to the full unique id (SQL-side, idempotent).
//  2. The legacy nested-layout 'up.sql' row in both ledgers is renamed to the
//     directory-derived key of the migration that actually ran.
//
// Must be called after ensureSchemaVersions + ensureMigrationsTable.
func upgradeLedger(ctx context.Context, cfg *config.Config, files []string) error {
	if err := pipeSQLToContainer(ctx, cfg, legacyOpsIDBackfillSQL); err != nil {
		return fmt.Errorf("ledger id backfill: %w", err)
	}

	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	out, err := querySQL(ctx, cfg, db, "SELECT count(*) FROM np_common.schema_versions WHERE name = 'up.sql'")
	if err != nil {
		return fmt.Errorf("ledger legacy row check: %w", err)
	}
	if strings.TrimSpace(out) == "0" || strings.TrimSpace(out) == "" {
		return nil
	}

	opsOut, err := querySQL(ctx, cfg, db, "SELECT id FROM nself_ops.migrations WHERE name = 'up.sql' ORDER BY id")
	if err != nil {
		return fmt.Errorf("ledger legacy ops rows: %w", err)
	}
	var opsIDs []string
	for _, line := range strings.Split(opsOut, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			opsIDs = append(opsIDs, line)
		}
	}

	key := resolveLegacyNestedKey(files, opsIDs)
	if key == "" {
		// No nested migrations on disk: nothing the row could collide with.
		return nil
	}
	if err := validateMigrationName(key); err != nil {
		return fmt.Errorf("ledger upgrade key: %w", err)
	}
	if err := pipeSQLToContainer(ctx, cfg, legacyNestedRenameSQL(key)); err != nil {
		return fmt.Errorf("ledger legacy row rename: %w", err)
	}
	return nil
}
