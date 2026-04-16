package commands

import (
	"fmt"

	"github.com/nself-org/cli/internal/tenant"
	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "Billing operations: usage, invoice-preview, report, retry-event",
	Long: `Per-tenant billing and usage metering.

Subcommands:
  usage            Show usage metrics for a tenant
  invoice-preview  Preview next Stripe invoice (requires STRIPE_SECRET_KEY)
  report           Generate billing report across tenants
  retry-event      Re-enqueue a failed Stripe outbox event`,
	// Only run when invoked without a subcommand. Cobra dispatches known
	// subcommands to their own RunE before reaching here. Unknown args bubble
	// up as an error so users see "unknown command" rather than silent help.
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return fmt.Errorf("unknown billing subcommand %q; run 'nself billing --help' for the list", args[0])
	},
}

// ── billing usage ──────────────────────────────────────────────────

var billingUsageCmd = &cobra.Command{
	Use:   "usage <tenant-id>",
	Short: "Show usage metrics for a tenant",
	Args:  cobra.ExactArgs(1),
	RunE:  runBillingUsage,
}

func runBillingUsage(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	month, _ := cmd.Flags().GetString("month")
	format, _ := cmd.Flags().GetString("format")
	if format == "" {
		format = "csv"
	}

	output, err := tenant.QueryUsage(cmd.Context(), cfg, args[0], month, format)
	if err != nil {
		return fmt.Errorf("billing usage: %w", err)
	}
	fmt.Println(output)
	return nil
}

// ── billing invoice-preview ────────────────────────────────────────

var billingInvoicePreviewCmd = &cobra.Command{
	Use:   "invoice-preview <tenant-id>",
	Short: "Preview next Stripe invoice (requires STRIPE_SECRET_KEY)",
	Args:  cobra.ExactArgs(1),
	RunE:  runBillingInvoicePreview,
}

func runBillingInvoicePreview(_ *cobra.Command, args []string) error {
	return fmt.Errorf("billing invoice-preview is not yet available: Stripe API integration is pending. Use 'nself billing usage %s' for current usage data, or check your Stripe dashboard directly", args[0])
}

// ── billing report ─────────────────────────────────────────────────

var billingReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate billing report across tenants",
	RunE:  runBillingReport,
}

func runBillingReport(cmd *cobra.Command, _ []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	tenantSlug, _ := cmd.Flags().GetString("tenant")
	month, _ := cmd.Flags().GetString("month")
	format, _ := cmd.Flags().GetString("format")
	if format == "" {
		format = "table"
	}

	output, err := tenant.BillingReport(cmd.Context(), cfg, tenant.BillingReportOptions{
		TenantSlug: tenantSlug,
		Month:      month,
		Format:     format,
	})
	if err != nil {
		return fmt.Errorf("billing report: %w", err)
	}
	fmt.Print(output)
	return nil
}

// ── billing retry-event ────────────────────────────────────────────

var billingRetryEventCmd = &cobra.Command{
	Use:   "retry-event <id>",
	Short: "Re-enqueue a failed Stripe outbox event",
	Args:  cobra.ExactArgs(1),
	RunE:  runBillingRetryEvent,
}

func runBillingRetryEvent(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	return tenant.RetryStripeEvent(cmd.Context(), cfg, args[0])
}

// ── init ────────────────────────────────────────────────────────────

func init() {
	// billing usage flags
	billingUsageCmd.Flags().String("month", "", "Filter by month (YYYY-MM)")
	billingUsageCmd.Flags().String("format", "csv", "Output format: csv or json")

	// billing report flags
	billingReportCmd.Flags().String("tenant", "", "Filter by tenant slug")
	billingReportCmd.Flags().String("month", "", "Filter by month (YYYY-MM)")
	billingReportCmd.Flags().String("format", "table", "Output format: table or json")

	// Wire subcommands
	billingCmd.AddCommand(billingUsageCmd)
	billingCmd.AddCommand(billingInvoicePreviewCmd)
	billingCmd.AddCommand(billingReportCmd)
	billingCmd.AddCommand(billingRetryEventCmd)

	RootCmd.AddCommand(billingCmd)
}
