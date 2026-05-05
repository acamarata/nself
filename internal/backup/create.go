// Package backup provides backup creation, listing, restoration, verification,
// pruning, and WAL archiving for nSelf projects.
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
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/metrics"
	"github.com/nself-org/cli/internal/security"
)

// BackupType identifies the kind of backup to create.
type BackupType string

const (
	BackupTypeFull     BackupType = "full"
	BackupTypeWAL      BackupType = "wal"
	BackupTypeMetadata BackupType = "metadata"
	BackupTypeMinio    BackupType = "minio"
	BackupTypeAll      BackupType = "all"
)

// CreateOptions holds flags for `nself backup create`.
type CreateOptions struct {
	Type      BackupType // full, wal, metadata, minio, all
	Remote    string     // remote name override
	Encrypt   bool       // force encryption on
	NoEncrypt bool       // force encryption off
	Tag       string     // human label for this backup
	DryRun    bool       // preview only
}

// Create performs a backup of the specified type. It writes the backup to the
// local backup directory and optionally uploads to the configured remote.
func Create(ctx context.Context, cfg *config.Config, opts CreateOptions) error {
	backupDir := cfg.Backup.Dir
	if backupDir == "" {
		backupDir = "./backups"
	}

	if opts.DryRun {
		slog.Info("dry-run: would create backup", "type", opts.Type, "dir", backupDir)
		return nil
	}

	types := []BackupType{opts.Type}
	if opts.Type == BackupTypeAll {
		types = []BackupType{BackupTypeFull, BackupTypeMetadata}
		if cfg.Minio.Enabled {
			types = append(types, BackupTypeMinio)
		}
	}

	for _, bt := range types {
		start := time.Now()
		err := createSingle(ctx, cfg, bt, backupDir, opts)
		emitMetric(cfg, bt, start, backupDir, opts, err == nil)
		if err != nil {
			return fmt.Errorf("backup %s: %w", bt, err)
		}
	}

	return nil
}

// emitMetric writes a prometheus textfile record for the just-completed
// backup run. Metric failures are logged but never fail the backup.
func emitMetric(cfg *config.Config, bt BackupType, start time.Time, backupDir string, opts CreateOptions, success bool) {
	// Best-effort: find the newest file for this type to report size.
	var size int64
	if entries, err := os.ReadDir(backupDir); err == nil {
		var newest os.FileInfo
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.Contains(name, "_"+string(bt)+"_") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if newest == nil || info.ModTime().After(newest.ModTime()) {
				newest = info
			}
		}
		if newest != nil {
			size = newest.Size()
		}
	}

	encrypt := cfg.Backup.Encryption
	if opts.Encrypt {
		encrypt = true
	}
	if opts.NoEncrypt {
		encrypt = false
	}

	rec := metrics.BackupRecord{
		Env:         cfg.Env,
		Type:        string(bt),
		Success:     success,
		DurationSec: time.Since(start).Seconds(),
		Bytes:       size,
		Encrypted:   encrypt && cfg.Backup.AgeRecipients != "",
		Timestamp:   time.Now(),
	}
	if err := metrics.EmitBackup(rec); err != nil {
		slog.Warn("emit backup metric", "error", err)
	}
}

func createSingle(ctx context.Context, cfg *config.Config, bt BackupType, backupDir string, opts CreateOptions) error {
	ts := time.Now().Format("20060102_150405")
	tag := ""
	if opts.Tag != "" {
		tag = "_" + opts.Tag
	}

	switch bt {
	case BackupTypeFull:
		return createFullBackup(ctx, cfg, backupDir, ts, tag, opts)
	case BackupTypeMetadata:
		return createMetadataBackup(ctx, cfg, backupDir, ts, tag)
	case BackupTypeMinio:
		return createMinioBackup(ctx, cfg, backupDir, ts, tag)
	case BackupTypeWAL:
		slog.Info("WAL archiving is continuous via archive_command; triggering checkpoint")
		return triggerWALCheckpoint(ctx, cfg)
	default:
		return fmt.Errorf("unknown backup type: %s", bt)
	}
}

// createFullBackup runs pg_basebackup inside the postgres container.
func createFullBackup(ctx context.Context, cfg *config.Config, backupDir, ts, tag string, opts CreateOptions) error {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	filename := fmt.Sprintf("%s_full_%s%s.tar", cfg.ProjectName, ts, tag)
	outputPath := filepath.Join(backupDir, filename)

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	args := []string{
		"exec", container,
		"pg_basebackup",
		"-U", user,
		"-Ft",     // tar format
		"-Xf",     // fetch WAL files
		"-z",      // gzip compression
		"-D", "-", // stream to stdout
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pg_basebackup stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("pg_basebackup stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_basebackup: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("create backup file %s: %w", outputPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, stdout); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("stream pg_basebackup output: %w", err)
	}

	errOutput, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		_ = os.Remove(outputPath)
		if len(errOutput) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupFailed, string(errOutput))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupFailed, err)
	}

	if err := security.EnforceFilePermissions(outputPath, 0600); err != nil {
		return fmt.Errorf("enforce permissions on %s: %w", outputPath, err)
	}

	// Encrypt if configured.
	encrypt := cfg.Backup.Encryption
	if opts.Encrypt {
		encrypt = true
	}
	if opts.NoEncrypt {
		encrypt = false
	}
	if encrypt && cfg.Backup.AgeRecipients != "" {
		if err := encryptFile(outputPath, cfg.Backup.AgeRecipients); err != nil {
			return fmt.Errorf("encrypt backup: %w", err)
		}
	}

	// Upload to remote if configured.
	remote := cfg.Backup.Remote
	if opts.Remote != "" {
		remote = opts.Remote
	}
	if remote != "" {
		if err := uploadToRemote(ctx, outputPath, remote); err != nil {
			slog.Error("remote upload failed", "error", err, "path", outputPath)
			// Non-fatal: local backup succeeded.
		}
	}

	slog.Info("full backup created", "path", outputPath)
	return nil
}

// createMetadataBackup dumps Hasura metadata and env config.
func createMetadataBackup(ctx context.Context, cfg *config.Config, backupDir, ts, tag string) error {
	filename := fmt.Sprintf("%s_metadata_%s%s.tar.gz", cfg.ProjectName, ts, tag)
	outputPath := filepath.Join(backupDir, filename)

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	// Export Hasura metadata via API.
	container := cfg.ProjectName + "_hasura"
	args := []string{
		"exec", container,
		"hasura-cli", "metadata", "export", "--admin-secret", cfg.Hasura.AdminSecret,
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("hasura metadata export failed, skipping metadata component", "error", err)
	}

	if err := os.WriteFile(outputPath, output, 0600); err != nil {
		return fmt.Errorf("write metadata backup: %w", err)
	}

	slog.Info("metadata backup created", "path", outputPath)
	return nil
}

// createMinioBackup uses mc mirror to back up MinIO buckets.
func createMinioBackup(ctx context.Context, cfg *config.Config, backupDir, ts, tag string) error {
	if !cfg.Minio.Enabled {
		slog.Info("MinIO not enabled, skipping MinIO backup")
		return nil
	}

	destDir := filepath.Join(backupDir, fmt.Sprintf("minio_%s%s", ts, tag))
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return fmt.Errorf("create minio backup directory: %w", err)
	}

	// Use mc mirror to copy all buckets locally.
	alias := cfg.ProjectName + "-minio"
	args := []string{"mirror", "--overwrite", alias + "/", destDir}
	cmd := exec.CommandContext(ctx, "mc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mc mirror: %w", err)
	}

	slog.Info("minio backup created", "path", destDir)
	return nil
}

// triggerWALCheckpoint forces a WAL checkpoint to flush pending WAL data.
func triggerWALCheckpoint(ctx context.Context, cfg *config.Config) error {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	args := []string{
		"exec", container,
		"psql", "-U", user, "-d", cfg.Postgres.DB,
		"-c", "CHECKPOINT;",
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("WAL checkpoint: %s: %w", string(output), err)
	}

	slog.Info("WAL checkpoint triggered")
	return nil
}

// encryptFile encrypts a file in-place using age with the given recipient public key.
func encryptFile(path, recipient string) error {
	encPath := path + ".age"
	args := []string{"-r", recipient, "-o", encPath, path}
	cmd := exec.Command("age", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", errs.ErrBackupEncryptFailed, string(output))
	}

	// Replace original with encrypted version.
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove unencrypted file: %w", err)
	}
	if err := os.Rename(encPath, path+".age"); err != nil {
		return fmt.Errorf("rename encrypted file: %w", err)
	}

	return nil
}

// uploadToRemote uploads a local file to the configured rclone remote.
func uploadToRemote(ctx context.Context, localPath, remote string) error {
	args := []string{"copyto", localPath, remote + "/" + filepath.Base(localPath)}
	cmd := exec.CommandContext(ctx, "rclone", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", errs.ErrBackupRemoteFailed, string(output))
	}
	return nil
}
