package database

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// Purpose: streams a pg_dump custom-format backup file into pg_restore
// against a specific target database (the drill DB created by RestoreDrill).
// Inputs: a context, *config.Config, the backup file path, and the target
// database name.
// Outputs: an error, or nil on a clean pg_restore exit.
// Constraints: split out of restore_drill.go (CLI-R12) as a pure move; no
// behavior changed.

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
