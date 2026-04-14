package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nself-org/cli/internal/config"
)

// CreateOptions holds flags for tenant creation.
type CreateOptions struct {
	Slug string
	Plan Plan
}

// Create provisions a new tenant: inserts the tenants row and logs the
// creation in audit_log. Stripe Customer/Subscription creation is handled
// separately by the billing integration layer (requires STRIPE_SECRET_KEY).
func Create(ctx context.Context, cfg *config.Config, opts CreateOptions) error {
	if opts.Slug == "" {
		return fmt.Errorf("tenant slug is required")
	}
	if !IsValidPlan(string(opts.Plan)) {
		return fmt.Errorf("invalid plan %q; valid plans: basic, pro, elite, business, business-plus, enterprise", opts.Plan)
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

	// Insert tenant row.
	insertSQL := fmt.Sprintf(
		"INSERT INTO public.tenants (slug, plan, status) VALUES ('%s', '%s', 'active') RETURNING id;",
		sanitize(opts.Slug), sanitize(string(opts.Plan)),
	)

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", insertSQL,
	)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("creating tenant %q: %w", opts.Slug, err)
	}

	tenantID := trimOutput(out)
	slog.Info("tenant created", "slug", opts.Slug, "plan", opts.Plan, "id", tenantID)

	// Write audit log entry.
	auditSQL := fmt.Sprintf(
		"INSERT INTO nself_ops.audit_log (tenant_id, action, resource_type, resource_id) "+
			"VALUES ('%s', 'tenant.create', 'tenant', '%s');",
		sanitize(tenantID), sanitize(tenantID),
	)
	auditCmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-c", auditSQL,
	)
	if err := auditCmd.Run(); err != nil {
		slog.Warn("failed to write audit log for tenant creation", "error", err)
	}

	fmt.Printf("Tenant %q created (plan: %s, id: %s)\n", opts.Slug, opts.Plan, tenantID)
	return nil
}
