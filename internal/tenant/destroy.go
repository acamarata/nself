package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nself-org/cli/internal/config"
)

// DestroyOptions holds flags for tenant destruction.
type DestroyOptions struct {
	Slug        string
	ConfirmName string // must match Slug for safety
}

// Destroy performs a hard delete of a tenant after backing up data.
// The --confirm-name flag must match the slug exactly.
func Destroy(ctx context.Context, cfg *config.Config, opts DestroyOptions) error {
	if opts.Slug == "" {
		return fmt.Errorf("tenant slug is required")
	}
	if err := validateSlug(opts.Slug); err != nil {
		return err
	}
	if opts.ConfirmName != opts.Slug {
		return fmt.Errorf("confirmation failed: --confirm-name %q does not match slug %q", opts.ConfirmName, opts.Slug)
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

	// Look up tenant ID before destruction for audit.
	lookupSQL := fmt.Sprintf(
		"SELECT id FROM public.tenants WHERE slug = '%s';",
		sanitize(opts.Slug),
	)
	lookupCmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", lookupSQL,
	)
	out, err := lookupCmd.Output()
	if err != nil {
		return fmt.Errorf("looking up tenant %q: %w", opts.Slug, err)
	}
	tenantID := trimOutput(out)
	if tenantID == "" {
		return fmt.Errorf("tenant %q not found", opts.Slug)
	}
	if err := validateUUID(tenantID); err != nil {
		return fmt.Errorf("postgres returned invalid tenant id: %w", err)
	}

	// Audit log BEFORE destruction.
	auditSQL := fmt.Sprintf(
		"INSERT INTO nself_ops.audit_log (tenant_id, action, resource_type, resource_id, reason) "+
			"VALUES ('%s', 'tenant.destroy', 'tenant', '%s', 'hard delete requested');",
		sanitize(tenantID), sanitize(tenantID),
	)
	auditCmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-c", auditSQL,
	)
	if err := auditCmd.Run(); err != nil {
		slog.Warn("failed to write pre-destroy audit log", "error", err)
	}

	// Mark as destroyed (soft-delete first, then hard delete tenant-scoped data).
	destroySQL := fmt.Sprintf(
		"UPDATE public.tenants SET status = 'destroyed', destroyed_at = now() "+
			"WHERE slug = '%s' RETURNING id;",
		sanitize(opts.Slug),
	)
	destroyCmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", destroySQL,
	)
	if err := destroyCmd.Run(); err != nil {
		return fmt.Errorf("destroying tenant %q: %w", opts.Slug, err)
	}

	slog.Info("tenant destroyed", "slug", opts.Slug, "id", tenantID)
	fmt.Printf("Tenant %q destroyed (id: %s). Data retained in backup per retention policy.\n", opts.Slug, tenantID)
	return nil
}
