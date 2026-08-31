package tenant

// Purpose: usage query path for `nself tenant usage --tenant` and the shared runSQL exec helper used across the metering collectors.
// Inputs: a *config.Config, tenant ID, optional month filter, and output format.
// Outputs: a formatted usage report string, or an error.
// Constraints: split out of metering_collectors.go as a pure move (CLI-R12); no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// QueryUsage retrieves usage records for a tenant and optional month filter.
// Uses a direct database/sql connection with $N parameterized queries (Path A).
func QueryUsage(ctx context.Context, cfg *config.Config, tenantID, month, format string) (string, error) {
	if err := validateUUID(tenantID); err != nil {
		return "", err
	}
	if month != "" {
		if err := validateMonth(month); err != nil {
			return "", err
		}
	}

	db, err := openTenantDB(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("querying usage: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Build query with $N placeholders. Month filtering appends a date-range
	// predicate using a CAST to date so the comparison is type-safe.
	args := []interface{}{tenantID}
	query := "SELECT tenant_id::text, day::text, metric, value FROM nself_ops.usage_daily WHERE tenant_id = $1"
	if month != "" {
		// $2 = month prefix (YYYY-MM), used for both lower and upper bounds.
		query += " AND day >= ($2 || '-01')::date AND day < (($2 || '-01')::date + interval '1 month')"
		args = append(args, month)
	}
	query += " ORDER BY day, metric"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("querying usage: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type usageRow struct {
		TenantID string `json:"tenant_id"`
		Day      string `json:"day"`
		Metric   string `json:"metric"`
		Value    int64  `json:"value"`
	}
	var results []usageRow
	for rows.Next() {
		var r usageRow
		if err := rows.Scan(&r.TenantID, &r.Day, &r.Metric, &r.Value); err != nil {
			return "", fmt.Errorf("scanning usage row: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterating usage rows: %w", err)
	}

	if len(results) == 0 {
		return "", nil
	}

	if format == "json" {
		enc, err := json.Marshal(results)
		if err != nil {
			return "", fmt.Errorf("marshalling usage json: %w", err)
		}
		return string(enc), nil
	}

	// CSV format.
	var sb strings.Builder
	sb.WriteString("tenant_id,day,metric,value\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%d\n", r.TenantID, r.Day, r.Metric, r.Value))
	}
	return strings.TrimSpace(sb.String()), nil
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
