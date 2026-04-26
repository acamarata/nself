package commands

import (
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/claw"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ui"
	"github.com/nself-org/cli/internal/version"

	"github.com/spf13/cobra"
)

var (
	clawMigrateFrom string
	clawMigrateTo   string
)

var clawMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending claw schema migrations",
	Long: `Apply pending claw database schema migrations in order.

By default the command reads the current schema version from the database
and applies every pending migration up to the latest available.

Use --from to skip migrations at or before a specific version.
Use --to to stop after applying a specific target version (default: current nself version).

Examples:
  nself claw migrate                     # apply all pending claw migrations
  nself claw migrate --to 005            # apply up to and including 005_*.sql
  nself claw migrate --from 002 --to 005 # apply only 003, 004, and 005`,
	RunE: runClawMigrate,
}

func init() {
	clawMigrateCmd.Flags().StringVar(&clawMigrateFrom, "from", "",
		"Skip migrations at or before this version (e.g. 003_add_index.sql)")
	clawMigrateCmd.Flags().StringVar(&clawMigrateTo, "to", version.Version,
		"Stop after applying this version (default: current nself version)")
}

func runClawMigrate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("loading project config: %w\nRun this command from your nself project directory", err)
	}

	ui.Section("claw migrate")
	ui.Info(fmt.Sprintf("From: %q  To: %q", clawMigrateFrom, clawMigrateTo))

	result, err := claw.Migrate(ctx, cfg, clawMigrateFrom, clawMigrateTo)
	if err != nil {
		return err
	}

	fmt.Println()
	if len(result.Applied) == 0 {
		ui.Success("No pending claw migrations — schema is up to date")
		return nil
	}

	ui.Success(fmt.Sprintf("Applied %d migration(s):", len(result.Applied)))
	for _, name := range result.Applied {
		fmt.Printf("  %s %s\n", ui.C(ui.Green, ui.IconSuccess), name)
	}
	if len(result.Skipped) > 0 {
		ui.Dimmed(fmt.Sprintf("Skipped %d (already applied or outside range)", len(result.Skipped)))
	}
	return nil
}
