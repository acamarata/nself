package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nself-org/cli/internal/config"
)

// UpgradeOptions holds flags for tenant plan upgrade.
type UpgradeOptions struct {
	Slug string
	Plan Plan
}

// Upgrade changes a tenant's plan. Stripe Subscription update is handled
// by the billing layer when STRIPE_SECRET_KEY is configured.
func Upgrade(ctx context.Context, cfg *config.Config, opts UpgradeOptions) error {
	if opts.Slug == "" {
		return fmt.Errorf("tenant slug is required")
	}
	if err := validateSlug(opts.Slug); err != nil {
		return err
	}
	if !IsValidPlan(string(opts.Plan)) {
		return fmt.Errorf("invalid plan %q", opts.Plan)
	}

	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	sql := fmt.Sprintf(
		"UPDATE public.tenants SET plan = '%s' WHERE slug = '%s' AND status = 'active' RETURNING id;",
		sanitize(string(opts.Plan)), sanitize(opts.Slug),
	)

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("upgrading tenant %q: %w", opts.Slug, err)
	}

	tenantID := trimOutput(out)
	if tenantID == "" {
		return fmt.Errorf("tenant %q not found or not active", opts.Slug)
	}
	if err := validateUUID(tenantID); err != nil {
		return fmt.Errorf("postgres returned invalid tenant id: %w", err)
	}

	slog.Info("tenant upgraded", "slug", opts.Slug, "plan", opts.Plan)

	// Audit log.
	auditSQL := fmt.Sprintf(
		"INSERT INTO nself_ops.audit_log (tenant_id, action, resource_type, resource_id) "+
			"VALUES ('%s', 'tenant.upgrade', 'tenant', '%s');",
		sanitize(tenantID), sanitize(tenantID),
	)
	auditCmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-c", auditSQL,
	)
	if err := auditCmd.Run(); err != nil {
		slog.Warn("failed to write audit log for tenant upgrade", "error", err)
	}

	slog.Info("tenant upgraded", "slug", opts.Slug, "plan", opts.Plan)
	return nil
}
