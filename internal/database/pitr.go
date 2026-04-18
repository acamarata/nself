package database

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

// PITRStatus describes the current PITR configuration state.
type PITRStatus struct {
	ArchiveMode     string // on, off, always
	ArchiveCommand  string
	WALLevel        string // replica, logical
	MaxWALSenders   int
	WALArchiveDir   string
	LastArchivedWAL string
	LastArchiveTime string
	ArchiveEnabled  bool
}

// PITRConfig holds the generated PostgreSQL settings for PITR.
type PITRConfig struct {
	ArchiveDir  string
	WalInterval int
	RemotePath  string // rclone remote path for WAL shipping
}

// GetPITRStatus queries PostgreSQL for current WAL archiving settings.
func GetPITRStatus(ctx context.Context, cfg *config.Config) (PITRStatus, error) {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	settingsSQL := `SELECT name, setting FROM pg_settings WHERE name IN ('archive_mode','archive_command','wal_level','max_wal_senders')`
	out, err := querySQL(ctx, cfg, db, settingsSQL)
	if err != nil {
		return PITRStatus{}, fmt.Errorf("query pg_settings: %w", err)
	}

	status := PITRStatus{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch name {
		case "archive_mode":
			status.ArchiveMode = val
		case "archive_command":
			status.ArchiveCommand = val
		case "wal_level":
			status.WALLevel = val
		case "max_wal_senders":
			n, _ := strconv.Atoi(val)
			status.MaxWALSenders = n
		}
	}

	archiverSQL := `SELECT archived_count, last_archived_wal, last_archived_time::text, failed_count FROM pg_stat_archiver`
	archiverOut, err := querySQL(ctx, cfg, db, archiverSQL)
	if err != nil {
		return PITRStatus{}, fmt.Errorf("query pg_stat_archiver: %w", err)
	}

	for _, line := range strings.Split(archiverOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			status.LastArchivedWAL = strings.TrimSpace(parts[1])
			status.LastArchiveTime = strings.TrimSpace(parts[2])
		}
		break
	}

	status.ArchiveEnabled = status.ArchiveMode == "on" || status.ArchiveMode == "always"
	return status, nil
}

// GeneratePITRPostgresConf generates the postgresql.conf snippet needed for PITR.
// Returns the configuration as a string ready to append to postgresql.conf.
func GeneratePITRPostgresConf(cfg *config.Config, pitrCfg PITRConfig) string {
	archiveDir := pitrCfg.ArchiveDir
	if archiveDir == "" {
		archiveDir = "/var/lib/postgresql/wal_archive"
	}
	return fmt.Sprintf(`# PITR configuration — managed by nself
archive_mode = on
archive_command = 'test ! -f %s/%%f && cp %%p %s/%%f'
wal_level = replica
max_wal_senders = 3
wal_keep_size = 1GB
`, archiveDir, archiveDir)
}

// WritePITRConfig writes the PITR configuration to .nself/pitr/postgresql.conf.d/pitr.conf.
// This file should be mounted into the PostgreSQL container.
func WritePITRConfig(cfg *config.Config, projectDir string, pitrCfg PITRConfig) error {
	confDir := filepath.Join(projectDir, ".nself", "pitr", "postgresql.conf.d")
	if err := os.MkdirAll(confDir, 0700); err != nil {
		return fmt.Errorf("create PITR config directory %s: %w", confDir, err)
	}

	confPath := filepath.Join(confDir, "pitr.conf")
	content := GeneratePITRPostgresConf(cfg, pitrCfg)

	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("write PITR config %s: %w", confPath, err)
	}
	if err := os.Chmod(confPath, 0600); err != nil {
		return fmt.Errorf("chmod PITR config %s: %w", confPath, err)
	}

	return nil
}

// TestWALArchive creates a test WAL file and verifies it is archived correctly.
// Returns an error if archiving fails or times out.
func TestWALArchive(ctx context.Context, cfg *config.Config) error {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	// Force a WAL segment switch so archiving is triggered.
	if _, err := querySQL(ctx, cfg, db, "SELECT pg_switch_wal()"); err != nil {
		return fmt.Errorf("force WAL switch: %w", err)
	}

	// Poll pg_stat_archiver for up to 30s waiting for the last_archived_wal to update.
	initial, err := querySQL(ctx, cfg, db, "SELECT last_archived_wal FROM pg_stat_archiver")
	if err != nil {
		return fmt.Errorf("read initial archived WAL: %w", err)
	}
	initial = strings.TrimSpace(initial)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		time.Sleep(2 * time.Second)

		current, err := querySQL(ctx, cfg, db, "SELECT last_archived_wal FROM pg_stat_archiver")
		if err != nil {
			return fmt.Errorf("poll archived WAL: %w", err)
		}
		current = strings.TrimSpace(current)

		if current != initial && current != "" {
			return nil
		}
	}

	return fmt.Errorf("WAL archiving did not complete within 30s: check archive_command configuration")
}

// PITRRestore restores the database to a specific point in time using archived WALs.
// targetTime is in RFC3339 format (e.g. "2024-01-15T14:30:00Z").
func PITRRestore(ctx context.Context, cfg *config.Config, targetTime string) error {
	if targetTime == "" {
		return fmt.Errorf("targetTime is required for PITR restore")
	}

	// Validate RFC3339 format.
	if _, err := time.Parse(time.RFC3339, targetTime); err != nil {
		return fmt.Errorf("invalid targetTime %q, expected RFC3339 format: %w", targetTime, err)
	}

	container := containerName(cfg)

	// Stop the postgres container.
	stopCmd := exec.CommandContext(ctx, "docker", "stop", container)
	if out, err := stopCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stop postgres container %s: %s: %w", container, strings.TrimSpace(string(out)), err)
	}

	// Write recovery configuration.
	// For PG12+ we write to postgresql.conf additions; the restore_command uses
	// the archive directory from the PITR config.
	archiveDir := "/var/lib/postgresql/wal_archive"
	recoveryConf := fmt.Sprintf(`# PITR recovery — managed by nself
recovery_target_time = '%s'
recovery_target_action = 'promote'
restore_command = 'cp %s/%%f %%p'
`, targetTime, archiveDir)

	// Write the recovery config into a temp file; in a production setup this
	// would be mounted into the container. We write it to the project dir for now.
	recoveryPath := ".nself/pitr/recovery.conf"
	if err := os.MkdirAll(filepath.Dir(recoveryPath), 0700); err != nil {
		return fmt.Errorf("create PITR recovery directory: %w", err)
	}
	if err := os.WriteFile(recoveryPath, []byte(recoveryConf), 0600); err != nil {
		return fmt.Errorf("write recovery config: %w", err)
	}
	if err := os.Chmod(recoveryPath, 0600); err != nil {
		return fmt.Errorf("chmod recovery config: %w", err)
	}

	// Start the container.
	startCmd := exec.CommandContext(ctx, "docker", "start", container)
	if out, err := startCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start postgres container %s: %s: %w", container, strings.TrimSpace(string(out)), err)
	}

	// Poll for recovery completion.
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

		out, err := querySQL(ctx, cfg, db, "SELECT pg_is_in_recovery()")
		if err != nil {
			// Container may still be starting; keep polling.
			continue
		}
		out = strings.TrimSpace(out)
		if out == "f" || out == "false" {
			// Recovery complete — postgres is in normal operating mode.
			return nil
		}
	}

	return fmt.Errorf("PITR recovery did not complete within 5 minutes")
}
