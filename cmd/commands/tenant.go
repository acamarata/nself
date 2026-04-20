package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nself-org/cli/internal/tenant"
	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "[PREVIEW] Tenant management: create, upgrade, suspend, destroy, audit",
	Long: `[PREVIEW] Multi-tenancy tenant lifecycle management.

PREVIEW (v1.0.9): the tenant CLI works at the data and RLS layer.
Provisioning automation, Stripe billing charges, license revocation
on destroy, and runtime suspension are PLANNED for v1.1.0+. Do NOT
use for paying-customer onboarding in v1.0.9 production deployments.

Subcommands:
  create    Create a new tenant
  upgrade   Change a tenant's plan
  suspend   Suspend a tenant
  destroy   Hard-delete a tenant (with confirmation)
  audit     Query tenant audit log`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── tenant create ──────────────────────────────────────────────────

var tenantCreateCmd = &cobra.Command{
	Use:   "create <slug>",
	Short: "Create a new tenant",
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantCreate,
}

func runTenantCreate(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	plan, _ := cmd.Flags().GetString("plan")
	if plan == "" {
		plan = "basic"
	}
	return tenant.Create(cmd.Context(), cfg, tenant.CreateOptions{
		Slug: args[0],
		Plan: tenant.Plan(plan),
	})
}

// ── tenant upgrade ─────────────────────────────────────────────────

var tenantUpgradeCmd = &cobra.Command{
	Use:   "upgrade <slug>",
	Short: "Change a tenant's plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantUpgrade,
}

func runTenantUpgrade(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	plan, _ := cmd.Flags().GetString("plan")
	if plan == "" {
		return fmt.Errorf("--plan is required")
	}
	return tenant.Upgrade(cmd.Context(), cfg, tenant.UpgradeOptions{
		Slug: args[0],
		Plan: tenant.Plan(plan),
	})
}

// ── tenant suspend ─────────────────────────────────────────────────

var tenantSuspendCmd = &cobra.Command{
	Use:   "suspend <slug>",
	Short: "Suspend a tenant",
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantSuspend,
}

func runTenantSuspend(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	reason, _ := cmd.Flags().GetString("reason")
	return tenant.Suspend(cmd.Context(), cfg, tenant.SuspendOptions{
		Slug:   args[0],
		Reason: reason,
	})
}

// ── tenant destroy ─────────────────────────────────────────────────

var tenantDestroyCmd = &cobra.Command{
	Use:   "destroy <slug>",
	Short: "Hard-delete a tenant (DESTRUCTIVE)",
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantDestroy,
}

func runTenantDestroy(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	confirmName, _ := cmd.Flags().GetString("confirm-name")
	return tenant.Destroy(cmd.Context(), cfg, tenant.DestroyOptions{
		Slug:        args[0],
		ConfirmName: confirmName,
	})
}

// ── tenant audit ───────────────────────────────────────────────────

var tenantAuditCmd = &cobra.Command{
	Use:   "audit <tenant-id>",
	Short: "Query tenant audit log",
	Args:  cobra.ExactArgs(1),
	RunE:  runTenantAudit,
}

func runTenantAudit(cmd *cobra.Command, args []string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}
	since, _ := cmd.Flags().GetString("since")
	format, _ := cmd.Flags().GetString("format")

	entries, err := tenant.Audit(cmd.Context(), cfg, tenant.AuditOptions{
		TenantID: args[0],
		Since:    since,
		Format:   format,
	})
	if err != nil {
		return fmt.Errorf("tenant audit: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No audit entries found.")
		return nil
	}

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	// Table format.
	fmt.Printf("%-36s %-20s %-15s %-15s %s\n", "ID", "ACTION", "RESOURCE_TYPE", "RESOURCE_ID", "CREATED_AT")
	for _, e := range entries {
		fmt.Printf("%-36s %-20s %-15s %-15s %s\n",
			e.ID, e.Action, e.ResourceType, e.ResourceID,
			e.CreatedAt.Format(time.RFC3339),
		)
	}
	return nil
}

// ── init ────────────────────────────────────────────────────────────

func init() {
	// tenant create flags
	tenantCreateCmd.Flags().String("plan", "basic", "Tenant plan: basic|pro|elite|business|business-plus|enterprise")

	// tenant upgrade flags
	tenantUpgradeCmd.Flags().String("plan", "", "Target plan: basic|pro|elite|business|business-plus|enterprise")

	// tenant suspend flags
	tenantSuspendCmd.Flags().String("reason", "", "Reason for suspension (required)")

	// tenant destroy flags
	tenantDestroyCmd.Flags().String("confirm-name", "", "Type the tenant slug to confirm destruction")

	// tenant audit flags
	tenantAuditCmd.Flags().String("since", "", "Time window, e.g. 7d, 24h, 2w")
	tenantAuditCmd.Flags().String("format", "table", "Output format: table or json")

	// Wire subcommands
	tenantCmd.AddCommand(tenantCreateCmd)
	tenantCmd.AddCommand(tenantUpgradeCmd)
	tenantCmd.AddCommand(tenantSuspendCmd)
	tenantCmd.AddCommand(tenantDestroyCmd)
	tenantCmd.AddCommand(tenantAuditCmd)

	RootCmd.AddCommand(tenantCmd)
}
