// Package commands — gdpr.go
//
// `nself gdpr` implements GDPR data portability (Art. 20) and right-to-erasure
// (Art. 17) flows for self-hosted nSelf instances.
//
// Subcommands:
//
//	nself gdpr export --user <id>         Build an export archive for a user
//	nself gdpr delete --user <id>         Cascade-delete/anonymize a user's data
//	nself gdpr status --request <id>      Show status of a GDPR request
//	nself gdpr list-requests              List all GDPR requests
package commands

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/gdpr"
	"github.com/nself-org/cli/internal/ui"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var gdprCmd = &cobra.Command{
	Use:   "gdpr",
	Short: "GDPR data portability and right-to-erasure (Art. 20, 17)",
	Long: `GDPR compliance tools for self-hosted nSelf instances.

Subcommands:
  nself gdpr export        Export all data for a user (Art. 20 portability)
  nself gdpr delete        Delete or anonymize all data for a user (Art. 17 erasure)
  nself gdpr status        Check the status of a GDPR request
  nself gdpr list-requests List all open and completed GDPR requests

All GDPR requests are logged to np_gdpr_requests for audit purposes.
That table is append-only: rows are never deleted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ---------- export ----------

var gdprExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all data for a user (GDPR Art. 20 portability)",
	Long: `Build a ZIP archive containing all personal data held for a given user,
across every registered plugin and core nSelf tables.

The archive is written to the path given by --output (default: stdout as binary).
Use --format csv to get CSV files inside the archive instead of JSON.

Example:
  nself gdpr export --user abc123 --output /tmp/export.zip
  nself gdpr export --user abc123 --format csv --output /tmp/export.zip`,
	RunE: runGDPRExport,
}

func init() {
	gdprExportCmd.Flags().String("user", "", "User ID to export data for (required)")
	gdprExportCmd.Flags().String("format", "json", "Output format: json or csv")
	gdprExportCmd.Flags().String("output", "", "Write archive to this file (default: ./gdpr-export-<user>.zip)")
	gdprExportCmd.Flags().String("notify", "", "Email address to notify when export is complete (optional)")
	gdprExportCmd.Flags().Bool("dry-run", false, "Print what would be exported without generating the archive")
	if err := gdprExportCmd.MarkFlagRequired("user"); err != nil {
		panic(err)
	}
}

func runGDPRExport(cmd *cobra.Command, _ []string) error {
	userID, _ := cmd.Flags().GetString("user")
	formatStr, _ := cmd.Flags().GetString("format")
	outputPath, _ := cmd.Flags().GetString("output")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	format := gdpr.FormatJSON
	if formatStr == "csv" {
		format = gdpr.FormatCSV
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := cmd.Context()

	requestID, err := gdpr.CreateRequest(ctx, db, gdpr.RequestTypeExport, gdpr.SubjectTypeUser, userID, nil)
	if err != nil {
		return fmt.Errorf("gdpr export: %w", err)
	}

	ui.Info(fmt.Sprintf("GDPR export request created: %s (30-day deadline: %s)",
		requestID, time.Now().AddDate(0, 0, gdpr.DeadlineDays).Format("2006-01-02")))

	if err := gdpr.UpdateRequestStatus(ctx, db, requestID, gdpr.StatusProcessing, nil, nil, nil); err != nil {
		return err
	}

	result, err := gdpr.ExportUserData(ctx, db, requestID, userID, format, dryRun)
	if err != nil {
		note := err.Error()
		_ = gdpr.UpdateRequestStatus(ctx, db, requestID, gdpr.StatusFailed, nil, nil, &note)
		return fmt.Errorf("gdpr export: %w", err)
	}

	if dryRun {
		ui.Info(fmt.Sprintf("Dry-run complete. No archive generated. Request ID: %s", requestID))
		_ = gdpr.UpdateRequestStatus(ctx, db, requestID, gdpr.StatusComplete, nil, nil, strPtr("dry-run"))
		return nil
	}

	if outputPath == "" {
		outputPath = fmt.Sprintf("gdpr-export-%s.zip", userID)
	}
	if err := os.WriteFile(outputPath, result.Data, 0600); err != nil {
		return fmt.Errorf("gdpr export: write file: %w", err)
	}

	expires := time.Now().Add(7 * 24 * time.Hour)
	note := fmt.Sprintf("archive written to %s", outputPath)
	if err := gdpr.UpdateRequestStatus(ctx, db, requestID, gdpr.StatusComplete, &outputPath, &expires, &note); err != nil {
		return err
	}

	ui.Success(fmt.Sprintf("Export complete. Archive: %s (request: %s)", outputPath, requestID))
	return nil
}

// ---------- delete ----------

var gdprDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete or anonymize all data for a user (GDPR Art. 17 erasure)",
	Long: `Execute a cascading delete/anonymization across every plugin-registered
table and core nSelf tables for the given user.

Use --dry-run to see which tables and row counts would be affected before
committing. Without --dry-run the operation executes immediately.

Example:
  nself gdpr delete --user abc123 --dry-run
  nself gdpr delete --user abc123`,
	RunE: runGDPRDelete,
}

func init() {
	gdprDeleteCmd.Flags().String("user", "", "User ID to erase data for (required)")
	gdprDeleteCmd.Flags().Bool("dry-run", false, "List affected rows without deleting anything")
	if err := gdprDeleteCmd.MarkFlagRequired("user"); err != nil {
		panic(err)
	}
}

func runGDPRDelete(cmd *cobra.Command, _ []string) error {
	userID, _ := cmd.Flags().GetString("user")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := cmd.Context()

	if dryRun {
		results, err := gdpr.DryRunUserDelete(ctx, db, userID)
		if err != nil {
			return err
		}
		ui.Info(fmt.Sprintf("Dry-run: rows that would be deleted/anonymized for user %s", userID))
		for _, r := range results {
			ui.Info(fmt.Sprintf("  [%s] %s.%s = %s  (%d rows)", r.Plugin, r.Table, r.UserCol, r.Strategy, r.RowCount))
		}
		return nil
	}

	requestID, err := gdpr.CreateRequest(ctx, db, gdpr.RequestTypeDelete, gdpr.SubjectTypeUser, userID, nil)
	if err != nil {
		return fmt.Errorf("gdpr delete: %w", err)
	}

	ui.Info(fmt.Sprintf("GDPR erasure request created: %s", requestID))
	if err := gdpr.UpdateRequestStatus(ctx, db, requestID, gdpr.StatusProcessing, nil, nil, nil); err != nil {
		return err
	}

	processed, errs := gdpr.DeleteUserData(ctx, db, userID)

	var note string
	status := gdpr.StatusComplete
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		note = strings.Join(msgs, "; ")
		status = gdpr.StatusFailed
		ui.Warn(fmt.Sprintf("Erasure completed with %d error(s): %s", len(errs), note))
	}

	if err := gdpr.UpdateRequestStatus(ctx, db, requestID, status, nil, nil, strPtr(note)); err != nil {
		return err
	}

	ui.Success(fmt.Sprintf("Erasure complete. %d tables processed. Request: %s", processed, requestID))
	return nil
}

// ---------- status ----------

var gdprStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of a GDPR request",
	RunE:  runGDPRStatus,
}

func init() {
	gdprStatusCmd.Flags().String("request", "", "Request ID to look up (required)")
	if err := gdprStatusCmd.MarkFlagRequired("request"); err != nil {
		panic(err)
	}
}

func runGDPRStatus(cmd *cobra.Command, _ []string) error {
	requestID, _ := cmd.Flags().GetString("request")

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	req, err := gdpr.GetRequest(cmd.Context(), db, requestID)
	if err != nil {
		return err
	}

	ui.Info(fmt.Sprintf("Request: %s", req.ID))
	ui.Info(fmt.Sprintf("Type:    %s / %s", req.RequestType, req.SubjectType))
	ui.Info(fmt.Sprintf("Subject: %s", req.SubjectID))
	ui.Info(fmt.Sprintf("Status:  %s", req.Status))
	ui.Info(fmt.Sprintf("Created: %s", req.RequestedAt.Format(time.RFC3339)))
	ui.Info(fmt.Sprintf("Deadline:%s", req.Deadline.Format("2006-01-02")))
	if req.ArtifactURL != nil {
		ui.Info(fmt.Sprintf("Archive: %s", *req.ArtifactURL))
	}
	if req.Notes != nil && *req.Notes != "" {
		ui.Info(fmt.Sprintf("Notes:   %s", *req.Notes))
	}
	return nil
}

// ---------- list-requests ----------

var gdprListCmd = &cobra.Command{
	Use:   "list-requests",
	Short: "List all GDPR requests",
	RunE:  runGDPRList,
}

func init() {
	gdprListCmd.Flags().String("status", "", "Filter by status: pending, processing, complete, failed")
}

func runGDPRList(cmd *cobra.Command, _ []string) error {
	statusFilter, _ := cmd.Flags().GetString("status")

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	requests, err := gdpr.ListRequests(cmd.Context(), db, statusFilter)
	if err != nil {
		return err
	}

	if len(requests) == 0 {
		ui.Info("No GDPR requests found.")
		return nil
	}

	ui.Info(fmt.Sprintf("%-36s  %-8s  %-8s  %-12s  %-10s  %s",
		"ID", "TYPE", "SUBJECT", "STATUS", "DEADLINE", "SUBJECT_ID"))
	for _, r := range requests {
		ui.Info(fmt.Sprintf("%-36s  %-8s  %-8s  %-12s  %-10s  %s",
			r.ID, r.RequestType, r.SubjectType, r.Status,
			r.Deadline.Format("2006-01-02"), r.SubjectID))
	}
	return nil
}

// ---------- helpers ----------

func init() {
	gdprCmd.AddCommand(gdprExportCmd)
	gdprCmd.AddCommand(gdprDeleteCmd)
	gdprCmd.AddCommand(gdprStatusCmd)
	gdprCmd.AddCommand(gdprListCmd)
	RootCmd.AddCommand(gdprCmd)
}

// openDB opens a Postgres connection from DATABASE_URL.
func openDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set; run `nself env` to verify your environment")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("gdpr: open database: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// strPtr returns a pointer to a string — used for optional nullable fields.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
