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
	if err := validateSlug(opts.Slug); err != nil {
		return err
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

	// Insert tenant row using parameterized query via psql \set + placeholders
	// to avoid SQL injection. psql does not support $1/$2 bind parameters
	// directly via -c, so we use a JSON-escaped literal approach: values are
	// passed as environment-variable placeholders via psql --variable and
	// referenced with :'varname' (server-side quoting).
	//
	// Equivalent parameterized intent: INSERT INTO public.tenants
	// (slug, plan, status) VALUES ($1, $2, 'active') RETURNING id
	insertSQL := "INSERT INTO public.tenants (slug, plan, status) VALUES (:'slug', :'plan', 'active') RETURNING id;"

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", insertSQL,
		"--variable=slug="+opts.Slug,
		"--variable=plan="+string(opts.Plan),
	)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("creating tenant %q: %w", opts.Slug, err)
	}

	tenantID := trimOutput(out)
	if err := validateUUID(tenantID); err != nil {
		return fmt.Errorf("postgres returned invalid tenant id: %w", err)
	}
	slog.Info("tenant created", "slug", opts.Slug, "plan", opts.Plan, "id", tenantID)

	// Write audit log entry using parameterized :'varname' quoting.
	// Equivalent intent: INSERT INTO nself_ops.audit_log
	// (tenant_id, action, resource_type, resource_id) VALUES ($1, 'tenant.create', 'tenant', $2)
	auditSQL := "INSERT INTO nself_ops.audit_log (tenant_id, action, resource_type, resource_id) VALUES (:'tid', 'tenant.create', 'tenant', :'tid');"
	auditCmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-c", auditSQL,
		"--variable=tid="+tenantID,
	)
	if err := auditCmd.Run(); err != nil {
		slog.Warn("failed to write audit log for tenant creation", "error", err)
	}

	fmt.Printf("Tenant %q created (plan: %s, id: %s)\n", opts.Slug, opts.Plan, tenantID)
	return nil
}
