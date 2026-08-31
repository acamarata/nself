package tenant

// Purpose: per-tenant usage collectors (MinIO storage, nginx bandwidth) and the shared SQL helpers (upsert, query, exec) backing `nself tenant usage`.
// Inputs: a *config.Config, target Postgres container/user/db, a day/month string, and an optional tenant filter.
// Outputs: rows upserted into nself_ops.usage_daily, or usage query results.
// Constraints: split out of metering.go as a pure move (CLI-R12); no behavior change.

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
		_ = resp.Body.Close()

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
	defer func() { _ = resp.Body.Close() }()

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
