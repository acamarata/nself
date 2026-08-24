package commands

// Purpose: `nself gdpr status` and `nself gdpr list-requests` split out of
// gdpr.go (CLI-R12 Batch B mechanical file-size split). Read-only lookups
// against the np_gdpr_requests audit table.
// Inputs: cobra command flags (--request, --status).
// Outputs: a single request's detail, or a table of matching requests.
// Constraints: pure move, no behavior change. gdprCmd (parent) and the
// shared openDB helper remain in gdpr.go.

import (
	"fmt"
	"log"
	"time"

	"github.com/nself-org/cli/internal/gdpr"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var gdprStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of a GDPR request",
	RunE:  runGDPRStatus,
}

func init() {
	gdprStatusCmd.Flags().String("request", "", "Request ID to look up (required)")
	if err := gdprStatusCmd.MarkFlagRequired("request"); err != nil {
		// Programming error: MarkFlagRequired only returns an error when the named
		// flag does not exist on the command. Since "request" is registered on the
		// line above, this can only fire if this code is misedited. It is a
		// bug-in-our-code guard, not a user-input boundary.
		log.Fatalf("gdpr status: mark --request required: %v — this is a code bug, not a config error", err)
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
