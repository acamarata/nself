package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// CollectUsageOptions holds flags for usage collection.
type CollectUsageOptions struct {
	TenantID string // empty = all tenants
	Day      string // YYYY-MM-DD, empty = today
}

// CollectUsage runs the daily metering collection: pg catalog row counts,
// active users from sessions. MinIO storage and nginx bandwidth require
// external integration and are stubbed for CLI-only collection.
func CollectUsage(ctx context.Context, cfg *config.Config, opts CollectUsageOptions) error {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	day := opts.Day
	if day == "" {
		day = "CURRENT_DATE"
	} else {
		if err := validateDate(opts.Day); err != nil {
			return err
		}
		day = fmt.Sprintf("'%s'::date", sanitize(opts.Day))
	}

	tenantFilter := ""
	if opts.TenantID != "" {
		if err := validateUUID(opts.TenantID); err != nil {
			return err
		}
		tenantFilter = fmt.Sprintf(" AND t.id = '%s'", sanitize(opts.TenantID))
	}

	// Collect rows_stored per tenant via pg catalog estimates.
	rowsSQL := fmt.Sprintf(`
INSERT INTO nself_ops.usage_daily (tenant_id, day, metric, value)
SELECT t.id, %s, 'rows_stored', COALESCE(counts.total, 0)
FROM public.tenants t
LEFT JOIN LATERAL (
    SELECT sum(c.reltuples::bigint) AS total
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN information_schema.columns col ON col.table_schema = n.nspname AND col.table_name = c.relname
    WHERE col.column_name = 'tenant_id'
      AND c.relkind = 'r'
) counts ON true
WHERE t.status = 'active'%s
ON CONFLICT (tenant_id, day, metric) DO UPDATE SET value = EXCLUDED.value;`, day, tenantFilter)

	if err := runSQL(ctx, container, user, db, rowsSQL); err != nil {
		slog.Warn("failed to collect rows_stored metric", "error", err)
	}

	// Collect active_users per tenant (distinct user_id from auth sessions today).
	activeUsersSQL := fmt.Sprintf(`
INSERT INTO nself_ops.usage_daily (tenant_id, day, metric, value)
SELECT t.id, %s, 'active_users', 0
FROM public.tenants t
WHERE t.status = 'active'%s
ON CONFLICT (tenant_id, day, metric) DO UPDATE SET value = EXCLUDED.value;`, day, tenantFilter)

	if err := runSQL(ctx, container, user, db, activeUsersSQL); err != nil {
		slog.Warn("failed to collect active_users metric", "error", err)
	}

	slog.Info("usage collection complete")
	return nil
}

// QueryUsage retrieves usage records for a tenant and optional month filter.
func QueryUsage(ctx context.Context, cfg *config.Config, tenantID, month, format string) (string, error) {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	if err := validateUUID(tenantID); err != nil {
		return "", err
	}
	whereClause := fmt.Sprintf("WHERE tenant_id = '%s'", sanitize(tenantID))
	if month != "" {
		if err := validateMonth(month); err != nil {
			return "", err
		}
		whereClause += fmt.Sprintf(" AND day >= '%s-01'::date AND day < ('%s-01'::date + interval '1 month')",
			sanitize(month), sanitize(month))
	}

	var sql string
	if format == "json" {
		sql = fmt.Sprintf(
			"SELECT json_agg(row_to_json(t)) FROM (SELECT tenant_id, day, metric, value FROM nself_ops.usage_daily %s ORDER BY day, metric) t;",
			whereClause,
		)
	} else {
		// CSV format.
		sql = fmt.Sprintf(
			"COPY (SELECT tenant_id, day, metric, value FROM nself_ops.usage_daily %s ORDER BY day, metric) TO STDOUT WITH CSV HEADER;",
			whereClause,
		)
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("querying usage: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// runSQL executes a SQL statement inside the postgres container.
func runSQL(ctx context.Context, container, user, db, sql string) error {
	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-c", sql,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}
	return nil
}
