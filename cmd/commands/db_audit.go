package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/database"
	"github.com/spf13/cobra"
)

// ── migrate audit ────────────────────────────────────────────────────

var dbMigrateAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit migrations for idempotency, rollback files, and checksum drift",
	Long: `Audit all migration files for:
  - Idempotency: safe-to-rerun SQL patterns (IF NOT EXISTS, CREATE OR REPLACE)
  - Rollback coverage: presence of a corresponding down.sql
  - Checksum drift: on-disk file matches the checksum recorded on apply`,
	RunE: runDBMigrateAudit,
}

// ── migrate idempotent ───────────────────────────────────────────────

var dbMigrateIdempotentCmd = &cobra.Command{
	Use:   "idempotent <file>",
	Short: "Check and optionally rewrite a migration file to be idempotent",
	Long: `Analyzes a migration SQL file for non-idempotent patterns and suggests
(or generates) an idempotent version using IF NOT EXISTS / IF EXISTS clauses.`,
	Args: cobra.ExactArgs(1),
	RunE: runDBMigrateIdempotent,
}

// ── db drift ─────────────────────────────────────────────────────────

var dbDriftCmd = &cobra.Command{
	Use:     "drift",
	Aliases: []string{"schema-drift"},
	Short:   "Schema drift detection: scan np_* tables for Theme 25 column compliance",
	Long: `Detect schema drift across all np_* tables.

Theme 25 mandates 5 columns on every np_* table:
  id         UUID primary key
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
  user_id    UUID (nullable, owner reference)
  deleted_at TIMESTAMPTZ (soft-delete)

Subcommands:
  scan   Report missing columns per table
  fix    Generate migration SQL to add missing columns`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var dbDriftScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan np_* tables for missing Theme 25 columns",
	RunE:  runDBDriftScan,
}

var dbDriftFixCmd = &cobra.Command{
	Use:   "fix [schema table]",
	Short: "Generate migration SQL to fix Theme 25 column drift",
	Long: `Generate up.sql and down.sql migration content for tables with missing columns.

With no arguments, lists drifted tables. Use --all to generate fixes for all drifted tables.
With schema and table arguments, generates fixes for the specified table only.`,
	Args: cobra.MaximumNArgs(2),
	RunE: runDBDriftFix,
}

// ── run functions ────────────────────────────────────────────────────

func runDBMigrateAudit(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	results, err := database.AuditMigrations(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("audit migrations: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No migration files found.")
		return nil
	}

	fmt.Printf("%-45s  %-7s  %-10s  %-8s  %-6s\n",
		"MIGRATION", "APPLIED", "IDEMPOTENT", "ROLLBACK", "DRIFT")
	fmt.Println(strings.Repeat("-", 85))

	hasIssues := false
	for _, r := range results {
		drift := "OK"
		if !r.ChecksumMatch {
			drift = "DRIFT"
		}

		fmt.Printf("%-45s  %-7s  %-10s  %-8s  %-6s\n",
			truncateStr(r.Name, 45),
			auditBoolMark(r.Applied),
			auditBoolMark(r.Idempotent),
			auditBoolMark(r.HasRollback),
			drift,
		)

		for _, issue := range r.Issues {
			fmt.Printf("  - %s\n", issue)
			hasIssues = true
		}
	}

	if hasIssues {
		fmt.Println()
		fmt.Println("Run 'nself db migrate idempotent <file>' to generate idempotent versions.")
	}
	return nil
}

func runDBMigrateIdempotent(_ *cobra.Command, args []string) error {
	filePath := args[0]

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	sqlContent := string(data)

	idempotent, issues := database.CheckMigrationIdempotency(sqlContent)

	if idempotent {
		fmt.Printf("Migration %q is idempotent. No changes needed.\n", filePath)
		return nil
	}

	fmt.Printf("Migration %q has non-idempotent patterns:\n", filePath)
	for _, issue := range issues {
		fmt.Printf("  - %s\n", issue)
	}

	converted, changes := database.GenerateIdempotentVersion(sqlContent)
	if len(changes) == 0 {
		fmt.Println("\nCould not automatically convert all patterns. Manual review required.")
		return nil
	}

	fmt.Printf("\nProposed idempotent version (%d change(s)):\n", len(changes))
	for _, c := range changes {
		fmt.Printf("  + %s\n", c)
	}
	fmt.Println()
	fmt.Println("--- Converted SQL ---")
	fmt.Println(converted)
	return nil
}

func runDBDriftScan(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	schemaFilter, _ := cmd.Flags().GetString("schema")

	results, err := database.ScanSchemaDrift(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("scan schema drift: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No np_* tables found.")
		return nil
	}

	if schemaFilter != "" {
		var filtered []database.SchemaDriftResult
		for _, r := range results {
			if r.TableSchema == schemaFilter {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	summary := database.SummarizeDrift(results)

	// Group by schema for display.
	schemaOrder := []string{}
	schemaMap := make(map[string][]database.SchemaDriftResult)
	for _, r := range results {
		if _, ok := schemaMap[r.TableSchema]; !ok {
			schemaOrder = append(schemaOrder, r.TableSchema)
		}
		schemaMap[r.TableSchema] = append(schemaMap[r.TableSchema], r)
	}

	for _, sname := range schemaOrder {
		tables := schemaMap[sname]
		drifted := 0
		for _, t := range tables {
			if len(t.MissingColumns) > 0 {
				drifted++
			}
		}

		fmt.Printf("\nSchema: %s (%d tables, %d drifted)\n", sname, len(tables), drifted)
		fmt.Printf("  %-30s  %-4s  %-10s  %-10s  %-7s  %-10s  SCORE\n",
			"TABLE", "ID", "CREATED_AT", "UPDATED_AT", "USER_ID", "DELETED_AT")
		fmt.Printf("  %s\n", strings.Repeat("-", 82))

		for _, r := range tables {
			presentSet := make(map[string]bool, len(r.PresentColumns))
			for _, c := range r.PresentColumns {
				presentSet[c] = true
			}

			colMark := func(name string) string {
				if presentSet[name] {
					return "ok"
				}
				return "x"
			}

			fmt.Printf("  %-30s  %-4s  %-10s  %-10s  %-7s  %-10s  %d\n",
				truncateStr(r.TableName, 30),
				colMark("id"),
				colMark("created_at"),
				colMark("updated_at"),
				colMark("user_id"),
				colMark("deleted_at"),
				r.DriftScore,
			)
		}
	}

	fmt.Printf("\nSummary: %d tables total, %d compliant, %d drifted, %d missing required columns, overall score: %d/100\n",
		summary.TotalTables,
		summary.CompliantTables,
		summary.DriftedTables,
		summary.MissingRequired,
		summary.OverallScore,
	)

	if summary.DriftedTables > 0 {
		fmt.Println("\nRun 'nself db drift fix' to generate migration SQL for drifted tables.")
	}
	return nil
}

func runDBDriftFix(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	fixAll, _ := cmd.Flags().GetBool("all")

	results, err := database.ScanSchemaDrift(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("scan schema drift: %w", err)
	}

	var drifted []database.SchemaDriftResult
	for _, r := range results {
		if len(r.MissingColumns) > 0 {
			drifted = append(drifted, r)
		}
	}

	if len(drifted) == 0 {
		fmt.Println("No drift detected. All np_* tables are Theme 25 compliant.")
		return nil
	}

	// Filter by specific schema+table if args provided and not --all.
	if !fixAll && len(args) == 2 {
		schema := args[0]
		table := args[1]
		var single []database.SchemaDriftResult
		for _, r := range drifted {
			if r.TableSchema == schema && r.TableName == table {
				single = append(single, r)
			}
		}
		if len(single) == 0 {
			fmt.Printf("No drift found for %s.%s\n", schema, table)
			return nil
		}
		drifted = single
	} else if !fixAll && len(args) == 0 {
		// Show list without generating SQL.
		fmt.Println("Drifted tables (pass schema and table, or use --all):")
		for _, r := range drifted {
			cols := make([]string, 0, len(r.MissingColumns))
			for _, c := range r.MissingColumns {
				cols = append(cols, c.Name)
			}
			fmt.Printf("  %s.%s  missing: %s\n", r.TableSchema, r.TableName, strings.Join(cols, ", "))
		}
		return nil
	}

	for _, r := range drifted {
		upSQL, downSQL := database.GenerateDriftMigration(r)
		if upSQL == "" {
			continue
		}
		fmt.Printf("-- up.sql for %s.%s\n", r.TableSchema, r.TableName)
		fmt.Println(upSQL)
		fmt.Printf("\n-- down.sql for %s.%s\n", r.TableSchema, r.TableName)
		fmt.Println(downSQL)
		fmt.Println()
	}

	return nil
}

// ── helpers ──────────────────────────────────────────────────────────

func auditBoolMark(v bool) string {
	if v {
		return "ok"
	}
	return "x"
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "~"
}

// ── init ─────────────────────────────────────────────────────────────

func init() {
	dbMigrateCmd.AddCommand(dbMigrateAuditCmd)
	dbMigrateCmd.AddCommand(dbMigrateIdempotentCmd)

	dbDriftScanCmd.Flags().String("schema", "", "Filter to specific schema")
	dbDriftFixCmd.Flags().Bool("all", false, "Fix drift on all drifted tables")

	dbDriftCmd.AddCommand(dbDriftScanCmd)
	dbDriftCmd.AddCommand(dbDriftFixCmd)
	dbCmd.AddCommand(dbDriftCmd)
}
