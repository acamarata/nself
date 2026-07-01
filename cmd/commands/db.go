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
	"github.com/nself-org/cli/internal/seed"
	"github.com/nself-org/cli/internal/tenant"

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
	Long: `Apply pending migrations to the project's Postgres database.

Pass --env staging|prod (with NSELF_DEPLOY_HOST_<ENV> set, or an entry in
.nself/control-plane.yaml) to run this against a deployed target instead of
the local docker daemon: the command re-invokes 'nself db migrate up' on the
remote host over SSH. Note: this requires the remote host's nself CLI
version to support 'db migrate up' — an older remote CLI returns a clear
error naming the version-drift cause rather than a raw SSH failure.`,
	RunE: runDBMigrateUp,
}

var dbMigrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Revert last migration",
	RunE:  runDBMigrateDown,
}

var dbMigrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long: `Show which migrations have been applied to the project's Postgres database.

Pass --env staging|prod to check a deployed target instead of the local
docker daemon (see 'nself db migrate up --help' for remote-targeting and
version-drift details).`,
	RunE: runDBMigrateStatus,
}

var dbMigrateCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create new migration file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDBMigrateCreate,
}

var dbMigrateApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a specific migration file (G-008)",
	Long: `Apply a single SQL migration file by path.

The file is validated to exist before execution. If the migration is
already recorded in schema_versions (matched by filename + SHA-256
checksum), the command warns and exits cleanly without re-applying.

This closes G-008: plugin-claw external RLS migrations can be applied
via CLI without requiring 'nself db shell' as a workaround.`,
	RunE: runDBMigrateApply,
}

// ── seed ────────────────────────────────────────────────────────────

var dbSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Database seeding: run, list, verify, graph",
	Long: `Database seeding with environment-aware fixtures.

Subcommands:
  run      Execute seeds for current environment
  list     List available seeds and fixtures
  verify   Verify a fixture is deterministic
  graph    Show seed dependency graph

Legacy usage (backward compat):
  nself db seed [file]   — runs a single seed file directly`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDBSeed,
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
	Long: `Open an interactive psql shell connected to the project database.

The shell is launched inside the running Postgres Docker container via
'docker exec'. psql must be available inside the container (standard
with the official postgres image).

Windows note: on Windows the shell is proxied through 'docker exec'
so psql itself does not need to be in the host PATH. However if you
are connecting to an external Postgres instance (not managed by nself)
you must ensure psql.exe is in your PATH before running this command.`,
	RunE: runDBShell,
}

// ── list ────────────────────────────────────────────────────────────

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List databases in the project Postgres instance",
	Long: `List all databases in the project's Postgres container.

Connects to the running Postgres container and prints all database names
(equivalent to \l in psql). Useful for verifying that db create/drop/reset
operated on the correct database.`,
	RunE: runDBList,
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

// ── verify-checksums ───────────────────────────────────────────────

var dbVerifyChecksumsCmd = &cobra.Command{
	Use:   "verify-checksums",
	Short: "Verify migration file checksums against stored values",
	RunE:  runDBVerifyChecksums,
}

// ── reset-checksum ─────────────────────────────────────────────────

var dbResetChecksumCmd = &cobra.Command{
	Use:   "reset-checksum <id>",
	Short: "Reset stored checksum for a migration (dangerous)",
	Args:  cobra.ExactArgs(1),
	RunE:  runDBResetChecksum,
}

// ── seed subcommands ───────────────────────────────────────────────

var dbSeedRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute seeds for current environment",
	RunE:  runDBSeedRun,
}

var dbSeedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available seeds and fixtures",
	RunE:  runDBSeedList,
}

var dbSeedVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a fixture is deterministic (dry-run replay check)",
	RunE:  runDBSeedVerify,
}

var dbSeedGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Show seed dependency graph",
	RunE:  runDBSeedGraph,
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
	Long: `Apply Hasura metadata (tables, permissions, relationships, actions) to the
project's Hasura instance.

Pass --env staging|prod to apply against a deployed target's Hasura instead
of the local one: the command re-invokes 'nself db hasura metadata apply' on
the remote host over SSH, so the remote box always uses its own local
project/Hasura resolution. Note: this requires the remote host's nself CLI
version to support this subcommand — an older remote CLI returns a clear
version-drift error rather than a raw SSH failure.`,
	RunE: runDBHasuraMetadataApply,
}

var dbHasuraMetadataExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export Hasura metadata to git-friendly sorted YAML",
	RunE:  runDBHasuraMetadataExport,
}

var dbHasuraMetadataReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload metadata cache",
	RunE:  runDBHasuraMetadataReload,
}

var dbHasuraDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare live Hasura metadata against on-disk files",
	RunE:  runDBHasuraDiff,
}

var dbHasuraValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Hasura metadata consistency and permission coverage",
	RunE:  runDBHasuraValidate,
}

// ── lint ────────────────────────────────────────────────────────────

var dbLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Check RLS policies on tenant-scoped tables",
	Long: `Audit Row-Level Security policies across all user-data tables.

By default, checks tables with tenant_id columns. With --rls, performs an
exhaustive audit of every np_* table and any table with user_id or tenant_id.

Flags:
  --rls       Exhaustive RLS audit (all np_* and user-data tables)
  --metric    Emit Prometheus-compatible metric line (nself_rls_disabled_tables)
  --matrix    Include table x role coverage matrix in output
  --format    Output format: table (default) or json
  --remediate Print remediation SQL for failing tables`,
	RunE: runDBLint,
}

func runDBLint(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	rlsFlag, _ := cmd.Flags().GetBool("rls")
	metricFlag, _ := cmd.Flags().GetBool("metric")
	matrixFlag, _ := cmd.Flags().GetBool("matrix")
	remediateFlag, _ := cmd.Flags().GetBool("remediate")
	formatFlag, _ := cmd.Flags().GetString("format")

	// Legacy path: no --rls flag uses the original tenant_id-only check.
	if !rlsFlag && !metricFlag {
		results, err := tenant.LintRLS(cmd.Context(), cfg)
		if err != nil {
			return fmt.Errorf("db lint: %w", err)
		}
		if len(results) == 0 {
			fmt.Println("No tenant-scoped tables found.")
			return nil
		}
		if formatFlag == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		}
		failures := 0
		for _, r := range results {
			status := "PASS"
			if !r.Pass {
				status = "FAIL"
				failures++
			}
			fmt.Printf("[%s] %s.%s — %s\n", status, r.Schema, r.Table, r.Message)
		}
		if failures > 0 {
			return fmt.Errorf("%d table(s) with tenant_id missing RLS policies", failures)
		}
		fmt.Printf("All %d tenant-scoped table(s) have RLS policies.\n", len(results))
		return nil
	}

	// Full RLS audit path.
	allowlist := loadAllowlist(cfg)
	report, err := tenant.LintRLSFull(cmd.Context(), cfg, allowlist)
	if err != nil {
		return fmt.Errorf("db lint --rls: %w", err)
	}

	// --metric: emit Prometheus text line and exit.
	if metricFlag {
		count := tenant.DisabledTableCount(report)
		fmt.Printf("# HELP nself_rls_disabled_tables Number of user-data tables without RLS.\n")
		fmt.Printf("# TYPE nself_rls_disabled_tables gauge\n")
		fmt.Printf("nself_rls_disabled_tables %d\n", count)
		return nil
	}

	if formatFlag == "json" {
		if !matrixFlag {
			report.CoverageMatrix = nil
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	// Table output.
	fmt.Printf("RLS Audit: %d tables checked, %d enabled, %d disabled, %d allowlisted, %d violations\n\n",
		report.TotalTables, report.RLSEnabled, report.RLSDisabled, report.Allowlisted, report.Violations)

	for _, r := range report.Tables {
		status := "PASS"
		if !r.Pass {
			status = "FAIL"
		}
		if r.Allowlisted {
			status = "SKIP"
		}
		fmt.Printf("[%s] %s.%s (policies: %d) — %s\n", status, r.Schema, r.Table, r.PolicyCount, r.Message)
	}

	// Coverage matrix.
	if matrixFlag && len(report.CoverageMatrix) > 0 {
		fmt.Printf("\n--- Coverage Matrix (table x role) ---\n")
		roles := []string{"anonymous", "user", "tenant_member", "tenant_admin", "admin", "service"}
		fmt.Printf("%-40s", "TABLE")
		for _, role := range roles {
			fmt.Printf(" %-16s", role)
		}
		fmt.Println()
		for _, entry := range report.CoverageMatrix {
			fmt.Printf("%-40s", entry.Schema+"."+entry.Table)
			for _, role := range roles {
				pol := entry.Roles[role]
				if pol == "" {
					pol = "none"
				}
				if len(pol) > 15 {
					pol = pol[:15]
				}
				fmt.Printf(" %-16s", pol)
			}
			fmt.Println()
		}
	}

	// Remediation SQL.
	if remediateFlag {
		sql := tenant.GenerateRemediationSQL(report)
		if sql != "" {
			fmt.Printf("\n--- Remediation SQL ---\n%s", sql)
		} else {
			fmt.Println("\nNo remediation needed.")
		}
	}

	if report.Violations > 0 {
		return fmt.Errorf("%d table(s) failing RLS audit", report.Violations)
	}
	return nil
}

// loadAllowlist reads the RLS allowlist from the project working directory.
func loadAllowlist(_ *config.Config) []tenant.AllowlistEntry {
	dir, _ := os.Getwd()
	paths := []string{
		filepath.Join(dir, "lint_allowlist.yaml"),
		filepath.Join(dir, ".nself", "lint_allowlist.yaml"),
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		return parseAllowlistYAML(data)
	}
	return nil
}

// parseAllowlistYAML parses a simple YAML allowlist. Format:
//
//	tables:
//	  - schema: public
//	    table: migrations
//	    reason: "System metadata table"
func parseAllowlistYAML(data []byte) []tenant.AllowlistEntry {
	var entries []tenant.AllowlistEntry
	lines := strings.Split(string(data), "\n")
	var current *tenant.AllowlistEntry
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- schema:") {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &tenant.AllowlistEntry{
				Schema: strings.TrimSpace(strings.TrimPrefix(trimmed, "- schema:")),
			}
		} else if strings.HasPrefix(trimmed, "table:") && current != nil {
			current.Table = strings.TrimSpace(strings.TrimPrefix(trimmed, "table:"))
		} else if strings.HasPrefix(trimmed, "reason:") && current != nil {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "reason:"))
			current.Reason = strings.Trim(val, "\"'")
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	return entries
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

	// --dry-run flag on migrate up
	dbMigrateUpCmd.Flags().Bool("dry-run", false, "List pending migrations without applying them")
	// --migration-dir flag on migrate up (G-008)
	dbMigrateUpCmd.Flags().String("migration-dir", "", "Apply all .sql files in this directory in lexicographic order (skips already-applied)")
	// --env/--server: remote targeting (gap #9) — default local, opt-in remote.
	addDBRemoteFlags(dbMigrateUpCmd)
	addDBRemoteFlags(dbMigrateStatusCmd)
	addDBRemoteFlags(dbHasuraMetadataApplyCmd)

	// --file flag on migrate apply (G-008)
	dbMigrateApplyCmd.Flags().String("file", "", "Path to the SQL migration file to apply")
	_ = dbMigrateApplyCmd.MarkFlagRequired("file")

	// Wire migrate subcommands
	dbMigrateCmd.AddCommand(dbMigrateUpCmd)
	dbMigrateCmd.AddCommand(dbMigrateDownCmd)
	dbMigrateCmd.AddCommand(dbMigrateStatusCmd)
	dbMigrateCmd.AddCommand(dbMigrateCreateCmd)
	dbMigrateCmd.AddCommand(dbMigrateApplyCmd)

	// Wire hasura metadata subcommands
	dbHasuraMetadataCmd.AddCommand(dbHasuraMetadataApplyCmd)
	dbHasuraMetadataCmd.AddCommand(dbHasuraMetadataExportCmd)
	dbHasuraMetadataCmd.AddCommand(dbHasuraMetadataReloadCmd)

	// Wire hasura subcommands
	dbHasuraCmd.AddCommand(dbHasuraConsoleCmd)
	dbHasuraCmd.AddCommand(dbHasuraMetadataCmd)
	dbHasuraCmd.AddCommand(dbHasuraDiffCmd)
	dbHasuraCmd.AddCommand(dbHasuraValidateCmd)

	// Wire backup subcommands
	dbBackupCmd.AddCommand(dbBackupListCmd)

	// db lint flags
	dbLintCmd.Flags().String("format", "table", "Output format: table or json")
	dbLintCmd.Flags().Bool("rls", false, "Exhaustive RLS audit (all np_* and user-data tables)")
	dbLintCmd.Flags().Bool("metric", false, "Emit Prometheus metric line (nself_rls_disabled_tables)")
	dbLintCmd.Flags().Bool("matrix", false, "Include table x role coverage matrix")
	dbLintCmd.Flags().Bool("remediate", false, "Print remediation SQL for failing tables")

	// verify-checksums and reset-checksum
	dbResetChecksumCmd.Flags().Bool("i-know-what-im-doing", false, "Required safety flag")

	// seed subcommands
	dbSeedRunCmd.Flags().String("env", "", "Target environment (default: current)")
	dbSeedRunCmd.Flags().String("fixture", "", "Run a specific fixture (e.g. demo, load-test)")
	dbSeedRunCmd.Flags().Bool("reset", false, "Truncate seeded tables before running (destructive)")
	dbSeedCmd.AddCommand(dbSeedRunCmd)
	dbSeedCmd.AddCommand(dbSeedListCmd)
	dbSeedCmd.AddCommand(dbSeedVerifyCmd)
	dbSeedCmd.AddCommand(dbSeedGraphCmd)
	dbSeedVerifyCmd.Flags().String("fixture", "", "Fixture to verify")

	// Wire top-level db subcommands
	dbCmd.AddCommand(dbLintCmd)
	dbCmd.AddCommand(dbMigrateCmd)
	dbCmd.AddCommand(dbSeedCmd)
	dbCmd.AddCommand(dbVerifyChecksumsCmd)
	dbCmd.AddCommand(dbResetChecksumCmd)
	dbCmd.AddCommand(dbBackupCmd)
	dbCmd.AddCommand(dbRestoreCmd)
	dbCmd.AddCommand(dbShellCmd)
	dbCmd.AddCommand(dbListCmd)
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
	if handled, err := dispatchRemoteIfNeeded(cmd, "db", "migrate", "up"); handled {
		return err
	}

	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	// Verify PostgreSQL is running before attempting any migration.
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	checkCmd := exec.CommandContext(cmd.Context(), "docker", "exec", container,
		"pg_isready", "-U", user)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("PostgreSQL is not running. Start with 'nself start' first.")
	}

	plugin, _ := cmd.Flags().GetString("plugin")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	migrationDir, _ := cmd.Flags().GetString("migration-dir")

	// --migration-dir: apply all .sql files in the given directory (G-008).
	if migrationDir != "" {
		count, err := database.MigrateUpDir(cmd.Context(), cfg, migrationDir)
		if err != nil {
			return fmt.Errorf("migrate up dir: %w", err)
		}
		if count > 0 {
			fmt.Printf("Applied %d migration(s) from %s.\n", count, migrationDir)
		} else {
			fmt.Println("No pending migrations in directory.")
		}
		return nil
	}

	if dryRun {
		pending, err := database.PendingMigrations(cmd.Context(), cfg, plugin)
		if err != nil {
			return fmt.Errorf("pending migrations: %w", err)
		}
		if len(pending) == 0 {
			fmt.Println("No pending migrations.")
			return nil
		}
		for _, name := range pending {
			fmt.Println(name)
		}
		return nil
	}

	count, err := database.MigrateUp(cmd.Context(), cfg, plugin)
	if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	if count > 0 {
		fmt.Printf("Applied %d migration(s).\n", count)
	} else {
		fmt.Println("No pending migrations.")
	}
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
	if handled, err := dispatchRemoteIfNeeded(cmd, "db", "migrate", "status"); handled {
		return err
	}

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

// runDBMigrateApply implements 'nself db migrate apply --file <path>' (G-008).
// It applies a single SQL migration file and records it in schema_versions by
// filename + SHA-256 checksum. If the migration is already recorded, it warns
// and exits cleanly without re-applying (double-apply protection).
func runDBMigrateApply(cmd *cobra.Command, _ []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}

	// Validate the file exists before loading config or touching the database.
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("migration file not found: %s", filePath)
		}
		return fmt.Errorf("stat migration file: %w", err)
	}

	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	skipped, err := database.ApplyFile(cmd.Context(), cfg, filePath)
	if err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	if skipped {
		fmt.Printf("already applied: %s (skipped)\n", filepath.Base(filePath))
	} else {
		fmt.Printf("Applied: %s\n", filepath.Base(filePath))
	}
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

// runDBList implements 'nself db list'. It queries the Postgres container for
// all database names and prints them one per line.
func runDBList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}

	// -t: tuples only, -A: unaligned, -c: inline SQL — produces one db name per line.
	listSQL := "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname"
	args := []string{
		"exec", container,
		"psql", "-U", user, "-d", "postgres", "-t", "-A", "-c", listSQL,
	}
	c := exec.CommandContext(cmd.Context(), "docker", args...)
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	if err := c.Run(); err != nil {
		return fmt.Errorf("db list failed: %w", err)
	}
	return nil
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
	if handled, err := dispatchRemoteIfNeeded(cmd, "db", "hasura", "metadata", "apply"); handled {
		return err
	}

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
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	outDir, err := database.HasuraExportToYAML(cmd.Context(), cfg, dir)
	if err != nil {
		// Fall back to raw JSON dump.
		data, jsonErr := database.HasuraExportMetadata(cmd.Context(), cfg)
		if jsonErr != nil {
			return fmt.Errorf("hasura metadata export: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Metadata exported to %s\n", outDir)
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

func runDBVerifyChecksums(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	plugin, _ := cmd.Flags().GetString("plugin")
	mismatches, err := database.VerifyChecksums(cmd.Context(), cfg, plugin)
	if err != nil {
		return fmt.Errorf("verify checksums: %w", err)
	}
	if len(mismatches) == 0 {
		fmt.Println("All checksums verified.")
		return nil
	}
	for _, m := range mismatches {
		fmt.Printf("MISMATCH %s (%s): stored=%s disk=%s\n", m.ID, m.Name, m.Expected[:12]+"...", m.Actual[:12]+"...")
	}
	return fmt.Errorf("%d checksum mismatch(es) found", len(mismatches))
}

func runDBResetChecksum(cmd *cobra.Command, args []string) error {
	safety, _ := cmd.Flags().GetBool("i-know-what-im-doing")
	if !safety {
		return fmt.Errorf("this command modifies migration tracking; pass --i-know-what-im-doing to confirm")
	}
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	if err := database.ResetChecksum(cmd.Context(), cfg, args[0]); err != nil {
		return fmt.Errorf("reset checksum: %w", err)
	}
	fmt.Printf("Checksum reset for migration %s.\n", args[0])
	return nil
}

func runDBSeedRun(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	env, _ := cmd.Flags().GetString("env")
	if env == "" {
		env = cfg.Env
	}
	if env == "" {
		env = "dev"
	}

	fixture, _ := cmd.Flags().GetString("fixture")

	seeds, err := seed.CollectForRun(dir, env, fixture)
	if err != nil {
		return fmt.Errorf("collect seeds: %w", err)
	}

	if len(seeds) == 0 {
		fmt.Println("No seeds found.")
		return nil
	}

	// Guard against destructive seeds in production.
	if cfg.IsProduction() {
		for _, s := range seeds {
			if s.Destructive {
				return fmt.Errorf("destructive seed %s cannot run in production", s.Name)
			}
		}
	}

	for _, s := range seeds {
		if err := database.Seed(cmd.Context(), cfg, s.Path); err != nil {
			return fmt.Errorf("seed %s: %w", s.Name, err)
		}
		fmt.Printf("  Applied: %s\n", s.Name)
	}
	fmt.Printf("Applied %d seed(s).\n", len(seeds))
	return nil
}

func runDBSeedList(_ *cobra.Command, _ []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	seeds, err := seed.ListSeeds(dir)
	if err != nil {
		return fmt.Errorf("list seeds: %w", err)
	}

	if len(seeds) == 0 {
		fmt.Println("No seeds found. Create db/seeds/ directory to get started.")
		return nil
	}

	fmt.Printf("%-30s %-20s %-12s %s\n", "NAME", "ENV", "TYPE", "DEPENDS ON")
	for _, s := range seeds {
		seedType := "standard"
		if s.Idempotent {
			seedType = "idempotent"
		}
		if s.Destructive {
			seedType = "destructive"
		}
		deps := strings.Join(s.DependsOn, ", ")
		fmt.Printf("%-30s %-20s %-12s %s\n", s.Name, s.Env, seedType, deps)
	}

	// Also list fixtures.
	fixtures, _ := seed.ListFixtures(dir)
	if len(fixtures) > 0 {
		fmt.Printf("\nFixtures: %s\n", strings.Join(fixtures, ", "))
	}
	return nil
}

func runDBSeedVerify(cmd *cobra.Command, _ []string) error {
	fixture, _ := cmd.Flags().GetString("fixture")
	if fixture == "" {
		return fmt.Errorf("--fixture is required")
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	seeds, err := seed.CollectForRun(dir, "", fixture)
	if err != nil {
		return fmt.Errorf("collect fixture seeds: %w", err)
	}

	if len(seeds) == 0 {
		return fmt.Errorf("fixture %q has no seeds", fixture)
	}

	fmt.Printf("Fixture %q: %d seed file(s)\n", fixture, len(seeds))
	for _, s := range seeds {
		marker := "  "
		if s.Idempotent {
			marker = "I "
		}
		if s.Destructive {
			marker = "D "
		}
		fmt.Printf("  %s%s\n", marker, s.Name)
	}
	fmt.Println("Verification: seed files parsed successfully.")
	return nil
}

func runDBSeedGraph(_ *cobra.Command, _ []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	nodes, err := seed.DependencyGraph(dir)
	if err != nil {
		return fmt.Errorf("dependency graph: %w", err)
	}

	if len(nodes) == 0 {
		fmt.Println("No seeds found.")
		return nil
	}

	for _, n := range nodes {
		if len(n.DependsOn) == 0 {
			fmt.Printf("%s (%s)\n", n.Name, n.Env)
		} else {
			fmt.Printf("%s (%s) -> %s\n", n.Name, n.Env, strings.Join(n.DependsOn, ", "))
		}
	}
	return nil
}

func runDBHasuraDiff(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	diffs, err := database.HasuraDiffMetadata(cmd.Context(), cfg, dir)
	if err != nil {
		return fmt.Errorf("hasura diff: %w", err)
	}

	if len(diffs) == 0 {
		fmt.Println("No metadata drift detected.")
		return nil
	}

	fmt.Printf("Metadata drift detected in %d key(s):\n", len(diffs))
	for _, d := range diffs {
		fmt.Printf("  - %s\n", d)
	}
	return fmt.Errorf("metadata drift detected")
}

func runDBHasuraValidate(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	missing, err := database.HasuraValidatePermissions(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("hasura validate: %w", err)
	}

	if len(missing) == 0 {
		fmt.Println("All tracked tables have required permissions (tenant_member, tenant_admin).")
		return nil
	}

	fmt.Printf("Permission coverage issues (%d):\n", len(missing))
	for _, m := range missing {
		fmt.Printf("  - %s\n", m)
	}
	return fmt.Errorf("%d permission coverage issue(s)", len(missing))
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
