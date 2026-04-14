package commands

import (
	"fmt"

	"github.com/nself-org/cli/internal/dr"
	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────

var drCmd = &cobra.Command{
	Use:   "dr",
	Short: "Disaster recovery operations: drill, promote-standby, reconfigure-dns, rollback, fence",
	Long: `Disaster recovery operations for nSelf projects.

Subcommands:
  drill              Execute a DR drill (cold-start, region-failover, data-corruption)
  promote-standby    Promote warm standby to primary
  reconfigure-dns    Update DNS records to point to new primary
  rollback           Demote promoted standby, resync from original primary
  fence              Set read-only flag in Redis for split-brain prevention`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── dr drill ────────────────────────────────────────────────────────

var drDrillCmd = &cobra.Command{
	Use:   "drill",
	Short: "Execute a DR drill",
	RunE:  runDRDrill,
}

func runDRDrill(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	scenario, _ := cmd.Flags().GetString("scenario")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	result, err := dr.Drill(cmd.Context(), cfg, dr.DrillOptions{
		Scenario: dr.Scenario(scenario),
		DryRun:   dryRun,
	})
	if err != nil {
		return fmt.Errorf("dr drill: %w", err)
	}

	output, _ := dr.FormatDrillResult(result, "json")
	fmt.Println(output)
	return nil
}

// ── dr promote-standby ─────────────────────────────────────────────

var drPromoteCmd = &cobra.Command{
	Use:   "promote-standby",
	Short: "Promote warm standby to primary",
	RunE:  runDRPromote,
}

func runDRPromote(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	region, _ := cmd.Flags().GetString("region")
	yes, _ := cmd.Flags().GetBool("yes")

	if !yes {
		if err := requireProductionConfirmation(cfg.ProjectName); err != nil {
			return err
		}
	}

	if err := dr.PromoteStandby(cmd.Context(), cfg, dr.PromoteOptions{
		Region: region,
		Yes:    yes,
	}); err != nil {
		return fmt.Errorf("dr promote: %w", err)
	}
	fmt.Println("Standby promoted to primary.")
	return nil
}

// ── dr reconfigure-dns ─────────────────────────────────────────────

var drReconfigureDNSCmd = &cobra.Command{
	Use:   "reconfigure-dns",
	Short: "Update DNS records to point to new primary",
	RunE:  runDRReconfigureDNS,
}

func runDRReconfigureDNS(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	ip, _ := cmd.Flags().GetString("ip")
	if ip == "" {
		return fmt.Errorf("--ip flag is required")
	}

	if err := dr.ReconfigureDNS(cmd.Context(), cfg, ip); err != nil {
		return fmt.Errorf("dr reconfigure-dns: %w", err)
	}
	fmt.Println("DNS reconfigured.")
	return nil
}

// ── dr rollback ────────────────────────────────────────────────────

var drRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Demote promoted standby and resync from original primary",
	RunE:  runDRRollback,
}

func runDRRollback(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	if err := dr.Rollback(cmd.Context(), cfg); err != nil {
		return fmt.Errorf("dr rollback: %w", err)
	}
	fmt.Println("DR rollback complete.")
	return nil
}

// ── dr fence ───────────────────────────────────────────────────────

var drFenceCmd = &cobra.Command{
	Use:   "fence",
	Short: "Set read-only flag in Redis for split-brain prevention",
	RunE:  runDRFence,
}

func runDRFence(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	if err := dr.Fence(cmd.Context(), cfg); err != nil {
		return fmt.Errorf("dr fence: %w", err)
	}
	fmt.Println("Split-brain fence set.")
	return nil
}

// ── init ────────────────────────────────────────────────────────────

func init() {
	// dr drill flags
	drDrillCmd.Flags().String("scenario", "cold-start", "Drill scenario: cold-start, region-failover, data-corruption")
	drDrillCmd.Flags().Bool("dry-run", false, "Preview only")

	// dr promote-standby flags
	drPromoteCmd.Flags().String("region", "", "Target region for promotion")
	drPromoteCmd.Flags().Bool("yes", false, "Skip confirmation")

	// dr reconfigure-dns flags
	drReconfigureDNSCmd.Flags().String("ip", "", "New primary IP address")

	// Wire subcommands
	drCmd.AddCommand(drDrillCmd)
	drCmd.AddCommand(drPromoteCmd)
	drCmd.AddCommand(drReconfigureDNSCmd)
	drCmd.AddCommand(drRollbackCmd)
	drCmd.AddCommand(drFenceCmd)

	RootCmd.AddCommand(drCmd)
}
