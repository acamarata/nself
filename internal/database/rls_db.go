package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/nself-org/cli/internal/config"
)

// Open opens a database/sql connection to the project's PostgreSQL instance.
// The caller is responsible for calling db.Close() when done.
func Open(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	url := cfg.DatabaseURL()
	if url == "" {
		return nil, fmt.Errorf("database URL is empty — check POSTGRES_* env vars in your .env")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w (is the postgres container running?)", err)
	}
	return db, nil
}
