package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// RestoreDrillResult describes the outcome of a restore test.
type RestoreDrillResult struct {
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    time.Time     `json:"completed_at"`
	BackupFile     string        `json:"backup_file"`
	Success        bool          `json:"success"`
	TablesVerified int           `json:"tables_verified"`
	RowsVerified   int64         `json:"rows_verified"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	Duration       time.Duration `json:"duration_ns"`
}

// RestoreDrill runs a restore drill using the most recent backup.
// It restores to a TEMPORARY database named {dbname}_drilltest, verifies
// data integrity, then drops the temporary database.
// This is non-destructive — it does not touch the production database.
func RestoreDrill(ctx context.Context, cfg *config.Config, backupFile string) (RestoreDrillResult, error) {
	result := RestoreDrillResult{
		StartedAt: time.Now(),
	}

	// Find most recent backup if not specified.
	if backupFile == "" {
		found, err := mostRecentBackup(cfg)
		if err != nil {
			return result, fmt.Errorf("find most recent backup: %w", err)
		}
		backupFile = found
	}
	result.BackupFile = backupFile

	// Derive drill database name.
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	drillDB := db + "_drilltest"

	// Validate and double-quote the drill DB identifier before any DDL. SEC-SQL-01.
	quotedDrillDB, err := SanitizeIdentifier(drillDB)
	if err != nil {
		return result, fmt.Errorf("invalid drill database name %q: %w", drillDB, err)
	}

	// Ensure drill database does not already exist (clean state).
	_ = runSQLOnDB(ctx, cfg, "postgres", "DROP DATABASE IF EXISTS "+quotedDrillDB)

	// Create the temporary drill database.
	if err := runSQLOnDB(ctx, cfg, "postgres", "CREATE DATABASE "+quotedDrillDB); err != nil {
		return result, fmt.Errorf("create drill database %s: %w", drillDB, err)
	}

	// Restore backup into the drill database.
	// We build a restore-like command targeting drillDB directly.
	if err := restoreToDB(ctx, cfg, backupFile, drillDB); err != nil {
		_ = runSQLOnDB(ctx, cfg, "postgres", "DROP DATABASE IF EXISTS "+quotedDrillDB)
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		_ = RecordDrillResult(".", result)
		return result, fmt.Errorf("restore into drill database: %w", err)
	}

	// Verify the restored database.
	tables, rows, err := VerifyRestoredDatabase(ctx, cfg, drillDB)
	if err != nil {
		_ = runSQLOnDB(ctx, cfg, "postgres", "DROP DATABASE IF EXISTS "+quotedDrillDB)
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		_ = RecordDrillResult(".", result)
		return result, fmt.Errorf("verify drill database: %w", err)
	}

	result.TablesVerified = tables
	result.RowsVerified = rows

	// Drop the drill database.
	if err := runSQLOnDB(ctx, cfg, "postgres", "DROP DATABASE IF EXISTS "+quotedDrillDB); err != nil {
		return result, fmt.Errorf("drop drill database %s: %w", drillDB, err)
	}

	result.Success = true
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	if err := RecordDrillResult(".", result); err != nil {
		return result, fmt.Errorf("record drill result: %w", err)
	}

	return result, nil
}

// VerifyRestoredDatabase connects to a restored database and runs basic
// integrity checks: counts rows in key tables, verifies pg_catalog consistency.
func VerifyRestoredDatabase(ctx context.Context, cfg *config.Config, drillDB string) (tables int, rows int64, err error) {
	// Count user tables.
	tableCountSQL := `SELECT count(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema')`
	out, err := querySQL(ctx, cfg, drillDB, tableCountSQL)
	if err != nil {
		return 0, 0, fmt.Errorf("count tables in %s: %w", drillDB, err)
	}
	out = strings.TrimSpace(out)
	tableCount := 0
	if out != "" {
		if _, scanErr := fmt.Sscanf(out, "%d", &tableCount); scanErr != nil {
			return 0, 0, fmt.Errorf("parse table count %q: %w", out, scanErr)
		}
	}

	// Get up to 10 user tables to sample row counts from.
	tableListSQL := `SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') LIMIT 10`
	tableListOut, err := querySQL(ctx, cfg, drillDB, tableListSQL)
	if err != nil {
		return tableCount, 0, fmt.Errorf("list sample tables in %s: %w", drillDB, err)
	}

	var totalRows int64
	for _, tableLine := range strings.Split(tableListOut, "\n") {
		tableLine = strings.TrimSpace(tableLine)
		if tableLine == "" {
			continue
		}
		// tableLine is "schema.table_name" returned by information_schema.
		// Sanitize each component individually to produce a safe qualified identifier.
		// SEC-SQL-01: never interpolate DB-sourced identifiers without quoting.
		parts := strings.SplitN(tableLine, ".", 2)
		var qualifiedTable string
		if len(parts) == 2 {
			quotedSchema, schErr := SanitizeIdentifier(parts[0])
			quotedTable, tblErr := SanitizeIdentifier(parts[1])
			if schErr != nil || tblErr != nil {
				continue // skip unquotable identifiers
			}
			qualifiedTable = quotedSchema + "." + quotedTable
		} else {
			quoted, qErr := SanitizeIdentifier(tableLine)
			if qErr != nil {
				continue
			}
			qualifiedTable = quoted
		}
		countSQL := "SELECT count(*) FROM " + qualifiedTable
		rowOut, rowErr := querySQL(ctx, cfg, drillDB, countSQL)
		if rowErr != nil {
			continue
		}
		rowOut = strings.TrimSpace(rowOut)
		var n int64
		if _, scanErr := fmt.Sscanf(rowOut, "%d", &n); scanErr == nil {
			totalRows += n
		}
	}

	// Verify pg_catalog is accessible by probing a relation size.
	catalogSQL := `SELECT pg_catalog.pg_relation_size(c.oid) FROM pg_catalog.pg_class c WHERE c.relname = 'pg_class' LIMIT 1`
	if _, catErr := querySQL(ctx, cfg, drillDB, catalogSQL); catErr != nil {
		return tableCount, totalRows, fmt.Errorf("pg_catalog probe failed in %s: %w", drillDB, catErr)
	}

	return tableCount, totalRows, nil
}

// RecordDrillResult appends the drill result to .nself/restore-drills.log
// in JSON format (one JSON object per line).
func RecordDrillResult(projectDir string, result RestoreDrillResult) error {
	logPath := filepath.Join(projectDir, ".nself", "restore-drills.log")

	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return fmt.Errorf("create .nself directory: %w", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal drill result: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open drill log %s: %w", logPath, err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write drill log entry: %w", err)
	}

	return nil
}

// mostRecentBackup returns the path to the most recently modified .dump file
// in the configured backup directory.
func mostRecentBackup(cfg *config.Config) (string, error) {
	backupDir := cfg.Backup.Dir
	if backupDir == "" {
		backupDir = "backups"
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return "", fmt.Errorf("read backup directory %s: %w", backupDir, err)
	}

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var dumps []fileInfo

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".dump") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		dumps = append(dumps, fileInfo{
			path:    filepath.Join(backupDir, e.Name()),
			modTime: info.ModTime(),
		})
	}

	if len(dumps) == 0 {
		return "", fmt.Errorf("no .dump files found in %s", backupDir)
	}

	sort.Slice(dumps, func(i, j int) bool {
		return dumps[i].modTime.After(dumps[j].modTime)
	})

	return dumps[0].path, nil
}

// restoreToDB restores a pg_dump custom-format file into a specific database.
// This is a targeted variant of Restore that accepts an explicit target database name.
func restoreToDB(ctx context.Context, cfg *config.Config, backupFile string, targetDB string) error {
	f, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("open backup file %s: %w", backupFile, err)
	}
	defer f.Close()

	container := containerName(cfg)
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	args := []string{
		"exec", "-i", container,
		"pg_restore",
		"-U", user,
		"-d", targetDB,
		"-Fc",
		"--no-owner",
		"--no-acl",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("pg_restore stdin pipe: %w", err)
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_restore: %w", err)
	}

	if _, err := io.Copy(stdin, f); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("stream to pg_restore: %w", err)
	}

	if err := stdin.Close(); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("close pg_restore stdin: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("pg_restore failed: %s", msg)
		}
		return fmt.Errorf("pg_restore failed: %w", err)
	}

	return nil
}
