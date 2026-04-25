package pitr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)


// TakeBaseBackup runs pg_basebackup against the project's Postgres container
// and stores the resulting tar archive at destDir. If encryptRecipient is
// non-empty the archive is encrypted with age before writing.
//
// Returns the path to the written archive file and the backup timestamp.
func TakeBaseBackup(ctx context.Context, cfg *config.Config, destDir, encryptRecipient string) (string, time.Time, error) {
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return "", time.Time{}, fmt.Errorf("create base backup directory %s: %w", destDir, err)
	}

	ts := time.Now().UTC()
	stamp := ts.Format("20060102T150405")
	tarName := "base-" + stamp + ".tar"
	tarPath := filepath.Join(destDir, tarName)

	host := cfg.Postgres.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Postgres.Port
	if port == 0 {
		port = 5432
	}
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	// pg_basebackup writes a tar-format backup to tarPath.
	//nolint:gosec // user-supplied config values, not user input
	pbArgs := []string{
		"-h", host,
		"-p", strconv.Itoa(port),
		"-U", user,
		"-F", "t",  // tar format
		"-z",       // gzip compress
		"-X", "stream", // stream WAL during backup
		"--checkpoint=fast",
		"-f", tarPath,
	}

	pbCmd := exec.CommandContext(ctx, "pg_basebackup", pbArgs...)
	if cfg.Postgres.Password != "" {
		pbCmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.Postgres.Password)
	}

	out, err := pbCmd.CombinedOutput()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("pg_basebackup failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	finalPath := tarPath
	if encryptRecipient != "" {
		encPath := tarPath + ".age"
		if err := ageEncryptFile(ctx, tarPath, encPath, encryptRecipient); err != nil {
			_ = os.Remove(tarPath)
			return "", time.Time{}, fmt.Errorf("encrypt base backup: %w", err)
		}
		_ = os.Remove(tarPath) // remove plaintext
		finalPath = encPath
	}

	return finalPath, ts, nil
}

// ageEncryptFile encrypts src to dst using the age CLI tool with the given
// recipient public key. src is removed on success; the caller must handle
// errors and cleanup.
func ageEncryptFile(ctx context.Context, src, dst, recipient string) error {
	//nolint:gosec // recipient is a config value, not user-controlled at runtime
	cmd := exec.CommandContext(ctx, "age", "-r", recipient, "-o", dst, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("age encrypt: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// ageDecryptFile decrypts src to dst using the age CLI tool with the given
// identity key file.
func ageDecryptFile(ctx context.Context, src, dst, identityPath string) error {
	//nolint:gosec // identityPath is a config value
	cmd := exec.CommandContext(ctx, "age", "-d", "-i", identityPath, "-o", dst, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("age decrypt: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
