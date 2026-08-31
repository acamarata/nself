package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/errs"
)

// ArchiveWALOptions holds configuration for WAL file archiving.
type ArchiveWALOptions struct {
	WALFile       string // path to local WAL file
	Remote        string // rclone remote path
	AgeRecipient  string // age public key for encryption
	Timeline      string // PG timeline ID
	MaxRetries    int    // max upload retries (default: 3)
	RetryInterval time.Duration
}

// ArchiveWAL encrypts and uploads a single WAL file to the remote.
// This is the core logic for the nself-backup-archive helper binary.
func ArchiveWAL(ctx context.Context, opts ArchiveWALOptions) error {
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.RetryInterval == 0 {
		opts.RetryInterval = 5 * time.Second
	}

	walBase := filepath.Base(opts.WALFile)

	// Validate WAL file exists.
	info, err := os.Stat(opts.WALFile)
	if err != nil {
		return fmt.Errorf("%w: WAL file not found: %v", errs.ErrWALArchiveFailed, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%w: WAL file is empty: %s", errs.ErrWALArchiveFailed, opts.WALFile)
	}

	uploadPath := opts.WALFile

	// Encrypt if age recipient is configured.
	if opts.AgeRecipient != "" {
		encPath := opts.WALFile + ".age"
		args := []string{"-r", opts.AgeRecipient, "-o", encPath, opts.WALFile}
		cmd := exec.CommandContext(ctx, "age", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%w: age encrypt failed: %s", errs.ErrWALArchiveFailed, string(output))
		}
		uploadPath = encPath
		defer func() { _ = os.Remove(encPath) }()
		walBase = walBase + ".age"
	}

	// Determine remote destination.
	remoteDest := fmt.Sprintf("%s/wal/%s/%s", opts.Remote, opts.Timeline, walBase)

	// Upload with retries.
	var lastErr error
	for attempt := 1; attempt <= opts.MaxRetries; attempt++ {
		slog.Info("uploading WAL", "file", walBase, "attempt", attempt, "dest", remoteDest)

		args := []string{"copyto", uploadPath, remoteDest}
		cmd := exec.CommandContext(ctx, "rclone", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("rclone upload attempt %d: %s: %w", attempt, string(output), err)
			slog.Warn("WAL upload failed", "attempt", attempt, "error", lastErr)

			if attempt < opts.MaxRetries {
				backoff := opts.RetryInterval * time.Duration(attempt)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
			}
			continue
		}

		// Verify upload with HEAD/stat.
		verifyArgs := []string{"lsf", remoteDest}
		verifyCmd := exec.CommandContext(ctx, "rclone", verifyArgs...)
		if output, err := verifyCmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("rclone verify failed: %s: %w", string(output), err)
			slog.Warn("WAL upload verification failed", "attempt", attempt, "error", lastErr)
			continue
		}

		slog.Info("WAL archived successfully", "file", walBase, "dest", remoteDest)
		return nil
	}

	return fmt.Errorf("%w: all %d attempts failed: %v", errs.ErrWALArchiveFailed, opts.MaxRetries, lastErr)
}
