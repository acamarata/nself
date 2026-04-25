package commands

// region.go — B49: multi-region nSelf deployment management.
//
// Status: PLANNED — deferred (requires UD-12 minor release approval).
// All subcommands are gated by feature flag "multi_region_enabled".
// When the flag is off (default), every command prints a clear message
// directing users to enable the flag — no hidden behavior.
//
// Commands:
//   nself region add     --region <id> --pg-url <url> [--redis-url <url>]
//   nself region list
//   nself region status
//   nself region promote --region <id>

import (
	"fmt"

	"github.com/nself-org/cli/internal/flags"
	"github.com/nself-org/cli/internal/region"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var regionCmd = &cobra.Command{
	Use:   "region",
	Short: "Manage multi-region nSelf deployments",
	Long: `Manage multi-region nSelf deployments.

Requires the multi_region_enabled feature flag to be enabled.
Contact nself.org or set the flag via 'nself flag set multi_region_enabled true'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var regionAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a replica region",
	Long:  `Add a read-replica region to the nSelf deployment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireMultiRegionFlag(cmd); err != nil {
			return err
		}
		id, _ := cmd.Flags().GetString("region")
		pgURL, _ := cmd.Flags().GetString("pg-url")
		redisURL, _ := cmd.Flags().GetString("redis-url")
		if id == "" {
			return fmt.Errorf("--region is required")
		}
		if pgURL == "" {
			return fmt.Errorf("--pg-url is required")
		}
		cfg, err := region.Load()
		if err != nil {
			return err
		}
		r := region.Region{
			ID:       id,
			PgURL:    pgURL,
			RedisURL: redisURL,
		}
		if err := cfg.Add(r); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		ui.Success(fmt.Sprintf("Region %q added. Run 'nself region status' to verify replication.", id))
		return nil
	},
}

var regionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured regions",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireMultiRegionFlag(cmd); err != nil {
			return err
		}
		cfg, err := region.Load()
		if err != nil {
			return err
		}
		if len(cfg.Regions) == 0 {
			fmt.Println("No regions configured.")
			return nil
		}
		for _, r := range cfg.Regions {
			primary := ""
			if r.Primary {
				primary = " (primary)"
			}
			fmt.Printf("  %s%s  pg=%s\n", r.ID, primary, r.PgURL)
		}
		return nil
	},
}

var regionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show replication and routing status for all regions",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireMultiRegionFlag(cmd); err != nil {
			return err
		}
		cfg, err := region.Load()
		if err != nil {
			return err
		}
		primary := cfg.Primary()
		if primary == nil {
			fmt.Println("No primary region configured.")
			return nil
		}
		fmt.Printf("Primary region : %s\n", primary.ID)
		fmt.Printf("Routing        : %s\n", cfg.RoutingStrategy)
		fmt.Printf("Replicas       : %s\n", cfg.ReplicaIDs())
		fmt.Println()
		fmt.Println("WAL replication lag check is performed at runtime via the nSelf health endpoint.")
		return nil
	},
}

var regionPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a replica to primary (failover)",
	Long: `Promote a replica region to become the new primary.

The former primary is demoted to a replica. DNS routing will shift to
the new primary within the configured TTL (60 seconds by default).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireMultiRegionFlag(cmd); err != nil {
			return err
		}
		id, _ := cmd.Flags().GetString("region")
		if id == "" {
			return fmt.Errorf("--region is required")
		}
		cfg, err := region.Load()
		if err != nil {
			return err
		}
		if err := cfg.Promote(id); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		ui.Success(fmt.Sprintf("Region %q promoted to primary. DNS will converge within 60 seconds.", id))
		return nil
	},
}

// requireMultiRegionFlag returns an error if the multi_region_enabled feature
// flag is not set. This gates all region subcommands.
func requireMultiRegionFlag(cmd *cobra.Command) error {
	client := flags.NewClient("")
	flag, err := client.Get(cmd.Context(), region.FeatureFlag)
	if err != nil || flag == nil || !flag.Enabled {
		// Flag plugin unavailable or flag disabled — fail with a clear message.
		return fmt.Errorf("multi-region is not enabled on this instance.\n\nEnable with: nself flag set %s true", region.FeatureFlag)
	}
	return nil
}

func init() {
	regionAddCmd.Flags().String("region", "", "Region identifier (e.g. hel1)")
	regionAddCmd.Flags().String("pg-url", "", "Postgres URL for the replica")
	regionAddCmd.Flags().String("redis-url", "", "Redis URL for the region (optional)")

	regionPromoteCmd.Flags().String("region", "", "Region ID to promote to primary")

	regionCmd.AddCommand(regionAddCmd)
	regionCmd.AddCommand(regionListCmd)
	regionCmd.AddCommand(regionStatusCmd)
	regionCmd.AddCommand(regionPromoteCmd)

	RootCmd.AddCommand(regionCmd)
}
