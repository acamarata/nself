package backup

// stream_restore_remote.go — RestoreFromRemote pipeline.
//
// Purpose: drive the end-to-end restore of a streamed backup from a remote destination, using the recipient/URL helpers in stream_restore.go, split out for file size.
// Inputs: the remote backup location and the target database connection info.
// Outputs: a restored database, or an error identifying which stage of the restore failed.
// Constraints: pure move from stream_restore.go (CLI-R12 Batch E); no behaviour change.

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// RestoreFromRemote decrypts and restores a backup directly from a remote URL.
// It streams: rclone cat | age -d | pg_restore. No local temp file.
func RestoreFromRemote(ctx context.Context, cfg *config.Config, from, keyPath string) error {
	if from == "" {
		return fmt.Errorf("--from destination required")
	}

	if keyPath == "" {
		keyPath = filepath.Join(os.Getenv("HOME"), ".config", "nself", "age-key.txt")
	}

	encrypted := strings.HasSuffix(from, ".age")

	// Check binaries.
	binaries := []string{"rclone", "pg_restore"}
	if encrypted {
		binaries = append(binaries, "age")
	}
	for _, bin := range binaries {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("required binary %q not found: %w", bin, err)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var downloadReader io.ReadCloser
	var restoreReader io.ReadCloser

	// Stage 1 output.
	dlR, dlW := io.Pipe()

	var wg sync.WaitGroup
	errc := make(chan error, 3)

	// ── Stage 1: rclone cat <remote> ─────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer dlW.Close()
		cmd := exec.CommandContext(ctx, "rclone", "cat", from)
		cmd.Stdout = dlW
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			errc <- fmt.Errorf("rclone stderr pipe: %w", err)
			return
		}
		if err := cmd.Start(); err != nil {
			cancel()
			errc <- fmt.Errorf("start rclone: %w", err)
			return
		}
		errOut, _ := io.ReadAll(stderr)
		if err := cmd.Wait(); err != nil {
			cancel()
			msg := strings.TrimSpace(string(errOut))
			if msg != "" {
				errc <- fmt.Errorf("%w: %s", errs.ErrBackupRemoteFailed, msg)
			} else {
				errc <- fmt.Errorf("%w: %v", errs.ErrBackupRemoteFailed, err)
			}
		}
	}()

	downloadReader = dlR

	// ── Stage 2: age -d (optional) ───────────────────────────────────
	var decR *io.PipeReader
	var decW *io.PipeWriter
	if encrypted {
		decR, decW = io.Pipe()
		restoreReader = decR
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer decW.Close()
			cmd := exec.CommandContext(ctx, "age", "--decrypt", "-i", keyPath)
			cmd.Stdin = downloadReader
			cmd.Stdout = decW
			stderr, err := cmd.StderrPipe()
			if err != nil {
				cancel()
				dlR.CloseWithError(err)
				errc <- fmt.Errorf("age stderr pipe: %w", err)
				return
			}
			if err := cmd.Start(); err != nil {
				cancel()
				dlR.CloseWithError(err)
				errc <- fmt.Errorf("start age: %w", err)
				return
			}
			errOut, _ := io.ReadAll(stderr)
			if err := cmd.Wait(); err != nil {
				cancel()
				msg := strings.TrimSpace(string(errOut))
				if msg != "" {
					errc <- fmt.Errorf("%w: %s", errs.ErrBackupDecryptFailed, msg)
				} else {
					errc <- fmt.Errorf("%w: %v", errs.ErrBackupDecryptFailed, err)
				}
			}
		}()
	} else {
		restoreReader = downloadReader
	}

	// ── Stage 3: pg_restore ───────────────────────────────────────────
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer restoreReader.Close()
		args := []string{
			"exec", "-i", container,
			"pg_restore",
			"-U", user,
			"-d", db,
			"--clean",
			"--if-exists",
		}
		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Stdin = restoreReader
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			errc <- fmt.Errorf("pg_restore stderr pipe: %w", err)
			return
		}
		if err := cmd.Start(); err != nil {
			cancel()
			errc <- fmt.Errorf("start pg_restore: %w", err)
			return
		}
		errOut, _ := io.ReadAll(stderr)
		if err := cmd.Wait(); err != nil {
			errStr := string(errOut)
			if strings.Contains(errStr, "FATAL") || strings.Contains(errStr, "could not") {
				cancel()
				errc <- fmt.Errorf("%w: %s", errs.ErrBackupRestoreFailed, strings.TrimSpace(errStr))
			} else if errStr != "" {
				slog.Warn("pg_restore warnings", "output", strings.TrimSpace(errStr))
			}
		}
	}()

	wg.Wait()
	close(errc)

	for e := range errc {
		if e != nil {
			return e
		}
	}

	slog.Info("remote restore complete", "from", from)
	return nil
}
