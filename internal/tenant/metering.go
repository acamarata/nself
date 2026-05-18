package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// CollectUsageOptions holds flags for usage collection.
type CollectUsageOptions struct {
	TenantID string // empty = all tenants
	Day      string // YYYY-MM-DD, empty = today
}

// CollectUsage runs the daily metering collection: pg catalog row counts,
// active users from auth sessions, MinIO bucket storage, and nginx bandwidth
// via Prometheus query API (when monitoring is enabled).
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
	// Scoped to n.nspname = 'public' so the catalog scan only touches application
	// tables (np_* prefix) and avoids traversing pg_catalog / information_schema
	// entries that can never carry tenant data. Values are unchanged — all
	// nSelf application tables with a tenant_id column live in the public schema.
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
      AND n.nspname = 'public'
) counts ON true
WHERE t.status = 'active'%s
ON CONFLICT (tenant_id, day, metric) DO UPDATE SET value = EXCLUDED.value;`, day, tenantFilter)

	if err := runSQL(ctx, container, user, db, rowsSQL); err != nil {
		slog.Warn("failed to collect rows_stored metric", "error", err)
	}

	// Collect active_users per tenant via auth sessions.
	activeUsersSQL := fmt.Sprintf(`
INSERT INTO nself_ops.usage_daily (tenant_id, day, metric, value)
SELECT t.id, %s, 'active_users',
    COALESCE((
        SELECT count(DISTINCT user_id)
        FROM auth.sessions s
        WHERE s.created_at >= CURRENT_DATE
          AND s.created_at < CURRENT_DATE + interval '1 day'
          AND s.user_id IN (
              SELECT id FROM auth.users u WHERE u.tenant_id = t.id
          )
    ), 0)
FROM public.tenants t
WHERE t.status = 'active'%s
ON CONFLICT (tenant_id, day, metric) DO UPDATE SET value = EXCLUDED.value;`, day, tenantFilter)

	if err := runSQL(ctx, container, user, db, activeUsersSQL); err != nil {
		slog.Warn("failed to collect active_users metric", "error", err)
	}

	// Collect MinIO storage bytes per tenant bucket (when MinIO is enabled).
	if cfg.Minio.Enabled {
		if err := collectMinIOStorage(ctx, cfg, container, user, db, day, tenantFilter); err != nil {
			slog.Warn("failed to collect storage_bytes metric", "error", err)
		}
	}

	// Collect nginx bandwidth bytes via Prometheus (when monitoring is enabled).
	if cfg.Monitoring.PrometheusEnabled && cfg.Monitoring.PrometheusPort > 0 {
		if err := collectNginxBandwidth(ctx, cfg, container, user, db, day, tenantFilter); err != nil {
			slog.Warn("failed to collect bandwidth_bytes metric", "error", err)
		}
	}

	slog.Info("usage collection complete")
	return nil
}

// collectMinIOStorage queries the MinIO admin API for per-bucket storage usage
// and upserts storage_bytes into nself_ops.usage_daily for each tenant.
//
// MinIO bucket naming convention: one bucket per tenant, named by tenant slug.
// The admin API endpoint GET /minio/admin/v3/info returns aggregate info;
// per-bucket sizes are obtained via GET /minio/admin/v3/bucket/{bucket}?usage.
// Both endpoints require admin credentials (MINIO_ROOT_USER/PASSWORD).
func collectMinIOStorage(ctx context.Context, cfg *config.Config, pgContainer, pgUser, pgDB, day, tenantFilter string) error {
	minioPort := cfg.Minio.Port
	if minioPort <= 0 {
		minioPort = 9000
	}
	rootUser := cfg.Minio.RootUser
	if rootUser == "" {
		rootUser = "minioadmin"
	}
	rootPass := cfg.Minio.RootPassword

	// Fetch the list of active tenant slugs from Postgres so we know which
	// buckets to query. We look up each bucket by the tenant's slug/id.
	slugSQL := fmt.Sprintf(
		"SELECT id, slug FROM public.tenants WHERE status = 'active'%s;",
		tenantFilter,
	)
	cmd := exec.CommandContext(ctx, "docker", "exec", pgContainer,
		"psql", "-U", pgUser, "-d", pgDB, "-tAc", slugSQL,
	)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("listing tenants for MinIO metering: %w", err)
	}

	type tenantRow struct {
		id   string
		slug string
	}
	var tenants []tenantRow
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 && parts[0] != "" {
			tenants = append(tenants, tenantRow{id: parts[0], slug: parts[1]})
		}
	}
	if len(tenants) == 0 {
		return nil
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}

	for _, t := range tenants {
		if err := validateUUID(t.id); err != nil {
			slog.Warn("skipping invalid tenant id in MinIO metering", "id", t.id)
			continue
		}
		// Bucket names follow the tenant slug; fall back to id prefix for safety.
		bucketName := t.slug
		if bucketName == "" {
			bucketName = t.id
		}

		// MinIO admin API: GET /minio/admin/v3/bucket?bucket=<name>&usage=true
		// Returns JSON with "size" field in bytes for the bucket.
		// Credentials are set via SetBasicAuth rather than embedded in the URL to
		// prevent the credential-bearing URL string from appearing in error logs
		// (Go's http error messages include the URL on connection failures).
		minioURL := fmt.Sprintf("http://127.0.0.1:%d/minio/admin/v3/bucket?bucket=%s&usage=true",
			minioPort, bucketName,
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, minioURL, nil)
		if err != nil {
			slog.Warn("failed to build MinIO request", "bucket", bucketName, "error", err)
			continue
		}
		req.SetBasicAuth(rootUser, rootPass)

		resp, err := httpClient.Do(req)
		if err != nil {
			slog.Warn("MinIO admin API unavailable", "bucket", bucketName, "error", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			// Bucket does not exist yet; record 0 bytes.
			if err := upsertUsage(ctx, pgContainer, pgUser, pgDB, t.id, day, MetricStorageBytes, 0); err != nil {
				slog.Warn("failed to record zero storage for missing bucket", "bucket", bucketName, "error", err)
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			slog.Warn("MinIO admin API returned error", "bucket", bucketName, "status", resp.StatusCode)
			continue
		}

		// Parse {"size": <bytes>} from the response.
		var info struct {
			Size int64 `json:"size"`
		}
		if err := json.Unmarshal(body, &info); err != nil {
			slog.Warn("failed to parse MinIO bucket info", "bucket", bucketName, "error", err)
			continue
		}

		if err := upsertUsage(ctx, pgContainer, pgUser, pgDB, t.id, day, MetricStorageBytes, info.Size); err != nil {
			slog.Warn("failed to upsert storage metric", "bucket", bucketName, "error", err)
		}
	}

	return nil
}

// collectNginxBandwidth queries Prometheus for total nginx egress bytes by
// tenant (via virtual-host labels) and upserts bandwidth_bytes into
// nself_ops.usage_daily.
//
// The nself monitoring stack includes nginx-vts or stub_status exporters.
// We query: sum by (tenant_id) (nginx_vts_server_bytes_total{direction="out"})
// If the metric is absent (exporter not installed), the function logs a warning
// and returns without error so the caller can continue collecting other metrics.
func collectNginxBandwidth(ctx context.Context, cfg *config.Config, pgContainer, pgUser, pgDB, day, tenantFilter string) error {
	promPort := cfg.Monitoring.PrometheusPort
	if promPort <= 0 {
		return nil
	}

	// PromQL: sum nginx outbound bytes grouped by tenant_id label.
	// The nginx-vts module exports per-vhost stats with a tenant_id label
	// set by nself's nginx template.
	query := `sum by (tenant_id) (nginx_vts_server_bytes_total{direction="out"})`
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/query?query=%s",
		promPort, strings.NewReplacer("{", "%7B", "}", "%7D", "\"", "%22").Replace(query),
	)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building Prometheus request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Warn("Prometheus unavailable for bandwidth metering", "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Prometheus returned non-200 for bandwidth query", "status", resp.StatusCode)
		return nil
	}

	// Parse Prometheus instant-query response.
	var promResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  [2]interface{}    `json:"value"` // [timestamp, "value_string"]
			} `json:"result"`
		} `json:"data"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &promResp); err != nil {
		slog.Warn("failed to parse Prometheus response", "error", err)
		return nil
	}
	if promResp.Status != "success" || promResp.Data.ResultType == "" {
		// Metric may not exist (exporter not installed); not an error.
		slog.Info("nginx bandwidth metric absent from Prometheus; skipping bandwidth collection")
		return nil
	}

	for _, r := range promResp.Data.Result {
		tenantID, ok := r.Metric["tenant_id"]
		if !ok || tenantID == "" {
			continue
		}
		if err := validateUUID(tenantID); err != nil {
			slog.Warn("skipping invalid tenant_id in Prometheus metric", "tenant_id", tenantID)
			continue
		}

		// Value is returned as a string in the JSON (Prometheus convention).
		valStr, _ := r.Value[1].(string)
		var bytes int64
		fmt.Sscanf(valStr, "%d", &bytes) //nolint:errcheck // zero on parse failure is acceptable

		if err := upsertUsage(ctx, pgContainer, pgUser, pgDB, tenantID, day, MetricBandwidthBytes, bytes); err != nil {
			slog.Warn("failed to upsert bandwidth metric", "tenant_id", tenantID, "error", err)
		}
	}

	return nil
}

// upsertUsage inserts or updates a single metric row in nself_ops.usage_daily.
// It returns any SQL error to the caller rather than silently dropping it.
func upsertUsage(ctx context.Context, pgContainer, pgUser, pgDB, tenantID, day, metric string, value int64) error {
	sql := fmt.Sprintf(`
INSERT INTO nself_ops.usage_daily (tenant_id, day, metric, value)
VALUES ('%s', %s, '%s', %d)
ON CONFLICT (tenant_id, day, metric) DO UPDATE SET value = EXCLUDED.value;`,
		sanitize(tenantID), day, sanitize(metric), value,
	)
	if err := runSQL(ctx, pgContainer, pgUser, pgDB, sql); err != nil {
		return fmt.Errorf("upsert usage metric (tenant=%s metric=%s): %w", tenantID, metric, err)
	}
	return nil
}

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
	defer db.Close()

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
	defer rows.Close()

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
