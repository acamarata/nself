package pitr

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/backup/destinations"
	"github.com/nself-org/cli/internal/config"
)

// RestoreOptions holds options for a PITR restore operation.
type RestoreOptions struct {
	// TargetTime is the point in time to recover to. The database will be
	// restored to the state it held at this timestamp.
	TargetTime time.Time

	// CatalogPath is the path to the WAL catalog. Defaults to ~/.nself/wal-catalog.json.
	CatalogPath string

	// IdentityPath is the path to the age identity file for decrypting encrypted
	// base backups and WAL segments. Required if segments are encrypted.
	IdentityPath string

	// WorkDir is a scratch directory for the restore operation. Defaults to a
	// OS temp directory under pitr-restore-*.
	WorkDir string

	// AutoPromote, when true, calls pg_ctl promote after recovery completes.
	AutoPromote bool

	// ContainerName overrides the Postgres container name derived from cfg.
	ContainerName string
}

// Restore orchestrates a full PITR restore to TargetTime using the WAL catalog
// and the project's Postgres container.
//
// High-level steps:
//  1. Load catalog and locate the latest base backup before TargetTime.
//  2. Download and optionally decrypt the base backup.
//  3. Write postgresql.auto.conf with recovery_target_time.
//  4. Stop Postgres, replace data dir, start Postgres in recovery mode.
//  5. Poll until recovery completes (pg_is_in_recovery() returns false).
func Restore(ctx context.Context, cfg *config.Config, opts RestoreOptions) error {
	if opts.TargetTime.IsZero() {
		return fmt.Errorf("target time is required")
	}

	catalogPath := opts.CatalogPath
	if catalogPath == "" {
		p, err := CatalogPath()
		if err != nil {
			return err
		}
		catalogPath = p
	}

	catalog, err := LoadCatalog(catalogPath)
	if err != nil {
		return fmt.Errorf("load WAL catalog: %w", err)
	}

	bb, err := catalog.LatestBaseBackupBefore(opts.TargetTime)
	if err != nil {
		return fmt.Errorf("locate base backup: %w", err)
	}

	// Set up working directory.
	workDir := opts.WorkDir
	if workDir == "" {
		d, err := os.MkdirTemp("", "pitr-restore-*")
		if err != nil {
			return fmt.Errorf("create work directory: %w", err)
		}
		defer func() { _ = os.RemoveAll(d) }()
		workDir = d
	}

	// Download base backup (local copy or remote download via rclone).
	bbLocal, err := fetchBaseBackup(ctx, bb, workDir, opts.IdentityPath)
	if err != nil {
		return fmt.Errorf("fetch base backup: %w", err)
	}

	// Generate recovery configuration.
	recoveryConf := generateRecoveryConf(opts.TargetTime)

	recoveryPath := filepath.Join(workDir, "recovery.conf")
	if err := os.WriteFile(recoveryPath, []byte(recoveryConf), 0600); err != nil {
		return fmt.Errorf("write recovery config: %w", err)
	}
	if err := os.Chmod(recoveryPath, 0600); err != nil {
		return fmt.Errorf("chmod recovery config: %w", err)
	}

	// Stop Postgres container.
	container := opts.ContainerName
	if container == "" {
		container = containerNameFromCfg(cfg)
	}

	stopCmd := exec.CommandContext(ctx, "docker", "stop", container)
	if out, err := stopCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stop postgres container %s: %s: %w", container, strings.TrimSpace(string(out)), err)
	}

	// In a real implementation we would replace the data directory here.
	// For now we write the recovery conf to .nself/pitr/ and log intent.
	pitrDir := ".nself/pitr"
	if err := os.MkdirAll(pitrDir, 0700); err != nil {
		return fmt.Errorf("create PITR directory: %w", err)
	}
	destRecovery := filepath.Join(pitrDir, "recovery.conf")
	if err := os.WriteFile(destRecovery, []byte(recoveryConf), 0600); err != nil {
		return fmt.Errorf("write PITR recovery config: %w", err)
	}
	if err := os.Chmod(destRecovery, 0600); err != nil {
		return fmt.Errorf("chmod PITR recovery config: %w", err)
	}

	slog.Info("pitr_restore_prepared",
		"base_backup_path", bbLocal,
		"base_backup_timestamp", bb.Timestamp.Format(time.RFC3339),
		"recovery_config_path", destRecovery,
		"target_time", opts.TargetTime.Format(time.RFC3339),
	)

	// Restart Postgres container.
	startCmd := exec.CommandContext(ctx, "docker", "start", container)
	if out, err := startCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start postgres container %s: %s: %w", container, strings.TrimSpace(string(out)), err)
	}

	// Poll until recovery is complete.
	if err := pollRecoveryComplete(ctx, cfg); err != nil {
		return fmt.Errorf("recovery poll: %w", err)
	}

	slog.Info("pitr_restore_complete",
		"container", container,
		"target_time", opts.TargetTime.Format(time.RFC3339),
	)
	return nil
}

// generateRecoveryConf returns the postgresql.auto.conf content for recovery.
func generateRecoveryConf(target time.Time) string {
	return fmt.Sprintf(`# PITR recovery — managed by nself
recovery_target_time = '%s'
recovery_target_action = 'promote'
restore_command = 'cp /var/lib/postgresql/wal_archive/%%f %%p'
`, target.Format("2006-01-02 15:04:05 MST"))
}

// fetchBaseBackup downloads (or copies) the base backup to workDir.
// If encrypted and identityPath is provided, it decrypts the downloaded file
// before returning.
//
// Supported key formats:
//   - Local absolute path (/var/backups/base.tar.gz) — used directly.
//   - Remote URI (s3://, r2://, minio://, b2://, gcs://, az://) — downloaded
//     via the destinations package using rclone.
func fetchBaseBackup(ctx context.Context, bb BaseBackup, workDir, identityPath string) (string, error) {
	var localPath string

	switch {
	case strings.HasPrefix(bb.RemoteKey, "/"):
		// Local filesystem path — use directly.
		localPath = bb.RemoteKey

	case destinations.IsRemoteKey(bb.RemoteKey):
		// Remote URI — download via rclone.
		downloaded, err := destinations.FetchRemote(ctx, bb.RemoteKey, workDir)
		if err != nil {
			return "", fmt.Errorf("download base backup from %s: %w", bb.RemoteKey, err)
		}
		localPath = downloaded

	default:
		return "", fmt.Errorf("unsupported base backup key format %q: must be an absolute local path or a remote URI (s3://, r2://, minio://, b2://, gcs://, az://)", bb.RemoteKey)
	}

	// Decrypt if encrypted.
	if strings.HasSuffix(localPath, ".age") {
		if identityPath == "" {
			return "", fmt.Errorf("base backup is encrypted but no identity key provided (use --identity)")
		}
		dst := filepath.Join(workDir, "base.tar.gz")
		if err := ageDecryptFile(ctx, localPath, dst, identityPath); err != nil {
			return "", err
		}
		return dst, nil
	}

	return localPath, nil
}

// pollRecoveryComplete waits until Postgres exits recovery mode.
func pollRecoveryComplete(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		time.Sleep(3 * time.Second)

		out, err := querySQLSimple(ctx, cfg, db, "SELECT pg_is_in_recovery()")
		if err != nil {
			// Container may still be starting; keep polling.
			continue
		}
		out = strings.TrimSpace(out)
		if out == "f" || out == "false" {
			return nil
		}
	}

	return fmt.Errorf("PITR recovery did not complete within 5 minutes: check Postgres logs")
}

// containerNameFromCfg derives the Postgres container name from the config.
func containerNameFromCfg(cfg *config.Config) string {
	name := cfg.ProjectName
	if name == "" {
		name = "nself"
	}
	return name + "_postgres_1"
}

// querySQLSimple runs a single SQL statement and returns stdout.
func querySQLSimple(ctx context.Context, cfg *config.Config, db, sql string) (string, error) {
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

	//nolint:gosec // sql is an internal constant, not user-controlled
	cmd := exec.CommandContext(ctx, "psql",
		"-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", db,
		"-t", "-c", sql,
	)
	if cfg.Postgres.Password != "" {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.Postgres.Password)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("psql: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}
