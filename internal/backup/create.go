// Package backup provides backup creation, listing, restoration, verification,
// pruning, and WAL archiving for nSelf projects.
package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/metrics"
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
