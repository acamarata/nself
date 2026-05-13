package commands

// plugin_audit.go — S1.T08: nself plugin audit-tables
//
// Reads row counts per np_* table per source_account_id.
// Outputs a JSON baseline suitable for diffing before plugin renames.
// Detects missing source_account_id columns and multi-account contamination.
//
// This command is read-only; it never modifies DB state.
// Security-Always-Free Doctrine: runs without a license key.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nself-org/cli/internal/audit"
	"github.com/spf13/cobra"
)

var pluginAuditTablesCmd = &cobra.Command{
	Use:   "audit-tables",
	Short: "Audit np_* table row counts per source_account_id",
	Long: `Read row counts for every np_* table in the public schema,
grouped by source_account_id.

The output is a JSON baseline that can be committed before a plugin rename
and diffed afterwards to confirm no data was lost or contaminated.

Detects:
  - Tables missing source_account_id (cross-app isolation unknown)
  - Multi-account contamination (rows from more than one source_account_id)

Requires NSELF_DB_URL (or DATABASE_URL) to be set.

Examples:
  nself plugin audit-tables
  nself plugin audit-tables > baseline.json
  diff baseline.json <(nself plugin audit-tables)`,
	RunE: runPluginAuditTables,
}

func init() {
	pluginAuditTablesCmd.Flags().Bool("json", true, "Output as JSON (default)")
	pluginCmd.AddCommand(pluginAuditTablesCmd)
}

func runPluginAuditTables(cmd *cobra.Command, _ []string) error {
	report, err := audit.RunTableAudit(context.Background())
	if err != nil {
		return fmt.Errorf("audit-tables: %w", err)
	}

	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal audit report: %w", err)
	}

	fmt.Println(string(out))
	return nil
}
