package database

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"nself/internal/config"
	"nself/internal/errs"
)

// waitForPostgres polls pg_isready inside the postgres container until it
// succeeds or the timeout (60s) is reached. Returns errs.ErrDatabaseNotRunning
// if postgres does not become ready in time.
func waitForPostgres(ctx context.Context, cfg *config.Config) error {
	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	const (
		maxWait  = 60 * time.Second
		interval = 1 * time.Second
	)

	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		cmd := exec.CommandContext(checkCtx, "docker", "exec", container, "pg_isready", "-U", user)
		err := cmd.Run()
		cancel()

		if err == nil {
			slog.Info("postgres is ready", "container", container)
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("PostgreSQL failed to start within 60s. Run 'nself logs postgres' for details: %w", errs.ErrDatabaseNotRunning)
}

// runSQLOnDB executes a SQL statement inside the postgres container via psql,
// targeting a specific database. This is needed for init operations that must
// connect to the "postgres" admin database (e.g., CREATE DATABASE).
func runSQLOnDB(ctx context.Context, cfg *config.Config, database string, sql string) error {
	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", database, "-c", sql,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql exec failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return nil
}

// createDatabase creates the target database if it does not already exist.
// It connects to the default "postgres" database to run the CREATE DATABASE.
func createDatabase(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	// Check if database already exists.
	checkSQL := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", db)
	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", "postgres", "-tAc", checkSQL,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("check database existence: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	if strings.TrimSpace(stdout.String()) == "1" {
		slog.Info("database already exists", "database", db)
		return nil
	}

	// Database does not exist; create it.
	createSQL := fmt.Sprintf("CREATE DATABASE \"%s\"", db)
	if err := runSQLOnDB(ctx, cfg, "postgres", createSQL); err != nil {
		return fmt.Errorf("create database %s: %w", db, err)
	}

	slog.Info("database created", "database", db)
	return nil
}

// createSchemas creates the required schemas (auth, storage, public) in the
// target database and grants all privileges to the configured user.
func createSchemas(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	schemas := []string{"auth", "storage", "public"}

	for _, schema := range schemas {
		createSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)
		if err := runSQLOnDB(ctx, cfg, db, createSQL); err != nil {
			return fmt.Errorf("create schema %s: %w", schema, err)
		}

		grantSQL := fmt.Sprintf("GRANT ALL ON SCHEMA %s TO \"%s\"", schema, user)
		if err := runSQLOnDB(ctx, cfg, db, grantSQL); err != nil {
			return fmt.Errorf("grant schema %s to %s: %w", schema, user, err)
		}
	}

	slog.Info("schemas created and granted", "schemas", schemas, "user", user)
	return nil
}

// createExtensions installs the required PostgreSQL extensions (pgcrypto, citext)
// in the target database. Note: uuid-ossp is NOT auto-created.
func createExtensions(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	extensions := []string{"pgcrypto", "citext"}

	for _, ext := range extensions {
		sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", ext)
		if err := runSQLOnDB(ctx, cfg, db, sql); err != nil {
			return fmt.Errorf("create extension %s: %w", ext, err)
		}
	}

	slog.Info("extensions created", "extensions", extensions)
	return nil
}

// InitializeDatabase waits for PostgreSQL to become ready, then creates the
// database, schemas, grants, and required extensions. This is Phase 3 of the
// nself startup sequence.
//
// Steps:
//  1. Wait for pg_isready (max 60s, check every 1s)
//  2. CREATE DATABASE IF NOT EXISTS
//  3. CREATE SCHEMA IF NOT EXISTS auth, storage, public
//  4. GRANT ALL ON SCHEMA auth, storage, public TO user
//  5. CREATE EXTENSION IF NOT EXISTS pgcrypto, citext
func InitializeDatabase(ctx context.Context, cfg *config.Config) error {
	slog.Info("initializing database")

	if err := waitForPostgres(ctx, cfg); err != nil {
		return err
	}

	if err := createDatabase(ctx, cfg); err != nil {
		return err
	}

	if err := createSchemas(ctx, cfg); err != nil {
		return err
	}

	if err := createExtensions(ctx, cfg); err != nil {
		return err
	}

	slog.Info("database initialization complete")
	return nil
}
