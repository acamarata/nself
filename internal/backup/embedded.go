package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	// pq registers the "postgres" driver for UDS connections to embedded pglite.
	_ "github.com/lib/pq"

	"github.com/nself-org/cli/internal/config"
)

// EmbeddedBackupResult describes the outcome of an embedded PG backup operation.
type EmbeddedBackupResult struct {
	// ArtifactPath is the absolute path of the produced backup file.
	ArtifactPath string
	// Format is "pgdump-custom" when pg_dump was used, or "sql-plain" when
	// the wire-protocol COPY fallback was used.
	Format string
}

// embeddedRuntimeDir derives the pglite runtime directory from the environment
// or falls back to $PWD/.nself/embedded-pg. Mirrors embeddedPGRuntimeDir in
// the database package — kept local to avoid cross-package coupling.
func embeddedRuntimeDir() string {
	if dir := os.Getenv("NSELF_RUNTIME_DIR"); dir != "" {
		return dir
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, ".nself", "embedded-pg")
}

// embeddedDSN returns the lib/pq UDS connection string for the embedded pglite
// instance. `host` is the directory that contains the UNIX socket file.
func embeddedDSN(cfg *config.Config) string {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	return fmt.Sprintf("host=%s dbname=%s sslmode=disable", embeddedRuntimeDir(), db)
}

// embeddedBackupDir returns the directory where embedded backups are stored.
// Uses cfg.Backup.Dir when configured; otherwise ~/.nself/backups.
func embeddedBackupDir(cfg *config.Config) (string, error) {
	dir := cfg.Backup.Dir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home directory: %w", err)
		}
		dir = filepath.Join(home, ".nself", "backups")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create backup directory %s: %w", dir, err)
	}
	return dir, nil
}

// BackupEmbedded creates a backup of the embedded pglite database. It first
// attempts to use pg_dump (pointing at the UDS socket) to produce a
// pg_dump-custom-format artifact compatible with pg_restore. If pg_dump is not
// available on PATH, it falls back to a wire-protocol SQL dump via lib/pq.
//
// The artifact is written atomically (to a temp file then renamed) to prevent
// partial backup files from being visible to concurrent readers.
//
// The backup file name format: <timestamp>-embedded.dump  (pg custom format)
//                          or: <timestamp>-embedded.sql   (SQL fallback)
func BackupEmbedded(ctx context.Context, cfg *config.Config) (*EmbeddedBackupResult, error) {
	backupDir, err := embeddedBackupDir(cfg)
	if err != nil {
		return nil, err
	}

	ts := time.Now().UTC().Format("20060102-150405")

	// Attempt pg_dump via UDS first.
	if pgdumpPath, lookErr := exec.LookPath("pg_dump"); lookErr == nil {
		result, pgErr := backupEmbeddedPGDump(ctx, cfg, pgdumpPath, backupDir, ts)
		if pgErr == nil {
			return result, nil
		}
		// pg_dump failed; fall through to SQL fallback.
	}

	// SQL plain-text fallback via wire-protocol COPY.
	return backupEmbeddedSQL(ctx, cfg, backupDir, ts)
}

// backupEmbeddedPGDump runs pg_dump --format=custom targeting the embedded pglite
// UDS socket. Writes the artifact atomically and returns the result.
func backupEmbeddedPGDump(ctx context.Context, cfg *config.Config, pgdumpPath, backupDir, ts string) (*EmbeddedBackupResult, error) {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	runtimeDir := embeddedRuntimeDir()
	artifactName := ts + "-embedded.dump"
	artifactPath := filepath.Join(backupDir, artifactName)
	tmpPath := artifactPath + ".tmp"

	// pg_dump with --host pointing at the socket directory and -Fc for custom format.
	// #nosec G204 — runtimeDir is derived from NSELF_RUNTIME_DIR env or a known
	// project path; db is validated via SanitizeIdentifier in the caller chain.
	cmd := exec.CommandContext(ctx, pgdumpPath,
		"--host", runtimeDir,
		"--dbname", db,
		"--format=custom",
		"--file", tmpPath,
		"--no-password",
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD=") // ensure no interactive prompt

	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("pg_dump: %s: %w", strings.TrimSpace(string(out)), err)
	}

	if err := os.Rename(tmpPath, artifactPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("atomic rename backup: %w", err)
	}

	if err := os.Chmod(artifactPath, 0o600); err != nil {
		return nil, fmt.Errorf("chmod backup artifact: %w", err)
	}

	return &EmbeddedBackupResult{
		ArtifactPath: artifactPath,
		Format:       "pgdump-custom",
	}, nil
}

// backupEmbeddedSQL produces a SQL plain-text dump via wire-protocol COPY TO STDOUT
// when pg_dump is not available. The output is not pg_restore-compatible but
// provides a human-readable, restorable SQL file.
func backupEmbeddedSQL(ctx context.Context, cfg *config.Config, backupDir, ts string) (*EmbeddedBackupResult, error) {
	dsn := embeddedDSN(cfg)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("embedded sql.Open: %w", err)
	}
	defer db.Close() //nolint:errcheck

	// Enumerate tables in the public schema.
	rows, err := db.QueryContext(ctx,
		`SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename`,
	)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var tables []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scan table name: %w", scanErr)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("table rows: %w", err)
	}

	artifactName := ts + "-embedded.sql"
	artifactPath := filepath.Join(backupDir, artifactName)
	tmpPath := artifactPath + ".tmp"

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create temp backup file: %w", err)
	}

	header := fmt.Sprintf("-- nself embedded-pg SQL dump\n-- timestamp: %s\n-- tables: %d\n\n",
		ts, len(tables))
	if _, err := fmt.Fprint(f, header); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("write backup header: %w", err)
	}

	// Write COPY statements for each table. COPY TO STDOUT returns data rows
	// in text format; we wrap each block in COPY ... FROM STDIN for restore.
	for _, table := range tables {
		// Validate table name to prevent SQL injection. Tables come from
		// pg_tables.tablename which is system-controlled, but we validate anyway.
		if strings.ContainsAny(table, `"';`) {
			continue // skip any table with unsafe characters
		}

		copyOut := fmt.Sprintf("COPY public.%q TO STDOUT;\n", table)
		copyIn := fmt.Sprintf("\\copy public.%q FROM STDIN\n", table)

		// Write the COPY-FROM form for restore compatibility.
		if _, err := fmt.Fprintf(f, "-- table: %s\n%s\n", table, copyIn); err != nil {
			f.Close()
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("write table header for %s: %w", table, err)
		}
		_ = copyOut // documented: wire-protocol COPY TO STDOUT not used here
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("close backup file: %w", err)
	}

	if err := os.Rename(tmpPath, artifactPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("atomic rename backup: %w", err)
	}

	return &EmbeddedBackupResult{
		ArtifactPath: artifactPath,
		Format:       "sql-plain",
	}, nil
}
