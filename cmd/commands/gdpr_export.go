package commands

// Purpose: `nself gdpr export` split out of gdpr.go (CLI-R12 Batch B
// mechanical file-size split). Builds a GDPR Art. 20 data-portability
// export archive for a single user.
// Inputs: cobra command flags (--user, --format, --output, --notify,
// --dry-run).
// Outputs: a ZIP archive written to --output (or stdout); a logged
// np_gdpr_requests row for audit purposes.
// Constraints: pure move, no behavior change. gdprCmd (parent) and the
// shared openDB/strPtr helpers remain in gdpr.go.

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nself-org/cli/internal/gdpr"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

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
		// Programming error: MarkFlagRequired only returns an error when the named
		// flag does not exist on the command. Since "user" is registered on the
		// line above, this can only fire if this code is misedited. It is a
		// bug-in-our-code guard, not a user-input boundary.
		log.Fatalf("gdpr export: mark --user required: %v — this is a code bug, not a config error", err)
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
