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
	// Ops-only flags short-circuit before loading project config: these are
	// invoked on nclaw-prod (or any host with /etc/nself/dr.env) where the
	// local working directory does not need to be a nSelf project root.
	if renderCI, _ := cmd.Flags().GetBool("render-cloud-init"); renderCI {
		yaml, err := dr.RenderCloudInit(dr.CloudInitParams{
			DrillID:      "render-preview",
			SSHPublicKey: "ssh-ed25519 AAAA...preview",
		})
		if err != nil {
			return err
		}
		fmt.Print(yaml)
		return nil
	}
	if renderAlerts, _ := cmd.Flags().GetBool("render-alerts"); renderAlerts {
		fmt.Print(dr.DrillAlertRuleYAML)
		return nil
	}
	if installCron, _ := cmd.Flags().GetBool("install-cron"); installCron {
		schedule, _ := cmd.Flags().GetString("schedule")
		project, _ := cmd.Flags().GetString("hetzner-project")
		vmType, _ := cmd.Flags().GetString("vm-type")
		sshKey, _ := cmd.Flags().GetString("ssh-key")
		region, _ := cmd.Flags().GetString("region")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return dr.InstallSystemdUnits(dr.SystemdInstallOptions{
			Schedule:       schedule,
			HetznerProject: project,
			VMType:         vmType,
			SSHKey:         sshKey,
			Region:         region,
			DryRun:         dryRun,
		})
	}

	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	scenario, _ := cmd.Flags().GetString("scenario")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	now, _ := cmd.Flags().GetBool("now")

	// --now is the cron/manual entrypoint for the full provision-restore-smoke
	// lifecycle. It defaults to cold-start if no scenario is provided.
	if now && scenario == "" {
		scenario = string(dr.ScenarioColdStart)
	}

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
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Println("(dry-run) Would promote warm standby to primary and update DNS records.")
		return nil
	}

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
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	ip, _ := cmd.Flags().GetString("ip")

	if dryRun {
		if ip == "" {
			fmt.Println("(dry-run) Would update DNS records to point to new primary (--ip required for live run).")
		} else {
			fmt.Printf("(dry-run) Would update DNS A records to point to %s.\n", ip)
		}
		return nil
	}

	if ip == "" {
		return fmt.Errorf("--ip flag is required")
	}

	cfg, err := loadProjectConfig()
	if err != nil {
		return err
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
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Println("(dry-run) Would demote standby and resync replication from original primary.")
		return nil
	}

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
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Println("(dry-run) Would set read-only split-brain fence in Redis (TTL: 3600s).")
		return nil
	}

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
	drDrillCmd.Flags().Bool("now", false, "Run a full provision-restore-smoke-destroy drill immediately (cron entrypoint)")
	drDrillCmd.Flags().Bool("install-cron", false, "Install the monthly drill systemd timer (nself-dr-drill.timer)")
	drDrillCmd.Flags().String("schedule", "monthly", "Drill cadence for --install-cron (only \"monthly\" supported)")
	drDrillCmd.Flags().String("hetzner-project", "", "Hetzner project name (e.g. camarata) used for drill VM")
	drDrillCmd.Flags().String("vm-type", "cx22", "Hetzner server type for drill VM")
	drDrillCmd.Flags().String("ssh-key", "/root/.config/nself/dr-key.pub", "Public SSH key path injected into drill VM")
	drDrillCmd.Flags().String("region", "fsn1", "Hetzner location for drill VM provisioning")
	drDrillCmd.Flags().Bool("render-cloud-init", false, "Print the cloud-init user-data template and exit")
	drDrillCmd.Flags().Bool("render-alerts", false, "Print the Prometheus/Alertmanager DRDrillFailed rule and exit")

	// dr promote-standby flags
	drPromoteCmd.Flags().String("region", "", "Target region for promotion")
	drPromoteCmd.Flags().Bool("yes", false, "Skip confirmation")
	drPromoteCmd.Flags().Bool("dry-run", false, "Preview steps without executing (irreversible operation guard)")

	// dr reconfigure-dns flags
	drReconfigureDNSCmd.Flags().String("ip", "", "New primary IP address")
	drReconfigureDNSCmd.Flags().Bool("dry-run", false, "Preview DNS change without executing")

	// dr rollback flags
	drRollbackCmd.Flags().Bool("dry-run", false, "Preview rollback steps without executing (irreversible operation guard)")

	// dr fence flags
	drFenceCmd.Flags().Bool("dry-run", false, "Preview fence operation without executing")

	// Wire subcommands
	drCmd.AddCommand(drDrillCmd)
	drCmd.AddCommand(drPromoteCmd)
	drCmd.AddCommand(drReconfigureDNSCmd)
	drCmd.AddCommand(drRollbackCmd)
	drCmd.AddCommand(drFenceCmd)

	RootCmd.AddCommand(drCmd)
}
