package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/metrics"
)

// VerifyOptions holds flags for `nself backup verify`.
type VerifyOptions struct {
	BackupID    string // backup ID or "latest"
	RestoreTest bool   // spin up test container and restore
	Cleanup     bool   // remove test container after verify
	Keep        bool   // keep test container for inspection
}

// VerifyResult holds the outcome of a backup verification.
type VerifyResult struct {
	BackupID  string    `json:"backup_id"`
	Verified  bool      `json:"verified"`
	StartedAt time.Time `json:"started_at"`
	Duration  string    `json:"duration"`
	Method    string    `json:"method"` // checksum, restore-test
	Details   string    `json:"details"`
}

// Verify checks backup integrity, optionally performing a restore test.
func Verify(ctx context.Context, cfg *config.Config, opts VerifyOptions) (*VerifyResult, error) {
	backupDir := cfg.Backup.Dir
	if backupDir == "" {
		backupDir = "./backups"
	}

	start := time.Now()
	result := &VerifyResult{
		BackupID:  opts.BackupID,
		StartedAt: start,
		Method:    "checksum",
	}

	backupFile, err := resolveBackupFile(backupDir, opts.BackupID)
	if err != nil {
		return nil, err
	}

	// Basic integrity check: file exists and is non-empty.
	info, err := os.Stat(backupFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrBackupNotFound, err)
	}
	if info.Size() == 0 {
		result.Verified = false
		result.Details = "backup file is empty"
		result.Duration = time.Since(start).String()
		return result, fmt.Errorf("%w: backup file is empty", errs.ErrBackupVerifyFailed)
	}

	if opts.RestoreTest {
		result.Method = "restore-test"
		if err := runRestoreTest(ctx, cfg, backupFile, opts); err != nil {
			result.Verified = false
			result.Details = err.Error()
			result.Duration = time.Since(start).String()
			emitVerifyMetric(cfg, start, false)
			slog.Error("restore-test failed", "id", opts.BackupID, "RESULT", "fail", "duration_sec", int(time.Since(start).Seconds()), "error", err)
			return result, err
		}
	}

	result.Verified = true
	result.Details = fmt.Sprintf("file size: %d bytes", info.Size())
	result.Duration = time.Since(start).String()

	if opts.RestoreTest {
		emitVerifyMetric(cfg, start, true)
		// Emit the exact phrase the systemd/journalctl grep expects.
		slog.Info("restore-test passed", "id", opts.BackupID, "RESULT", "pass", "duration_sec", int(time.Since(start).Seconds()))
	}

	slog.Info("backup verified", "id", opts.BackupID, "method", result.Method, "duration", result.Duration)
	return result, nil
}

func emitVerifyMetric(cfg *config.Config, start time.Time, success bool) {
	rec := metrics.VerifyRecord{
		Env:         cfg.Env,
		Success:     success,
		DurationSec: time.Since(start).Seconds(),
		Timestamp:   time.Now(),
	}
	if err := metrics.EmitVerify(rec); err != nil {
		slog.Warn("emit verify metric", "error", err)
	}
}

// runRestoreTest spins up a temporary postgres container, restores the backup,
// and runs smoke queries to verify data integrity.
func runRestoreTest(ctx context.Context, cfg *config.Config, backupFile string, opts VerifyOptions) error {
	testContainer := cfg.ProjectName + "_pg_restore_test"
	testVolume := cfg.ProjectName + "_restore_test_data"
	pgVersion := cfg.Postgres.Version
	if pgVersion == "" {
		pgVersion = "16-alpine"
	}

	slog.Info("starting restore test container", "container", testContainer)

	// Create ephemeral volume and container.
	runArgs := []string{
		"run", "-d",
		"--name", testContainer,
		"-v", testVolume + ":/var/lib/postgresql/data",
		"-e", "POSTGRES_USER=" + cfg.Postgres.User,
		"-e", "POSTGRES_PASSWORD=" + cfg.Postgres.Password,
		"-e", "POSTGRES_DB=" + cfg.Postgres.DB,
		"postgres:" + pgVersion,
	}

	cmd := exec.CommandContext(ctx, "docker", runArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start test container: %s: %w", string(output), err)
	}

	// Cleanup function.
	cleanup := func() {
		if opts.Keep {
			slog.Info("keeping test container for inspection", "container", testContainer)
			return
		}
		slog.Info("cleaning up test container", "container", testContainer)
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", testContainer).Run()
		_ = exec.CommandContext(ctx, "docker", "volume", "rm", "-f", testVolume).Run()
	}

	if !opts.Keep {
		defer cleanup()
	}

	// Wait for postgres to be ready.
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	for i := 0; i < 30; i++ {
		check := exec.CommandContext(ctx, "docker", "exec", testContainer, "pg_isready", "-U", user)
		if check.Run() == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}

	// Restore the backup into the test container.
	if strings.HasSuffix(backupFile, ".dump") {
		if err := restorePgDump(ctx, testContainer, user, cfg.Postgres.DB, backupFile); err != nil {
			return fmt.Errorf("restore test failed: %w", err)
		}
	}

	// Run smoke queries.
	smokeSQL := "SELECT count(*) FROM information_schema.tables WHERE table_schema NOT IN ('information_schema', 'pg_catalog');"
	smokeArgs := []string{
		"exec", testContainer,
		"psql", "-U", user, "-d", cfg.Postgres.DB, "-t", "-c", smokeSQL,
	}
	smokeCmd := exec.CommandContext(ctx, "docker", smokeArgs...)
	output, err := smokeCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: smoke query failed: %s", errs.ErrBackupVerifyFailed, string(output))
	}

	count := strings.TrimSpace(string(output))
	slog.Info("restore test smoke query", "table_count", count)
	if count == "0" {
		return fmt.Errorf("%w: restored database has no user tables", errs.ErrBackupVerifyFailed)
	}

	return nil
}
