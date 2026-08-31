// Package backup — streaming encrypted backup to remote destinations.
//
// The pipeline runs three concurrent goroutines:
//
//	pg_dump (streaming) | age (encrypt) | rclone (multipart upload)
//
// No temp files are written. Encryption recipients may be age public keys,
// SSH public keys, or GitHub user keys (fetched via github.com/users/<user>/keys).
// The upload destination is any rclone-supported remote: s3://, r2://, b2://, gcs://, az://.
package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// StreamConfig holds parameters for a streaming encrypted backup.
type StreamConfig struct {
	// PgURL is the postgres DSN used by pg_dump. If empty, derived from cfg.
	PgURL string

	// Destination is the rclone remote path, e.g. "s3:bucket/prefix/".
	Destination string

	// Recipients holds age/SSH public keys (one per entry). May also be
	// "github:<username>" to fetch the user's SSH public keys from GitHub.
	Recipients []string

	// Key is the object name appended to Destination. Auto-generated if empty.
	Key string

	// ChunkMB is the multipart chunk size. Defaults to 64.
	ChunkMB int
}

// StreamOptions holds CLI flag values for `nself backup stream`.
type StreamOptions struct {
	To         string   // destination URL (rclone remote path)
	Recipients []string // --recipient flags (may be specified multiple times)
	DryRun     bool
}

// StreamResult is returned by Stream on success.
type StreamResult struct {
	BackupID    string    `json:"backup_id"`
	Destination string    `json:"destination"`
	StartedAt   time.Time `json:"started_at"`
	Duration    string    `json:"duration"`
	Encrypted   bool      `json:"encrypted"`
}

// Stream runs a three-stage concurrent pipeline:
//
//  1. pg_dump stdout -> pgWriter
//  2. age encrypts pgReader -> encWriter (skipped if no recipients)
//  3. rclone rcat uploads encReader to Destination/Key
//
// All three stages run in parallel goroutines. The first error from any stage
// cancels the others via context cancellation.
func Stream(ctx context.Context, cfg *config.Config, opts StreamOptions) (*StreamResult, error) {
	if opts.To == "" {
		if cfg.Backup.Remote != "" {
			opts.To = cfg.Backup.Remote
		} else {
			return nil, fmt.Errorf("destination required: use --to <url> or set NSELF_BACKUP_DESTINATION")
		}
	}

	recipients := opts.Recipients
	if len(recipients) == 0 && cfg.Backup.AgeRecipients != "" {
		recipients = strings.Fields(cfg.Backup.AgeRecipients)
	}

	// Resolve GitHub keys.
	var err error
	recipients, err = resolveRecipients(ctx, recipients)
	if err != nil {
		return nil, fmt.Errorf("resolve recipients: %w", err)
	}

	// Build the object key.
	ts := time.Now().UTC().Format("20060102_150405")
	key := fmt.Sprintf("%s_stream_%s.sql", cfg.ProjectName, ts)
	if len(recipients) > 0 {
		key += ".age"
	}

	pgURL := buildPgURL(cfg)

	if opts.DryRun {
		slog.Info("dry-run: streaming backup",
			"destination", opts.To+"/"+key,
			"encrypted", len(recipients) > 0,
			"pg_url", redactURL(pgURL),
		)
		return &StreamResult{
			BackupID:    key,
			Destination: opts.To + "/" + key,
			StartedAt:   time.Now(),
			Duration:    "0s",
			Encrypted:   len(recipients) > 0,
		}, nil
	}

	start := time.Now()

	if err := checkBinaries(recipients); err != nil {
		return nil, err
	}

	result, err := runStreamPipeline(ctx, cfg, pgURL, opts.To, key, recipients)
	if err != nil {
		return nil, err
	}
	result.StartedAt = start
	result.Duration = time.Since(start).String()
	return result, nil
}

// runStreamPipeline wires the three concurrent goroutines and waits for all.
func runStreamPipeline(ctx context.Context, cfg *config.Config, pgURL, destination, key string, recipients []string) (*StreamResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Stage plumbing: two io.Pipe connections.
	//   pg_dump -> pgW | pgR -> (age) -> encW | encR -> rclone
	pgR, pgW := io.Pipe()
	var uploadReader io.ReadCloser

	encrypt := len(recipients) > 0
	var encW *io.PipeWriter
	var encR *io.PipeReader
	if encrypt {
		encR, encW = io.Pipe()
		uploadReader = encR
	} else {
		uploadReader = pgR
	}

	var wg sync.WaitGroup
	errc := make(chan error, 3)

	// ── Stage 1: pg_dump ─────────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { _ = pgW.Close() }()
		if err := runPgDump(ctx, cfg, pgURL, pgW); err != nil {
			cancel()
			errc <- fmt.Errorf("pg_dump: %w", err)
		}
	}()

	// ── Stage 2: age encrypt (optional) ──────────────────────────────
	if encrypt {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { _ = encW.Close() }()
			if err := ageEncryptStream(ctx, pgR, encW, recipients); err != nil {
				cancel()
				pgR.CloseWithError(err)
				errc <- fmt.Errorf("age encrypt: %w", err)
			}
		}()
	}

	// ── Stage 3: rclone rcat upload ───────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { _ = uploadReader.Close() }()
		if err := rcloneRcat(ctx, uploadReader, destination, key); err != nil {
			cancel()
			errc <- fmt.Errorf("rclone upload: %w", err)
		}
	}()

	wg.Wait()
	close(errc)

	for e := range errc {
		if e != nil {
			return nil, e
		}
	}

	full := destination
	if !strings.HasSuffix(destination, "/") {
		full = destination + "/" + key
	} else {
		full = destination + key
	}

	slog.Info("streaming backup complete", "destination", full, "encrypted", encrypt)

	return &StreamResult{
		BackupID:    key,
		Destination: full,
		Encrypted:   encrypt,
	}, nil
}
