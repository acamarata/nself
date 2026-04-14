package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// BillingReportOptions holds flags for billing report generation.
type BillingReportOptions struct {
	TenantSlug string
	Month      string // YYYY-MM
	Format     string // table, json, csv
}

// BillingReport generates a usage and billing summary for a tenant or all tenants.
func BillingReport(ctx context.Context, cfg *config.Config, opts BillingReportOptions) (string, error) {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	where := "WHERE 1=1"
	if opts.TenantSlug != "" {
		where += fmt.Sprintf(" AND t.slug = '%s'", sanitize(opts.TenantSlug))
	}
	if opts.Month != "" {
		where += fmt.Sprintf(" AND u.day >= '%s-01'::date AND u.day < ('%s-01'::date + interval '1 month')",
			sanitize(opts.Month), sanitize(opts.Month))
	}

	sql := fmt.Sprintf(`SELECT json_agg(row_to_json(r)) FROM (
		SELECT t.slug, t.plan, u.metric, sum(u.value) as total
		FROM public.tenants t
		JOIN nself_ops.usage_daily u ON u.tenant_id = t.id
		%s
		GROUP BY t.slug, t.plan, u.metric
		ORDER BY t.slug, u.metric
	) r;`, where)

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("generating billing report: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == "null" {
		return "No usage data found.", nil
	}

	if opts.Format == "json" {
		return raw, nil
	}

	// Format as table.
	var rows []struct {
		Slug   string  `json:"slug"`
		Plan   string  `json:"plan"`
		Metric string  `json:"metric"`
		Total  float64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return "", fmt.Errorf("parsing report: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-20s %-12s %-25s %15s\n", "TENANT", "PLAN", "METRIC", "TOTAL"))
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%-20s %-12s %-25s %15.2f\n", r.Slug, r.Plan, r.Metric, r.Total))
	}
	return sb.String(), nil
}

// RetryStripeEvent re-enqueues a failed Stripe outbox entry for retry.
func RetryStripeEvent(ctx context.Context, cfg *config.Config, eventID string) error {
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
		"UPDATE nself_ops.stripe_outbox SET processed_at = NULL, attempts = 0, last_error = NULL "+
			"WHERE id = '%s' RETURNING id;",
		sanitize(eventID),
	)

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("retrying stripe event: %w", err)
	}

	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("stripe outbox entry %q not found", eventID)
	}

	slog.Info("stripe event re-enqueued", "id", eventID)
	fmt.Printf("Stripe event %s re-enqueued for retry.\n", eventID)
	return nil
}
