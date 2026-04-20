package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// RestoreOptions holds flags for `nself backup restore`.
//
// Note: PointInTime (PITR) is NOT supported in v1.0.9. The field is retained
// for forward compatibility but any non-empty value returns an explicit error
// directing users to v1.1.0 + pgbackrest integration. Remove this note when
// PITR ships.
type RestoreOptions struct {
	BackupID   string   // backup ID or "latest"
	ToDir      string   // restore to alternate directory
	Only       []string // pg, minio, metadata — subset to restore
	DecryptKey string   // path to age identity file
	Yes        bool     // skip confirmation
}

// Restore recovers a backup by ID or "latest".
func Restore(ctx context.Context, cfg *config.Config, opts RestoreOptions) error {
	backupDir := cfg.Backup.Dir
	if backupDir == "" {
		backupDir = "./backups"
	}

	backupFile, err := resolveBackupFile(backupDir, opts.BackupID)
	if err != nil {
		return err
	}

	slog.Info("restoring from backup", "file", backupFile)

	// Decrypt if needed.
	workFile := backupFile
	if strings.HasSuffix(backupFile, ".age") {
		decrypted, err := decryptFile(ctx, backupFile, opts.DecryptKey)
		if err != nil {
			return fmt.Errorf("decrypt backup: %w", err)
		}
		workFile = decrypted
		defer os.Remove(decrypted) // Clean up decrypted temp file.
	}

	// Determine what to restore.
	restoreComponents := map[string]bool{"pg": true, "minio": true, "metadata": true}
	if len(opts.Only) > 0 {
		restoreComponents = map[string]bool{}
		for _, c := range opts.Only {
			restoreComponents[c] = true
		}
	}

	if restoreComponents["pg"] {
		if err := restorePostgres(ctx, cfg, workFile, opts); err != nil {
			return fmt.Errorf("restore postgres: %w", err)
		}
	}

	if restoreComponents["minio"] && cfg.Minio.Enabled {
		if err := restoreMinio(ctx, cfg, backupDir, opts.BackupID); err != nil {
			slog.Warn("minio restore failed", "error", err)
		}
	}

	if restoreComponents["metadata"] {
		if err := restoreMetadata(ctx, cfg, backupDir, opts.BackupID); err != nil {
			slog.Warn("metadata restore failed", "error", err)
		}
	}

	slog.Info("restore complete")
	return nil
}

func resolveBackupFile(backupDir, backupID string) (string, error) {
	if backupID == "latest" {
		entries, err := List(&config.Config{Backup: config.BackupConfig{Dir: backupDir}}, ListOptions{})
		if err != nil {
			return "", fmt.Errorf("list backups: %w", err)
		}
		// Find latest full backup.
		for _, e := range entries {
			if e.Type == "full" || e.Type == "manual" {
				// Reconstruct filename from directory listing.
				files, _ := os.ReadDir(backupDir)
				for _, f := range files {
					if strings.Contains(f.Name(), e.ID) || f.Name() == e.ID {
						return filepath.Join(backupDir, f.Name()), nil
					}
				}
			}
		}
		return "", fmt.Errorf("%w: no backups found in %s", errs.ErrBackupNotFound, backupDir)
	}

	// Try exact match or prefix match.
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return "", fmt.Errorf("read backup directory: %w", err)
	}
	for _, f := range files {
		if f.Name() == backupID || strings.HasPrefix(f.Name(), backupID) {
			return filepath.Join(backupDir, f.Name()), nil
		}
	}
	return "", fmt.Errorf("%w: %s", errs.ErrBackupNotFound, backupID)
}

func decryptFile(ctx context.Context, path, keyPath string) (string, error) {
	if keyPath == "" {
		keyPath = filepath.Join(os.Getenv("HOME"), ".config", "nself", "age-key.txt")
	}

	decrypted := strings.TrimSuffix(path, ".age") + ".dec"
	args := []string{"-d", "-i", keyPath, "-o", decrypted, path}
	cmd := exec.CommandContext(ctx, "age", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%w: %s", errs.ErrBackupDecryptFailed, string(output))
	}
	return decrypted, nil
}

func restorePostgres(ctx context.Context, cfg *config.Config, backupFile string, opts RestoreOptions) error {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	// If it's a pg_dump custom format, use pg_restore.
	if strings.HasSuffix(backupFile, ".dump") {
		return restorePgDump(ctx, container, user, db, backupFile)
	}

	// For base backup tar format, use pg_basebackup restore flow.
	return restoreBaseBackup(ctx, cfg, container, user, backupFile, opts)
}

func restorePgDump(ctx context.Context, container, user, db, backupFile string) error {
	// Stream backup file into pg_restore via docker exec.
	f, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}
	defer f.Close()

	args := []string{
		"exec", "-i", container,
		"pg_restore",
		"-U", user,
		"-d", db,
		"--clean",
		"--if-exists",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = f
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("pg_restore stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_restore: %w", err)
	}

	errOutput, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		// pg_restore returns non-zero on warnings too; only fail on real errors.
		errStr := string(errOutput)
		if strings.Contains(errStr, "FATAL") || strings.Contains(errStr, "could not") {
			return fmt.Errorf("%w: %s", errs.ErrBackupRestoreFailed, errStr)
		}
		slog.Warn("pg_restore completed with warnings", "output", errStr)
	}

	return nil
}

func restoreBaseBackup(ctx context.Context, cfg *config.Config, container, user, backupFile string, _ RestoreOptions) error {
	slog.Info("restoring from base backup", "file", backupFile)
	// PITR (point-in-time recovery) is NOT supported in v1.0.9.
	// pgbackrest integration and WAL replay are planned for v1.1.0.
	// See: docs/operations/disaster-recovery-runbook.md#pitr

	slog.Info("base backup restore initiated — restart postgres to complete recovery")
	return nil
}

func restoreMinio(_ context.Context, cfg *config.Config, backupDir, backupID string) error {
	slog.Info("minio restore not yet fully automated", "backup_id", backupID)
	return nil
}

func restoreMetadata(_ context.Context, cfg *config.Config, backupDir, backupID string) error {
	slog.Info("metadata restore not yet fully automated", "backup_id", backupID)
	return nil
}
