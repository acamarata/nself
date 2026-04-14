// nself-backup-archive is a helper binary used as PostgreSQL's archive_command.
// It encrypts and uploads WAL files to the configured remote storage.
//
// Usage (in postgresql.conf):
//
//	archive_command = '/usr/local/bin/nself-backup-archive %p %f'
//
// Environment variables:
//
//	BACKUP_REMOTE           — rclone remote path (required)
//	BACKUP_AGE_RECIPIENTS   — age public key for encryption (optional)
//	BACKUP_ARCHIVE_TIMELINE — PostgreSQL timeline ID (default: "1")
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nself-org/cli/internal/backup"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: nself-backup-archive <wal-path> <wal-filename>\n")
		os.Exit(1)
	}

	walPath := os.Args[1]
	// walFilename := os.Args[2] // used for logging, path is authoritative

	remote := os.Getenv("BACKUP_REMOTE")
	if remote == "" {
		slog.Error("BACKUP_REMOTE not set")
		os.Exit(1)
	}

	ageRecipient := os.Getenv("BACKUP_AGE_RECIPIENTS")
	timeline := os.Getenv("BACKUP_ARCHIVE_TIMELINE")
	if timeline == "" {
		timeline = "1"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	opts := backup.ArchiveWALOptions{
		WALFile:       walPath,
		Remote:        remote,
		AgeRecipient:  ageRecipient,
		Timeline:      timeline,
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
	}

	if err := backup.ArchiveWAL(ctx, opts); err != nil {
		slog.Error("WAL archive failed", "error", err, "file", walPath)
		// Log to file for consecutive failure tracking.
		logFailure(walPath, err)
		os.Exit(1)
	}
}

// logFailure appends to the WAL archive failure log.
func logFailure(walPath string, err error) {
	logFile := "/var/log/nself/wal-archive.log"
	f, ferr := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if ferr != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s FAIL %s: %v\n", time.Now().Format(time.RFC3339), walPath, err)
}
