package claw

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"os/exec"
)

// postgresContainer returns the postgres container name for the project.
func postgresContainer(cfg *config.Config) string {
	return cfg.ProjectName + "_postgres"
}

// postgresUser returns the postgres user from config with a safe default.
func postgresUser(cfg *config.Config) string {
	if cfg.Postgres.User != "" {
		return cfg.Postgres.User
	}
	return "postgres"
}

// postgresDB returns the target database name with a safe default.
func postgresDB(cfg *config.Config) string {
	if cfg.Postgres.DB != "" {
		return cfg.Postgres.DB
	}
	return "nself"
}

// runClawSQL pipes raw SQL into psql inside the postgres container.
// The SQL is executed against the project's database (cfg.Postgres.DB or "nself").
// On error, the psql stderr is included in the returned error message.
func runClawSQL(ctx context.Context, cfg *config.Config, sql string) error {
	container := postgresContainer(cfg)
	user := postgresUser(cfg)
	db := postgresDB(cfg)

	args := []string{
		"exec", "-i", container,
		"psql",
		"-U", user,
		"-d", db,
		"-v", "ON_ERROR_STOP=1",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = strings.NewReader(sql)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// queryClawSQL executes a SQL query inside the postgres container and returns
// the trimmed stdout. Useful for SELECT queries that return a single value or
// newline-separated list.
func queryClawSQL(ctx context.Context, cfg *config.Config, sql string) (string, error) {
	container := postgresContainer(cfg)
	user := postgresUser(cfg)
	db := postgresDB(cfg)

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("psql: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}
