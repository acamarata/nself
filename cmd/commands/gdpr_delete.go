package commands

// Purpose: `nself gdpr delete` (and its `forget` alias) split out of
// gdpr.go (CLI-R12 Batch B mechanical file-size split). Cascades a
// delete/anonymize pass across every plugin-registered and core table for
// a user (GDPR Art. 17 erasure).
// Inputs: cobra command flags (--user, --dry-run).
// Outputs: a dry-run row-count report, or a logged np_gdpr_requests row and
// per-table erasure summary.
// Constraints: pure move, no behavior change. gdprCmd (parent) and the
// shared openDB/strPtr helpers remain in gdpr.go. gdprForgetCmd is a true
// interface alias of gdprDeleteCmd (same RunE, duplicated flags) — kept
// alongside it here since they share the same underlying operation.

import (
	"fmt"
	"log"
	"strings"

	"github.com/nself-org/cli/internal/gdpr"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

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
		// Programming error: MarkFlagRequired only returns an error when the named
		// flag does not exist on the command. Since "user" is registered on the
		// line above, this can only fire if this code is misedited. It is a
		// bug-in-our-code guard, not a user-input boundary.
		log.Fatalf("gdpr delete: mark --user required: %v — this is a code bug, not a config error", err)
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

// gdprForgetCmd is a user-friendly alias for gdprDeleteCmd.
// The GDPR "right to be forgotten" (Art. 17) is commonly expressed as "forget" in
// consumer-facing products. This alias keeps the interface familiar without duplicating
// the underlying implementation.
var gdprForgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Alias for 'delete' — right to be forgotten (GDPR Art. 17)",
	Long: `Same as 'nself gdpr delete'. Provided as a user-friendly alias because
the GDPR Art. 17 right-to-erasure is commonly described as the "right to be
forgotten" in consumer-facing products.

Example:
  nself gdpr forget --user abc123
  nself gdpr forget --user abc123 --dry-run`,
	RunE: runGDPRDelete,
}

func init() {
	// Copy flags from gdprDeleteCmd so forget is a true interface alias.
	gdprForgetCmd.Flags().String("user", "", "User ID to erase data for (required)")
	gdprForgetCmd.Flags().Bool("dry-run", false, "List affected rows without deleting anything")
	if err := gdprForgetCmd.MarkFlagRequired("user"); err != nil {
		log.Fatalf("gdpr forget: mark --user required: %v — this is a code bug, not a config error", err)
	}
}
