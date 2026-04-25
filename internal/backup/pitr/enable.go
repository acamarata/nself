package pitr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/config"
)

// EnableConfig holds the PITR enable options.
type EnableConfig struct {
	Destination       string // remote destination URL (s3://, r2://, etc.)
	EncryptRecipient  string // age public key recipient for encryption
	RetentionDays     int    // WAL retention window in days
	WALArchiveTimeout int    // archive_timeout in seconds
}

// Enable writes the PostgreSQL configuration snippet that activates WAL
// archiving. The snippet is written to:
//
//	.nself/pitr/postgresql.conf.d/pitr.conf
//
// Mount this file into the PostgreSQL container to activate PITR.
func Enable(cfg *config.Config, pitrCfg EnableConfig) error {
	if pitrCfg.RetentionDays <= 0 {
		pitrCfg.RetentionDays = 7
	}
	if pitrCfg.WALArchiveTimeout <= 0 {
		pitrCfg.WALArchiveTimeout = 60
	}

	confDir := filepath.Join(".nself", "pitr", "postgresql.conf.d")
	if err := os.MkdirAll(confDir, 0700); err != nil {
		return fmt.Errorf("create PITR config directory %s: %w", confDir, err)
	}

	confPath := filepath.Join(confDir, "pitr.conf")
	content := generatePostgresConf(pitrCfg)

	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("write PITR config %s: %w", confPath, err)
	}
	if err := os.Chmod(confPath, 0600); err != nil {
		return fmt.Errorf("chmod PITR config %s: %w", confPath, err)
	}

	// Write a nself PITR env file so the WAL archive helper knows the destination.
	envPath := filepath.Join(".nself", "pitr", "pitr.env")
	envContent := fmt.Sprintf("NSELF_PITR_ENABLED=true\nNSELF_PITR_DESTINATION=%s\nNSELF_PITR_RETENTION_DAYS=%d\n",
		pitrCfg.Destination, pitrCfg.RetentionDays)
	if pitrCfg.EncryptRecipient != "" {
		envContent += "NSELF_PITR_ENCRYPT_RECIPIENT=" + pitrCfg.EncryptRecipient + "\n"
	}
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("write PITR env file %s: %w", envPath, err)
	}
	if err := os.Chmod(envPath, 0600); err != nil {
		return fmt.Errorf("chmod PITR env file %s: %w", envPath, err)
	}

	_ = cfg // reserved for future use (e.g. writing env vars back to project config)
	return nil
}

// Disable removes the PITR configuration snippet so that WAL archiving stops
// on the next container restart.
func Disable(cfg *config.Config) error {
	confPath := filepath.Join(".nself", "pitr", "postgresql.conf.d", "pitr.conf")
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PITR config %s: %w", confPath, err)
	}
	envPath := filepath.Join(".nself", "pitr", "pitr.env")
	if err := os.Remove(envPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PITR env %s: %w", envPath, err)
	}
	_ = cfg
	return nil
}

// generatePostgresConf returns the postgresql.conf snippet for WAL archiving.
// It sets wal_level=logical (superset of replica; satisfies both PITR and CDC).
func generatePostgresConf(pitrCfg EnableConfig) string {
	archiveCmd := fmt.Sprintf("test ! -f /var/lib/postgresql/wal_archive/%%f && cp %%p /var/lib/postgresql/wal_archive/%%f")
	return fmt.Sprintf(`# PITR configuration — managed by nself
# Source: nself pitr enable --to %s
wal_level = logical
archive_mode = on
archive_command = '%s'
archive_timeout = %d
`, pitrCfg.Destination, archiveCmd, pitrCfg.WALArchiveTimeout)
}
