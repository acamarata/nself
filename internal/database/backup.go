package database

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/backup"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/security"
)

// Backup runs pg_dump inside the project's postgres container and streams the
// output directly to a file on disk. The custom format (-Fc) is used so the
// dump can be restored with pg_restore.
//
// If outputPath is empty, a timestamped filename is generated under a
// "backups/" directory relative to the current working directory.
func Backup(ctx context.Context, cfg *config.Config, outputPath string) error {
	// Determine the backup root directory from config (default: "backups").
	backupDir := cfg.Backup.Dir
	if backupDir == "" {
		backupDir = "backups"
	}

	if outputPath == "" {
		ts := time.Now().Format("20060102_150405")
		outputPath = filepath.Join(backupDir, fmt.Sprintf("%s_%s.dump", cfg.ProjectName, ts))
	} else {
		// Validate that a caller-supplied path does not escape the backup directory.
		validated, err := backup.ValidateBackupPath(backupDir, outputPath)
		if err != nil {
			return fmt.Errorf("invalid backup path: %w", err)
		}
		outputPath = validated
	}

	// Ensure the parent directory exists with restricted permissions.
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create backup directory %s: %w", dir, err)
	}

	container := cfg.ProjectName + "_postgres"

	// Build the docker exec + pg_dump command.
	args := []string{
		"exec", container,
		"pg_dump",
		"-U", cfg.Postgres.User,
		"-d", cfg.Postgres.DB,
		"-Fc",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	// TRAP 3: Stream stdout directly to disk. Never buffer in memory.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pg_dump stdout pipe: %w", err)
	}

	// Capture stderr so we can surface pg_dump errors.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("pg_dump stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_dump: %w", err)
	}

	// Open the destination file for writing.
	f, err := os.Create(outputPath)
	if err != nil {
		// Kill the running process since we cannot consume its output.
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("create backup file %s: %w", outputPath, err)
	}
	defer func() { _ = f.Close() }()

	// Stream pg_dump output to the file.
	if _, err := io.Copy(f, stdout); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("stream pg_dump output: %w", err)
	}

	// Read any stderr output for error reporting.
	errOutput, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		// Clean up the partial dump file on failure.
		_ = os.Remove(outputPath)
		if len(errOutput) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupFailed, string(errOutput))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupFailed, err)
	}

	// Restrict backup dump file to owner-read/write only.
	if err := security.EnforceFilePermissions(outputPath, 0600); err != nil {
		return fmt.Errorf("enforcing permissions on backup %s: %w", outputPath, err)
	}

	return nil
}
