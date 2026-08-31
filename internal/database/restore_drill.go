package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// RestoreDrillResult describes the outcome of a restore test.
type RestoreDrillResult struct {
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
	BackupFile     string    `json:"backup_file"`
	Success        bool      `json:"success"`
	TablesVerified int       `json:"tables_verified"`
	RowsVerified   int64     `json:"rows_verified"`
	// MissingCriticalTables lists the entries of CriticalTables that were not
	// found (by name, any schema) in the restored database. Informational —
	// see the naming-mismatch note in drill.go: several drilled environments
	// do not use the np_ prefix, so this is reported rather than failing the
	// drill outright.
	MissingCriticalTables []string      `json:"missing_critical_tables,omitempty"`
	ErrorMessage          string        `json:"error_message,omitempty"`
	Duration              time.Duration `json:"duration_ns"`
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

	// Which of the canonical CriticalTables actually exist by name. Query
	// while the scratch DB is still alive — the caller in drill.go cannot
	// requery after this function drops it below.
	missing, missErr := verifyCriticalTables(ctx, cfg, drillDB)
	if missErr != nil {
		_ = runSQLOnDB(ctx, cfg, "postgres", "DROP DATABASE IF EXISTS "+quotedDrillDB)
		result.ErrorMessage = missErr.Error()
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		_ = RecordDrillResult(".", result)
		return result, fmt.Errorf("verify critical tables: %w", missErr)
	}
	result.MissingCriticalTables = missing

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

	// Exact total row count across ALL user tables, in one round trip.
	//
	// The prior approach sampled up to 10 tables (LIMIT 10, no ORDER BY) and
	// summed their row counts. On a schema with many tables that sample could
	// land entirely on empty ones and report zero rows for a perfectly good
	// restore — a false fail. It could just as easily miss the tables that
	// actually hold data and report zero for the opposite reason: a false
	// pass. Neither is acceptable once the caller (drill.go) starts asserting
	// on this number.
	//
	// query_to_xml lets Postgres build and run one dynamic COUNT(*) per table
	// server-side and return all the results in a single query. format('%I.%I', ...)
	// is Postgres's own identifier quoting (SEC-SQL-01: table_schema/table_name
	// here are DB-sourced and must never be interpolated by the Go side without
	// it — %I does that quoting inside the server, so no client-side
	// SanitizeIdentifier call is needed for this query).
	rowTotalSQL := `SELECT COALESCE(SUM((xpath('/row/c/text()', ` +
		`query_to_xml(format('SELECT count(*) AS c FROM %I.%I', table_schema, table_name), false, true, '')` +
		`))[1]::text::bigint), 0) FROM information_schema.tables ` +
		`WHERE table_schema NOT IN ('pg_catalog','information_schema')`
	rowOut, err := querySQL(ctx, cfg, drillDB, rowTotalSQL)
	if err != nil {
		return tableCount, 0, fmt.Errorf("count total rows in %s: %w", drillDB, err)
	}
	rowOut = strings.TrimSpace(rowOut)
	var totalRows int64
	if rowOut != "" {
		if _, scanErr := fmt.Sscanf(rowOut, "%d", &totalRows); scanErr != nil {
			return tableCount, 0, fmt.Errorf("parse row total %q: %w", rowOut, scanErr)
		}
	}

	// Verify pg_catalog is accessible by probing a relation size.
	catalogSQL := `SELECT pg_catalog.pg_relation_size(c.oid) FROM pg_catalog.pg_class c WHERE c.relname = 'pg_class' LIMIT 1`
	if _, catErr := querySQL(ctx, cfg, drillDB, catalogSQL); catErr != nil {
		return tableCount, totalRows, fmt.Errorf("pg_catalog probe failed in %s: %w", drillDB, catErr)
	}

	return tableCount, totalRows, nil
}

// verifyCriticalTables reports which entries of CriticalTables (drill.go) are
// missing by name, in any schema, from drillDB. CriticalTables are
// compile-time-constant literals defined in this package, not DB-sourced
// input, so they are safe to place directly into a SQL literal list here —
// SEC-SQL-01's "never interpolate DB-sourced identifiers without quoting"
// concerns table_schema/table_name values read back FROM the database
// (handled via format('%I.%I') in VerifyRestoredDatabase above), not our own
// hardcoded constants.
func verifyCriticalTables(ctx context.Context, cfg *config.Config, drillDB string) ([]string, error) {
	literals := make([]string, len(CriticalTables))
	for i, name := range CriticalTables {
		literals[i] = "'" + strings.ReplaceAll(name, "'", "''") + "'"
	}
	sqlText := fmt.Sprintf(
		`SELECT DISTINCT table_name FROM information_schema.tables WHERE table_name = ANY(ARRAY[%s])`,
		strings.Join(literals, ","),
	)
	out, err := querySQL(ctx, cfg, drillDB, sqlText)
	if err != nil {
		return nil, fmt.Errorf("query critical tables in %s: %w", drillDB, err)
	}
	present := make(map[string]bool, len(CriticalTables))
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			present[line] = true
		}
	}
	var missing []string
	for _, name := range CriticalTables {
		if !present[name] {
			missing = append(missing, name)
		}
	}
	return missing, nil
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
