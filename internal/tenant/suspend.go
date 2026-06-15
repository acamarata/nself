package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nself-org/cli/internal/config"
)

// SuspendOptions holds flags for tenant suspension.
type SuspendOptions struct {
	Slug   string
	Reason string
}

// Suspend marks a tenant as suspended with a reason. Stripe subscription
// pause is handled by the billing layer when STRIPE_SECRET_KEY is configured.
func Suspend(ctx context.Context, cfg *config.Config, opts SuspendOptions) error {
	if opts.Slug == "" {
		return fmt.Errorf("tenant slug is required")
	}
	if err := validateSlug(opts.Slug); err != nil {
		return err
	}
	if opts.Reason == "" {
		return fmt.Errorf("suspension reason is required (--reason)")
	}
	if err := validateReason(opts.Reason); err != nil {
		return err
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
		"UPDATE public.tenants SET status = 'suspended', suspended_at = now(), suspend_reason = '%s' "+
			"WHERE slug = '%s' AND status = 'active' RETURNING id;",
		sanitize(opts.Reason), sanitize(opts.Slug),
	)

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("suspending tenant %q: %w", opts.Slug, err)
	}

	tenantID := trimOutput(out)
	if tenantID == "" {
		return fmt.Errorf("tenant %q not found or not active", opts.Slug)
	}
	if err := validateUUID(tenantID); err != nil {
		return fmt.Errorf("postgres returned invalid tenant id: %w", err)
	}

	slog.Info("tenant suspended", "slug", opts.Slug, "reason", opts.Reason)

	// Audit log with required reason.
	auditSQL := fmt.Sprintf(
		"INSERT INTO nself_ops.audit_log (tenant_id, action, resource_type, resource_id, reason) "+
			"VALUES ('%s', 'tenant.suspend', 'tenant', '%s', '%s');",
		sanitize(tenantID), sanitize(tenantID), sanitize(opts.Reason),
	)
	auditCmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-c", auditSQL,
	)
	if err := auditCmd.Run(); err != nil {
		slog.Warn("failed to write audit log for tenant suspension", "error", err)
	}

	slog.Info("tenant suspended", "slug", opts.Slug, "reason", opts.Reason)
	return nil
}
