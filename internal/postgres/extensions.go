package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DefaultExtensions are always installed regardless of POSTGRES_EXTENSIONS config.
var DefaultExtensions = []string{"uuid-ossp", "pgcrypto"}

// extensionNameRE validates extension names to prevent injection.
// Allows lowercase letters, digits, underscores, and hyphens; max 64 chars.
var extensionNameRE = regexp.MustCompile(`^[a-z][a-z0-9_\-]{0,63}$`)

// InstallExtensions runs CREATE EXTENSION IF NOT EXISTS for DefaultExtensions
// plus any extras provided. Called from nself start Phase 2 (DB init).
// Safe to call multiple times — IF NOT EXISTS is idempotent.
func InstallExtensions(ctx context.Context, dsn string, extras []string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() { _ = db.Close() }()

	all := dedupStrings(append(DefaultExtensions, extras...))
	for _, ext := range all {
		if !extensionNameRE.MatchString(ext) {
			return fmt.Errorf("invalid extension name %q", ext)
		}
		q := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s"`, ext)
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("install %s: %w", ext, err)
		}
	}
	return nil
}

func dedupStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
