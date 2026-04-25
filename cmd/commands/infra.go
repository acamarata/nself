package commands

// infra.go — B51/B52: Terraform-based infrastructure provisioning for nSelf.
//
// Status: PLANNED — deferred (requires UD-12 minor release approval).
//
// Commands:
//   nself infra plan    --provider <provider> --domain <domain>
//   nself infra apply   --provider <provider> --domain <domain>
//   nself infra destroy --provider <provider> [--domain <domain>]

import (
	"fmt"

	"github.com/nself-org/cli/internal/infra"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var infraCmd = &cobra.Command{
	Use:   "infra",
	Short: "Provision nSelf infrastructure via Terraform",
	Long: `Provision cloud infrastructure for nSelf using provider-specific Terraform modules.

Supported providers: aws, gcp, azure, hetzner, do, linode

Requires terraform to be installed: https://developer.hashicorp.com/terraform

Examples:
  nself infra plan  --provider hetzner --domain myapp.com
  nself infra apply --provider hetzner --domain myapp.com
  nself infra destroy --provider hetzner`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var infraPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show the Terraform plan for the given provider",
	Long: `Run 'terraform plan' for the nSelf module matching the given provider.

The plan shows what infrastructure will be created without making any changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		providerStr, _ := cmd.Flags().GetString("provider")
		domain, _ := cmd.Flags().GetString("domain")
		stateBucket, _ := cmd.Flags().GetString("state-bucket")
		if providerStr == "" {
			return fmt.Errorf("--provider is required (aws, gcp, azure, hetzner, do, linode)")
		}
		if domain == "" {
			return fmt.Errorf("--domain is required")
		}
		provider := infra.Provider(providerStr)
		if !infra.ValidProviders[provider] {
			return fmt.Errorf("unknown provider %q; supported: aws, gcp, azure, hetzner, do, linode", providerStr)
		}
		opts := infra.PlanOptions{
			Provider:    provider,
			Domain:      domain,
			StateBucket: stateBucket,
		}
		ui.Info(fmt.Sprintf("Planning nSelf infrastructure on %s for %s...", providerStr, domain))
		return infra.Plan(cmd.Context(), opts)
	},
}

var infraApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Provision nSelf infrastructure via Terraform",
	Long: `Run 'terraform apply' for the nSelf module matching the given provider.

Outputs include NSELF_URL, ssh_host, and postgres_url after provisioning.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		providerStr, _ := cmd.Flags().GetString("provider")
		domain, _ := cmd.Flags().GetString("domain")
		stateBucket, _ := cmd.Flags().GetString("state-bucket")
		autoApprove, _ := cmd.Flags().GetBool("auto-approve")
		if providerStr == "" {
			return fmt.Errorf("--provider is required (aws, gcp, azure, hetzner, do, linode)")
		}
		if domain == "" {
			return fmt.Errorf("--domain is required")
		}
		provider := infra.Provider(providerStr)
		if !infra.ValidProviders[provider] {
			return fmt.Errorf("unknown provider %q; supported: aws, gcp, azure, hetzner, do, linode", providerStr)
		}
		opts := infra.ApplyOptions{
			PlanOptions: infra.PlanOptions{
				Provider:    provider,
				Domain:      domain,
				StateBucket: stateBucket,
			},
			AutoApprove: autoApprove,
		}
		ui.Info(fmt.Sprintf("Provisioning nSelf on %s for %s...", providerStr, domain))
		return infra.Apply(cmd.Context(), opts)
	},
}

var infraDestroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy nSelf infrastructure",
	Long: `Run 'terraform destroy' for the nSelf module matching the given provider.

WARNING: This removes all provisioned cloud resources. Data will be lost
unless you have taken a backup via 'nself backup'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		providerStr, _ := cmd.Flags().GetString("provider")
		domain, _ := cmd.Flags().GetString("domain")
		autoApprove, _ := cmd.Flags().GetBool("auto-approve")
		if providerStr == "" {
			return fmt.Errorf("--provider is required")
		}
		provider := infra.Provider(providerStr)
		if !infra.ValidProviders[provider] {
			return fmt.Errorf("unknown provider %q; supported: aws, gcp, azure, hetzner, do, linode", providerStr)
		}
		if !autoApprove {
			ui.Warn("This will destroy all cloud resources. Use --auto-approve to confirm, or run 'nself backup' first.")
			return fmt.Errorf("requires --auto-approve to proceed")
		}
		ui.Info(fmt.Sprintf("Destroying nSelf infrastructure on %s...", providerStr))
		return infra.Destroy(cmd.Context(), provider, domain, "", true)
	},
}

func init() {
	infraPlanCmd.Flags().String("provider", "", "Cloud provider (aws, gcp, azure, hetzner, do, linode)")
	infraPlanCmd.Flags().String("domain", "", "Domain for the nSelf deployment")
	infraPlanCmd.Flags().String("state-bucket", "", "S3/GCS bucket for Terraform state (optional)")

	infraApplyCmd.Flags().String("provider", "", "Cloud provider (aws, gcp, azure, hetzner, do, linode)")
	infraApplyCmd.Flags().String("domain", "", "Domain for the nSelf deployment")
	infraApplyCmd.Flags().String("state-bucket", "", "S3/GCS bucket for Terraform state (optional)")
	infraApplyCmd.Flags().Bool("auto-approve", false, "Skip interactive approval")

	infraDestroyCmd.Flags().String("provider", "", "Cloud provider (aws, gcp, azure, hetzner, do, linode)")
	infraDestroyCmd.Flags().String("domain", "", "Domain for the nSelf deployment")
	infraDestroyCmd.Flags().Bool("auto-approve", false, "Confirm destruction without prompt")

	infraCmd.AddCommand(infraPlanCmd)
	infraCmd.AddCommand(infraApplyCmd)
	infraCmd.AddCommand(infraDestroyCmd)

	RootCmd.AddCommand(infraCmd)
}
