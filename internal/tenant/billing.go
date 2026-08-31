package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
// Uses a direct database/sql connection with $N parameterized queries (Path A).
func BillingReport(ctx context.Context, cfg *config.Config, opts BillingReportOptions) (string, error) {
	if opts.TenantSlug != "" {
		if err := validateSlug(opts.TenantSlug); err != nil {
			return "", err
		}
	}
	if opts.Month != "" {
		if err := validateMonth(opts.Month); err != nil {
			return "", err
		}
	}

	sqlDB, err := openTenantDB(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("generating billing report: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	// Build query with $N placeholders. Filters are applied only when provided.
	args := []interface{}{}
	query := `
		SELECT t.slug, t.plan, u.metric, sum(u.value) AS total
		FROM public.tenants t
		JOIN nself_ops.usage_daily u ON u.tenant_id = t.id
		WHERE 1=1`

	if opts.TenantSlug != "" {
		args = append(args, opts.TenantSlug)
		query += fmt.Sprintf(" AND t.slug = $%d", len(args))
	}
	if opts.Month != "" {
		args = append(args, opts.Month)
		// month is YYYY-MM; cast to date range server-side, never interpolated.
		query += fmt.Sprintf(
			" AND u.day >= ($%[1]d || '-01')::date AND u.day < (($%[1]d || '-01')::date + interval '1 month')",
			len(args),
		)
	}
	query += " GROUP BY t.slug, t.plan, u.metric ORDER BY t.slug, u.metric"

	dbRows, err := sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("generating billing report: %w", err)
	}
	defer func() { _ = dbRows.Close() }()

	type reportRow struct {
		Slug   string  `json:"slug"`
		Plan   string  `json:"plan"`
		Metric string  `json:"metric"`
		Total  float64 `json:"total"`
	}
	var rows []reportRow
	for dbRows.Next() {
		var r reportRow
		if err := dbRows.Scan(&r.Slug, &r.Plan, &r.Metric, &r.Total); err != nil {
			return "", fmt.Errorf("scanning billing row: %w", err)
		}
		rows = append(rows, r)
	}
	if err := dbRows.Err(); err != nil {
		return "", fmt.Errorf("iterating billing rows: %w", err)
	}

	if len(rows) == 0 {
		return "No usage data found.", nil
	}

	if opts.Format == "json" {
		enc, err := json.Marshal(rows)
		if err != nil {
			return "", fmt.Errorf("marshalling billing json: %w", err)
		}
		return string(enc), nil
	}

	// Format as table.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-20s %-12s %-25s %15s\n", "TENANT", "PLAN", "METRIC", "TOTAL"))
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("%-20s %-12s %-25s %15.2f\n", r.Slug, r.Plan, r.Metric, r.Total))
	}
	return sb.String(), nil
}

// RetryStripeEvent re-enqueues a failed Stripe outbox entry for retry.
// Uses a direct database/sql connection with $1 parameterized query (Path A).
func RetryStripeEvent(ctx context.Context, cfg *config.Config, eventID string) error {
	if err := validateEventID(eventID); err != nil {
		return err
	}

	sqlDB, err := openTenantDB(ctx, cfg)
	if err != nil {
		return fmt.Errorf("retrying stripe event: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	var returnedID string
	err = sqlDB.QueryRowContext(ctx,
		"UPDATE nself_ops.stripe_outbox SET processed_at = NULL, attempts = 0, last_error = NULL WHERE id = $1 RETURNING id",
		eventID,
	).Scan(&returnedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("stripe outbox entry %q not found", eventID)
		}
		return fmt.Errorf("retrying stripe event: %w", err)
	}

	slog.Info("stripe event re-enqueued for retry", "id", eventID)
	return nil
}
