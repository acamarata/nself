package database

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/backup"
	"github.com/nself-org/cli/internal/config"
)

// Restore streams a pg_dump custom-format file into the project's postgres
// container via pg_restore. The file contents are piped to docker exec's stdin,
// avoiding the need to copy the dump into the container first.
func Restore(ctx context.Context, cfg *config.Config, inputPath string) error {
	// Validate that the restore file path stays within the backup directory.
	backupDir := cfg.Backup.Dir
	if backupDir == "" {
		backupDir = "backups"
	}
	validated, err := backup.ValidateBackupPath(backupDir, inputPath)
	if err != nil {
		return fmt.Errorf("invalid restore path: %w", err)
	}
	inputPath = validated

	// Validate restore file: must be .sql or .sql.gz and a regular file.
	if err := validateRestoreFile(inputPath); err != nil {
		return err
	}

	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open backup file %s: %w", inputPath, err)
	}
	defer func() { _ = f.Close() }()

	container := cfg.ProjectName + "_postgres"

	// Build the docker exec -i + pg_restore command.
	args := []string{
		"exec", "-i", container,
		"pg_restore",
		"-U", cfg.Postgres.User,
		"-d", cfg.Postgres.DB,
		"-Fc",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	// Pipe the backup file into the container's stdin.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("pg_restore stdin pipe: %w", err)
	}

	// Capture stderr for error reporting.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("pg_restore stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_restore: %w", err)
	}

	// Stream the file into pg_restore's stdin.
	if _, err := io.Copy(stdin, f); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("stream to pg_restore: %w", err)
	}

	// Close stdin to signal EOF so pg_restore finishes processing.
	if err := stdin.Close(); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("close pg_restore stdin: %w", err)
	}

	// Read any stderr output for error reporting.
	errOutput, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		if len(errOutput) > 0 {
			return fmt.Errorf("pg_restore failed: %s", string(errOutput))
		}
		return fmt.Errorf("pg_restore failed: %w", err)
	}

	return nil
}

// validateRestoreFile checks that path has a valid backup extension and is a regular file.
func validateRestoreFile(path string) error {
	if !strings.HasSuffix(path, ".sql") && !strings.HasSuffix(path, ".sql.gz") && !strings.HasSuffix(path, ".dump") {
		return fmt.Errorf("restore file must have .sql, .sql.gz, or .dump extension: %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("restore file not accessible: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("restore path %q is not a regular file", path)
	}
	return nil
}
