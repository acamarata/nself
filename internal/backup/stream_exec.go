package backup

// stream_exec.go — subprocess helpers for the streaming backup pipeline.
//
// Purpose: run the pg_dump, age-encrypt and rclone-rcat legs of the streaming pipeline that Stream (stream.go) wires together, split out for file size.
// Inputs: a context, the resolved StreamConfig fields and the pipe endpoints connecting the three legs.
// Outputs: the started *exec.Cmd for each leg, or an error if the binary is missing or fails to start.
// Constraints: pure move from stream.go (CLI-R12 Batch E); no behaviour change. Keep in sync with runStreamPipeline in stream.go, which calls these in order.

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// runPgDump executes pg_dump and writes its output to w.
// Uses pg_dump custom format for efficient streaming and restore.
func runPgDump(ctx context.Context, cfg *config.Config, pgURL string, w io.Writer) error {
	args := []string{
		"--format=custom",
		"--no-password",
		pgURL,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Stdout = w
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pg_dump: %w", err)
	}

	errOut, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		if len(errOut) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupFailed, strings.TrimSpace(string(errOut)))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupFailed, err)
	}

	return nil
}

// ageEncryptStream pipes r through the age binary and writes ciphertext to w.
// Supports multiple recipients (age public keys, SSH public keys).
func ageEncryptStream(ctx context.Context, r io.Reader, w io.Writer, recipients []string) error {
	args := []string{}
	for _, rec := range recipients {
		args = append(args, "-r", rec)
	}

	cmd := exec.CommandContext(ctx, "age", args...)
	cmd.Stdin = r
	cmd.Stdout = w

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("age stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start age: %w", err)
	}

	errOut, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		if len(errOut) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupEncryptFailed, strings.TrimSpace(string(errOut)))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupEncryptFailed, err)
	}

	return nil
}

// rcloneRcat uploads from r to destination/key using rclone rcat.
// rclone handles multipart uploads internally for all supported backends.
func rcloneRcat(ctx context.Context, r io.Reader, destination, key string) error {
	remote := destination
	if !strings.HasSuffix(remote, "/") {
		remote = remote + "/"
	}
	remote = remote + key

	cmd := exec.CommandContext(ctx, "rclone", "rcat", remote)
	cmd.Stdin = r

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("rclone stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start rclone: %w", err)
	}

	errOut, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		if len(errOut) > 0 {
			return fmt.Errorf("%w: %s", errs.ErrBackupRemoteFailed, strings.TrimSpace(string(errOut)))
		}
		return fmt.Errorf("%w: %v", errs.ErrBackupRemoteFailed, err)
	}

	return nil
}
