package database

import (
	"context"
	"fmt"

	"github.com/nself-org/cli/internal/config"
)

// advisoryLockKey is the hash used for pg_advisory_lock to prevent concurrent migration runs.
const advisoryLockKey = "hashtext('nself:migrations')"

// acquireAdvisoryLock obtains a PostgreSQL advisory lock for migrations.
// Returns an error if the lock cannot be acquired (another migration is running).
func acquireAdvisoryLock(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	sql := fmt.Sprintf("SELECT pg_advisory_lock(%s)", advisoryLockKey)
	_, err := querySQL(ctx, cfg, db, sql)
	if err != nil {
		return fmt.Errorf("acquire migration lock (another migration may be running): %w", err)
	}
	return nil
}

// releaseAdvisoryLock releases the PostgreSQL advisory lock for migrations.
func releaseAdvisoryLock(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	sql := fmt.Sprintf("SELECT pg_advisory_unlock(%s)", advisoryLockKey)
	_, err := querySQL(ctx, cfg, db, sql)
	if err != nil {
		return fmt.Errorf("release migration lock: %w", err)
	}
	return nil
}
