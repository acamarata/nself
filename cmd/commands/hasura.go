package commands

// Purpose: top-level "nself hasura ..." aliases for the most-used metadata
// operations (apply/export/diff), so operators don't have to remember these
// live under "nself db hasura ...". Thin wrappers only — every RunE here
// delegates to the existing runDBHasura* implementation (db_hasura.go /
// db_commands_ext.go) so there is exactly one behavior to keep correct.
// Constraints: FIX-CLI-3 (P6 2026-09-04); new file, single init() registering
// against RootCmd, to keep the footprint isolated from concurrent start*.go /
// internal/compose work tonight (coordination note).

import (
	"github.com/spf13/cobra"
)

var hasuraCmd = &cobra.Command{
	Use:   "hasura",
	Short: "Hasura metadata operations (alias for 'nself db hasura')",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var hasuraMetadataCmd = &cobra.Command{
	Use:   "metadata",
	Short: "Manage Hasura metadata",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var hasuraMetadataApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Hasura metadata (alias for 'nself db hasura metadata apply')",
	Long:  dbHasuraMetadataApplyCmd.Long,
	RunE:  runDBHasuraMetadataApply,
}

var hasuraMetadataExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export Hasura metadata to git-friendly sorted YAML (alias for 'nself db hasura metadata export')",
	RunE:  runDBHasuraMetadataExport,
}

var hasuraDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare live Hasura metadata against on-disk files (alias for 'nself db hasura diff')",
	RunE:  runDBHasuraDiff,
}

func init() {
	// Same remote-targeting support (--env/--server) as its "db hasura
	// metadata apply" counterpart — see db_remote.go.
	addDBRemoteFlags(hasuraMetadataApplyCmd)

	hasuraMetadataCmd.AddCommand(hasuraMetadataApplyCmd)
	hasuraMetadataCmd.AddCommand(hasuraMetadataExportCmd)

	hasuraCmd.AddCommand(hasuraMetadataCmd)
	hasuraCmd.AddCommand(hasuraDiffCmd)

	RootCmd.AddCommand(hasuraCmd)
}
