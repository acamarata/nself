package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/database"
	"github.com/nself-org/cli/internal/docker"

	"github.com/spf13/cobra"
)

// migrationNameAllowed matches only lowercase alphanumeric characters, underscores, and hyphens.
var migrationNameAllowed = regexp.MustCompile(`^[a-z0-9_-]+$`)

// ── Parent command ──────────────────────────────────────────────────

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database operations: migrations, backups, restore, seed, shell",
	Long: `Database operations: migrations, backups, restore, seed, shell.

Subcommands:
  migrate   Manage database migrations (up/down/status/create)
  seed      Run seed data
  backup    Create pg_dump backup
  restore   Restore from backup
  shell     Open psql interactive shell
  reset     Drop and recreate database (DESTRUCTIVE)
  hasura    Hasura metadata operations`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── migrate ─────────────────────────────────────────────────────────

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var dbMigrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply pending migrations",
	RunE:  runDBMigrateUp,
}

var dbMigrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Revert last migration",
	RunE:  runDBMigrateDown,
}

var dbMigrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE:  runDBMigrateStatus,
}

var dbMigrateCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create new migration file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDBMigrateCreate,
}

// ── seed ────────────────────────────────────────────────────────────

var dbSeedCmd = &cobra.Command{
	Use:   "seed [file]",
	Short: "Run seed data",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDBSeed,
}

// ── backup ──────────────────────────────────────────────────────────

var dbBackupCmd = &cobra.Command{
	Use:   "backup [file]",
	Short: "Create pg_dump backup",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDBBackup,
}

// ── restore ─────────────────────────────────────────────────────────

var dbRestoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Restore from backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runDBRestore,
}

// ── shell ───────────────────────────────────────────────────────────

var dbShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open psql interactive shell",
	RunE:  runDBShell,
}

// ── drop ────────────────────────────────────────────────────────────

var dbDropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop the project database (DESTRUCTIVE)",
	RunE:  runDBDrop,
}

// ── reset ───────────────────────────────────────────────────────────

var dbResetForce bool

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Drop and recreate database (DESTRUCTIVE)",
	RunE:  runDBReset,
}

// ── backup list ─────────────────────────────────────────────────────

var dbBackupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups with size and date",
	RunE:  runDBBackupList,
}

// ── hasura ──────────────────────────────────────────────────────────

var dbHasuraCmd = &cobra.Command{
	Use:   "hasura",
	Short: "Hasura metadata operations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var dbHasuraConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Open Hasura Console",
	RunE:  runDBHasuraConsole,
}

var dbHasuraMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "Manage Hasura metadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var dbHasuraMetadataApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Hasura metadata",
	RunE:  runDBHasuraMetadataApply,
}

var dbHasuraMetadataExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export Hasura metadata",
	RunE:  runDBHasuraMetadataExport,
}

var dbHasuraMetadataReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload metadata cache",
	RunE:  runDBHasuraMetadataReload,
}

// ── init ────────────────────────────────────────────────────────────

func init() {
	// --force / --yes flags on reset (--force kept for backward compat)
	dbResetCmd.Flags().BoolVarP(&dbResetForce, "force", "f", false, "Skip confirmation prompt (for CI/automation)")
	dbResetCmd.Flags().Bool("yes", false, "Skip confirmation prompt (for CI/automation)")

	// --yes flag on drop and restore
	dbDropCmd.Flags().Bool("yes", false, "Skip confirmation prompt (for CI/automation)")
	dbRestoreCmd.Flags().Bool("overwrite", false, "Allow overwriting existing data")
	dbRestoreCmd.Flags().Bool("yes", false, "Skip confirmation prompt in production (for planned maintenance)")

	// --format flag on backup list
	dbBackupListCmd.Flags().String("format", "", "Output format: table (default) or json")

	// --plugin flag on migrate and its subcommands
	dbMigrateCmd.PersistentFlags().String("plugin", "", "Migrate specific plugin schema")

	// Wire migrate subcommands
	dbMigrateCmd.AddCommand(dbMigrateUpCmd)
	dbMigrateCmd.AddCommand(dbMigrateDownCmd)
	dbMigrateCmd.AddCommand(dbMigrateStatusCmd)
	dbMigrateCmd.AddCommand(dbMigrateCreateCmd)

	// Wire hasura metadata subcommands
	dbHasuraMetadataCmd.AddCommand(dbHasuraMetadataApplyCmd)
	dbHasuraMetadataCmd.AddCommand(dbHasuraMetadataExportCmd)
	dbHasuraMetadataCmd.AddCommand(dbHasuraMetadataReloadCmd)

	// Wire hasura subcommands
	dbHasuraCmd.AddCommand(dbHasuraConsoleCmd)
	dbHasuraCmd.AddCommand(dbHasuraMetadataCmd)

	// Wire backup subcommands
	dbBackupCmd.AddCommand(dbBackupListCmd)

	// Wire top-level db subcommands
	dbCmd.AddCommand(dbMigrateCmd)
	dbCmd.AddCommand(dbSeedCmd)
	dbCmd.AddCommand(dbBackupCmd)
	dbCmd.AddCommand(dbRestoreCmd)
	dbCmd.AddCommand(dbShellCmd)
	dbCmd.AddCommand(dbDropCmd)
	dbCmd.AddCommand(dbResetCmd)
	dbCmd.AddCommand(dbHasuraCmd)

	RootCmd.AddCommand(dbCmd)
}

// ── helpers ─────────────────────────────────────────────────────────

// loadProjectConfig loads the nSelf configuration from the current working directory.
func loadProjectConfig() (*config.Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

// ── run functions ───────────────────────────────────────────────────

func runDBMigrateUp(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	plugin, _ := cmd.Flags().GetString("plugin")
	if err := database.MigrateUp(cmd.Context(), cfg, plugin); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	fmt.Println("Migrations applied successfully.")
	return nil
}

func runDBMigrateDown(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	if err := database.MigrateDown(cmd.Context(), cfg); err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}
	fmt.Println("Last migration reverted.")
	return nil
}

func runDBMigrateStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	statuses, err := database.MigrateStatus(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("migrate status: %w", err)
	}
	if len(statuses) == 0 {
		fmt.Println("No migrations found.")
		return nil
	}
	fmt.Printf("%-40s %-10s %s\n", "MIGRATION", "STATUS", "APPLIED AT")
	for _, s := range statuses {
		status := "pending"
		appliedAt := ""
		if s.Applied {
			status = "applied"
			if !s.Timestamp.IsZero() {
				appliedAt = s.Timestamp.Format(time.RFC3339)
			}
		}
		fmt.Printf("%-40s %-10s %s\n", s.Name, status, appliedAt)
	}
	return nil
}

func runDBMigrateCreate(_ *cobra.Command, args []string) error {
	name := args[0]
	if name == "" {
		return fmt.Errorf("migration name is required")
	}
	// Allow only lowercase alphanumeric characters, underscores, and hyphens.
	// This prevents path traversal and other filesystem attacks.
	if !migrationNameAllowed.MatchString(name) {
		return fmt.Errorf("migration name %q contains invalid characters: only lowercase letters, digits, underscores, and hyphens are allowed", name)
	}

	ts := time.Now().Format("20060102150405")

	dir := "migrations"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create migrations directory: %w", err)
	}

	upFile := filepath.Join(dir, fmt.Sprintf("%s_%s.sql", ts, name))
	downFile := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", ts, name))

	if err := os.WriteFile(upFile, []byte("-- migrate up\n"), 0o644); err != nil {
		return fmt.Errorf("create up migration: %w", err)
	}
	if err := os.WriteFile(downFile, []byte("-- migrate down\n"), 0o644); err != nil {
		return fmt.Errorf("create down migration: %w", err)
	}

	fmt.Printf("Created migration files:\n  %s\n  %s\n", upFile, downFile)
	return nil
}

func runDBSeed(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	var file string
	if len(args) > 0 {
		file = args[0]
	}
	if err := database.Seed(cmd.Context(), cfg, file); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	fmt.Println("Seed data applied.")
	return nil
}

func runDBBackup(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	var outputPath string
	if len(args) > 0 {
		outputPath = args[0]
	}
	if err := database.Backup(cmd.Context(), cfg, outputPath); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	fmt.Println("Backup created successfully.")
	return nil
}

func runDBRestore(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	yesFlag, _ := cmd.Flags().GetBool("yes")
	if overwrite && cfg.IsProduction() && !yesFlag {
		if err := requireProductionConfirmation(cfg.ProjectName); err != nil {
			return err
		}
	}
	if err := database.Restore(cmd.Context(), cfg, args[0]); err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	fmt.Println("Database restored successfully.")
	return nil
}

func runDBShell(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	psqlCmd := []string{"psql", "-U", user, "-d", db}
	opts := docker.ExecOptions{
		Interactive: true,
		TTY:         true,
	}

	return docker.Exec(cmd.Context(), container, psqlCmd, opts)
}

func runDBReset(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	yesFlag, _ := cmd.Flags().GetBool("yes")
	skipConfirm := dbResetForce || yesFlag

	// Production environments require typing the project name to confirm.
	if cfg.IsProduction() && !skipConfirm {
		if err := requireProductionConfirmation(cfg.ProjectName); err != nil {
			return err
		}
	} else if !skipConfirm {
		// Non-production: simple yes/no confirmation.
		fmt.Fprintf(os.Stderr, "WARNING: This will drop and recreate database %q. Type \"yes\" to confirm: ", db)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading confirmation: %w", err)
		}
		if strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Fprintln(os.Stderr, "aborted")
			return fmt.Errorf("reset aborted by user")
		}
	}

	// Terminate existing connections and drop the database, then recreate.
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	terminateSQL := fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
		strings.ReplaceAll(db, "'", "''"),
	)
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", db)
	createSQL := fmt.Sprintf("CREATE DATABASE \"%s\" OWNER \"%s\"", db, user)

	for _, sql := range []string{terminateSQL, dropSQL, createSQL} {
		args := []string{
			"exec", container,
			"psql", "-U", user, "-d", "postgres", "-c", sql,
		}
		c := exec.CommandContext(cmd.Context(), "docker", args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("reset failed on %q: %w", sql, err)
		}
	}

	// Re-initialize schemas and extensions.
	if err := database.InitializeDatabase(cmd.Context(), cfg); err != nil {
		return fmt.Errorf("reinitialize after reset: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Database reset complete.")
	return nil
}

func runDBHasuraConsole(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	port := cfg.Hasura.Port
	if port == 0 {
		port = 8080
	}
	url := fmt.Sprintf("http://localhost:%d/console", port)
	fmt.Printf("Opening Hasura Console: %s\n", url)

	// Attempt to open the browser. Best-effort on macOS/Linux.
	_ = exec.Command("open", url).Start()

	return nil
}

func runDBHasuraMetadataApply(cmd *cobra.Command, _ []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	if err := database.HasuraApplyMetadata(cmd.Context(), cfg, dir); err != nil {
		return fmt.Errorf("hasura metadata apply: %w", err)
	}
	fmt.Println("Hasura metadata applied.")
	return nil
}

func runDBHasuraMetadataExport(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	data, err := database.HasuraExportMetadata(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("hasura metadata export: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runDBHasuraMetadataReload(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	if err := database.HasuraReloadMetadata(cmd.Context(), cfg); err != nil {
		return fmt.Errorf("hasura metadata reload: %w", err)
	}
	fmt.Println("Hasura metadata reloaded.")
	return nil
}

// requireProductionConfirmation prints a production warning and requires the
// user to type the project name exactly to proceed. Returns an error if the
// confirmation fails or the input cannot be read.
func requireProductionConfirmation(projectName string) error {
	fmt.Printf("WARNING: PRODUCTION: This will DESTROY the database %s.\n", projectName)
	fmt.Print("   Type the project name to confirm: ")
	var confirm string
	if _, err := fmt.Scanln(&confirm); err != nil {
		return fmt.Errorf("reading confirmation: %w", err)
	}
	if strings.TrimSpace(confirm) != projectName {
		return fmt.Errorf("confirmation failed: got %q, expected %q", confirm, projectName)
	}
	return nil
}

func runDBDrop(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	yesFlag, _ := cmd.Flags().GetBool("yes")

	if cfg.IsProduction() && !yesFlag {
		if err := requireProductionConfirmation(cfg.ProjectName); err != nil {
			return err
		}
	} else if !yesFlag {
		fmt.Fprintf(os.Stderr, "WARNING: This will drop database %q. Type \"yes\" to confirm: ", db)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if scanErr := scanner.Err(); scanErr != nil {
			return fmt.Errorf("reading confirmation: %w", scanErr)
		}
		if strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Fprintln(os.Stderr, "aborted")
			return fmt.Errorf("drop aborted by user")
		}
	}

	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	terminateSQL := fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()",
		strings.ReplaceAll(db, "'", "''"),
	)
	dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", db)

	for _, sql := range []string{terminateSQL, dropSQL} {
		args := []string{
			"exec", container,
			"psql", "-U", user, "-d", "postgres", "-c", sql,
		}
		c := exec.CommandContext(cmd.Context(), "docker", args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("drop failed on %q: %w", sql, err)
		}
	}

	fmt.Fprintln(os.Stderr, "Database dropped.")
	return nil
}

// backupEntry holds parsed metadata for a single backup file.
type backupEntry struct {
	ID   string    `json:"id"`
	Date time.Time `json:"date"`
	Size int64     `json:"size"`
	Type string    `json:"type"`
}

func runDBBackupList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	backupDir := cfg.Backup.Dir
	if backupDir == "" {
		backupDir = "backups"
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No backups found.")
			return nil
		}
		return fmt.Errorf("reading backup directory %s: %w", backupDir, err)
	}

	var backups []backupEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Accept .dump and .sql files only.
		if !strings.HasSuffix(name, ".dump") && !strings.HasSuffix(name, ".sql") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Derive backup type from filename: files containing "scheduled" or
		// created outside the manual CLI path are labelled "scheduled";
		// everything else is "manual".
		backupType := "manual"
		if strings.Contains(strings.ToLower(name), "scheduled") ||
			strings.Contains(strings.ToLower(name), "auto") {
			backupType = "scheduled"
		}

		// Use the file modification time as the backup date.
		id := strings.TrimSuffix(strings.TrimSuffix(name, ".dump"), ".sql")
		backups = append(backups, backupEntry{
			ID:   id,
			Date: info.ModTime(),
			Size: info.Size(),
			Type: backupType,
		})
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(backups)
	}

	// Table output.
	fmt.Printf("%-30s  %-21s  %-8s  %s\n", "ID", "DATE", "SIZE", "TYPE")
	for _, b := range backups {
		sizeStr := formatBackupSize(b.Size)
		fmt.Printf("%-30s  %-21s  %-8s  %s\n",
			b.ID,
			b.Date.Format("2006-01-02 15:04:05"),
			sizeStr,
			b.Type,
		)
	}
	return nil
}

// formatBackupSize returns a human-readable size string (KB, MB, GB).
func formatBackupSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%.0fGB", float64(bytes)/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.0fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.0fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
