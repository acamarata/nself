package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/backup"
	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup operations: create, list, restore, verify, prune, config, status, init-key",
	Long: `Backup operations for nSelf projects.

Subcommands:
  create     Create a new backup (full, wal, metadata, minio, all)
  list       List available backups
  restore    Restore from a backup
  verify     Verify backup integrity
  prune      Remove old backups by retention policy
  config     View/set backup configuration
  status     Show backup subsystem status
  init-key   Generate age encryption keypair`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── backup create ──────────────────────────────────────────────────

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new backup",
	RunE:  runBackupCreate,
}

func runBackupCreate(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	backupType, _ := cmd.Flags().GetString("type")
	remote, _ := cmd.Flags().GetString("remote")
	encrypt, _ := cmd.Flags().GetBool("encrypt")
	noEncrypt, _ := cmd.Flags().GetBool("no-encrypt")
	tag, _ := cmd.Flags().GetString("tag")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	opts := backup.CreateOptions{
		Type:      backup.BackupType(backupType),
		Remote:    remote,
		Encrypt:   encrypt,
		NoEncrypt: noEncrypt,
		Tag:       tag,
		DryRun:    dryRun,
	}

	if err := backup.Create(cmd.Context(), cfg, opts); err != nil {
		return fmt.Errorf("backup create: %w", err)
	}
	fmt.Println("Backup created successfully.")
	return nil
}

// ── backup list ────────────────────────────────────────────────────

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	RunE:  runBackupList,
}

func runBackupList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	remote, _ := cmd.Flags().GetString("remote")
	env, _ := cmd.Flags().GetString("env")
	sinceStr, _ := cmd.Flags().GetString("since")
	format, _ := cmd.Flags().GetString("format")

	var since time.Duration
	if sinceStr != "" {
		since, err = time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since duration: %w", err)
		}
	}

	entries, err := backup.List(cfg, backup.ListOptions{
		Remote: remote,
		Env:    env,
		Since:  since,
		Format: format,
	})
	if err != nil {
		return fmt.Errorf("backup list: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	output, err := backup.FormatList(entries, format)
	if err != nil {
		return err
	}
	fmt.Print(output)
	return nil
}

// ── backup restore ─────────────────────────────────────────────────

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-id|latest>",
	Short: "Restore from a backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupRestore,
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	toDir, _ := cmd.Flags().GetString("to")
	onlyStr, _ := cmd.Flags().GetString("only")
	pitr, _ := cmd.Flags().GetString("point-in-time")
	decryptKey, _ := cmd.Flags().GetString("decrypt-key")
	yes, _ := cmd.Flags().GetBool("yes")

	var only []string
	if onlyStr != "" {
		only = strings.Split(onlyStr, ",")
	}

	if cfg.IsProduction() && !yes {
		if err := requireProductionConfirmation(cfg.ProjectName); err != nil {
			return err
		}
	}

	opts := backup.RestoreOptions{
		BackupID:    args[0],
		ToDir:       toDir,
		Only:        only,
		PointInTime: pitr,
		DecryptKey:  decryptKey,
		Yes:         yes,
	}

	if err := backup.Restore(cmd.Context(), cfg, opts); err != nil {
		return fmt.Errorf("backup restore: %w", err)
	}
	fmt.Println("Backup restored successfully.")
	return nil
}

// ── backup verify ──────────────────────────────────────────────────

var backupVerifyCmd = &cobra.Command{
	Use:   "verify <backup-id|latest>",
	Short: "Verify backup integrity",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupVerify,
}

func runBackupVerify(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	restoreTest, _ := cmd.Flags().GetBool("restore-test")
	cleanup, _ := cmd.Flags().GetBool("cleanup")
	keep, _ := cmd.Flags().GetBool("keep")

	result, err := backup.Verify(cmd.Context(), cfg, backup.VerifyOptions{
		BackupID:    args[0],
		RestoreTest: restoreTest,
		Cleanup:     cleanup,
		Keep:        keep,
	})
	if err != nil {
		return fmt.Errorf("backup verify: %w", err)
	}

	if result.Verified {
		fmt.Printf("Backup %s verified (%s, %s).\n", result.BackupID, result.Method, result.Duration)
	} else {
		fmt.Printf("Backup %s verification FAILED: %s\n", result.BackupID, result.Details)
	}
	return nil
}

// ── backup prune ───────────────────────────────────────────────────

var backupPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old backups by retention policy",
	RunE:  runBackupPrune,
}

func runBackupPrune(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	keepDaily, _ := cmd.Flags().GetInt("keep-daily")
	keepWeekly, _ := cmd.Flags().GetInt("keep-weekly")
	keepMonthly, _ := cmd.Flags().GetInt("keep-monthly")
	format, _ := cmd.Flags().GetString("format")

	result, err := backup.Prune(cfg, backup.PruneOptions{
		DryRun:      dryRun,
		KeepDaily:   keepDaily,
		KeepWeekly:  keepWeekly,
		KeepMonthly: keepMonthly,
	})
	if err != nil {
		return fmt.Errorf("backup prune: %w", err)
	}

	if format == "json" {
		return backup.FormatPruneJSON(result, keepDaily)
	}

	prefix := ""
	if result.DryRun {
		prefix = "(dry-run) "
	}
	fmt.Printf("%sKept %d backups, pruned %d.\n", prefix, len(result.Kept), len(result.Pruned))
	return nil
}

// ── backup config ──────────────────────────────────────────────────

var backupConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "View backup configuration",
	RunE:  runBackupConfig,
}

func runBackupConfig(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	installCron, _ := cmd.Flags().GetBool("install-cron")
	if installCron {
		fullAt, _ := cmd.Flags().GetString("full-at")
		walEvery, _ := cmd.Flags().GetString("wal-every")
		pruneAt, _ := cmd.Flags().GetString("prune-at")
		verifyOn, _ := cmd.Flags().GetString("verify-on")
		verifyAt, _ := cmd.Flags().GetString("verify-at")
		remote, _ := cmd.Flags().GetString("remote")
		unitDir, _ := cmd.Flags().GetString("unit-dir")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		opts := backup.SystemdInstallOptions{
			FullAt:      fullAt,
			WALEvery:    walEvery,
			PruneAt:     pruneAt,
			VerifyOnDay: verifyOn,
			VerifyAt:    verifyAt,
			Remote:      remote,
			UnitDir:     unitDir,
			DryRun:      dryRun,
		}
		if err := backup.InstallSystemdUnits(cfg, opts); err != nil {
			return fmt.Errorf("install-cron: %w", err)
		}
		if dryRun {
			return nil
		}
		fmt.Println("Systemd timers installed and enabled: nself-backup-{full,wal,prune,verify}.timer")
		return nil
	}

	format, _ := cmd.Flags().GetString("format")
	output, err := backup.ConfigView(cfg, format)
	if err != nil {
		return err
	}
	fmt.Print(output)
	return nil
}

// ── backup status ──────────────────────────────────────────────────

var backupStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show backup subsystem status",
	RunE:  runBackupStatus,
}

func runBackupStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	info, err := backup.Status(cfg)
	if err != nil {
		return fmt.Errorf("backup status: %w", err)
	}

	output, err := backup.FormatStatus(info, format)
	if err != nil {
		return err
	}
	fmt.Print(output)
	return nil
}

// ── backup init-key ────────────────────────────────────────────────

var backupInitKeyCmd = &cobra.Command{
	Use:   "init-key",
	Short: "Generate age encryption keypair for backups",
	RunE:  runBackupInitKey,
}

func runBackupInitKey(_ *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	pubKey, err := backup.InitKey(cfg)
	if err != nil {
		return fmt.Errorf("init-key: %w", err)
	}

	if pubKey != "" {
		fmt.Printf("Age keypair generated.\nPublic key: %s\n", pubKey)
		fmt.Println("Add to your .env: BACKUP_AGE_RECIPIENTS=" + pubKey)
	}
	return nil
}

// ── init ────────────────────────────────────────────────────────────

func init() {
	// backup create flags
	backupCreateCmd.Flags().String("type", "full", "Backup type: full, wal, metadata, minio, all")
	backupCreateCmd.Flags().String("remote", "", "Remote name override")
	backupCreateCmd.Flags().Bool("encrypt", false, "Force encryption on")
	backupCreateCmd.Flags().Bool("no-encrypt", false, "Force encryption off")
	backupCreateCmd.Flags().String("tag", "", "Human label for this backup")
	backupCreateCmd.Flags().Bool("dry-run", false, "Preview only")

	// backup list flags
	backupListCmd.Flags().String("remote", "", "Filter by remote name")
	backupListCmd.Flags().String("env", "", "Filter by environment")
	backupListCmd.Flags().String("since", "", "Show backups newer than this duration (e.g. 24h, 7d)")
	backupListCmd.Flags().String("format", "table", "Output format: table or json")

	// backup restore flags
	backupRestoreCmd.Flags().String("to", "", "Restore to alternate directory")
	backupRestoreCmd.Flags().String("only", "", "Restore subset: pg,minio,metadata (comma-separated)")
	backupRestoreCmd.Flags().String("point-in-time", "", "ISO8601 timestamp for PITR")
	backupRestoreCmd.Flags().String("decrypt-key", "", "Path to age identity file")
	backupRestoreCmd.Flags().Bool("yes", false, "Skip confirmation")

	// backup verify flags
	backupVerifyCmd.Flags().Bool("restore-test", false, "Spin up test container and restore")
	backupVerifyCmd.Flags().Bool("cleanup", true, "Remove test container after verify")
	backupVerifyCmd.Flags().Bool("keep", false, "Keep test container for inspection")

	// backup prune flags
	backupPruneCmd.Flags().Bool("dry-run", false, "Preview only")
	backupPruneCmd.Flags().Int("keep-daily", 7, "Keep last N daily backups")
	backupPruneCmd.Flags().Int("keep-weekly", 4, "Keep last N weekly backups")
	backupPruneCmd.Flags().Int("keep-monthly", 12, "Keep last N monthly backups")
	backupPruneCmd.Flags().String("format", "", "Output format: json")

	// backup config flags
	backupConfigCmd.Flags().String("format", "", "Output format: json")
	backupConfigCmd.Flags().Bool("install-cron", false, "Install systemd timers for backup/wal/prune/verify")
	backupConfigCmd.Flags().String("full-at", "03:00", "Full backup time UTC (HH:MM) when --install-cron")
	backupConfigCmd.Flags().String("wal-every", "15m", "WAL checkpoint interval when --install-cron")
	backupConfigCmd.Flags().String("prune-at", "04:00", "Prune time UTC (HH:MM) when --install-cron")
	backupConfigCmd.Flags().String("verify-on", "Sun", "Weekly restore-test day when --install-cron")
	backupConfigCmd.Flags().String("verify-at", "05:00", "Weekly restore-test time UTC (HH:MM) when --install-cron")
	backupConfigCmd.Flags().String("remote", "", "Override configured remote when --install-cron")
	backupConfigCmd.Flags().String("unit-dir", "/etc/systemd/system", "Systemd unit directory when --install-cron")
	backupConfigCmd.Flags().Bool("dry-run", false, "Print unit files without writing when --install-cron")

	// backup status flags
	backupStatusCmd.Flags().String("format", "", "Output format: json")

	// Wire subcommands
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupVerifyCmd)
	backupCmd.AddCommand(backupPruneCmd)
	backupCmd.AddCommand(backupConfigCmd)
	backupCmd.AddCommand(backupStatusCmd)
	backupCmd.AddCommand(backupInitKeyCmd)

	RootCmd.AddCommand(backupCmd)
}
