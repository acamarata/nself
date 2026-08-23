package doctor

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Purpose: Postgres and Hasura --deep checks — pg_isready, longest running
// query, dead tuple ratio, last vacuum, plus Hasura's /healthz and metadata
// consistency.
// Inputs: a context and verbose flag.
// Outputs: []CheckResult per category.
// Constraints: split out of deep.go (CLI-R12) as a pure move; no behavior
// changed.

// PostgresChecks verifies pg_isready, replication lag, longest query, dead tuples, vacuum.
func PostgresChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// pg_isready
	cmd := exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "pg_isready", "-U", "postgres")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "postgres", Name: "pg_isready", Status: "fail", Message: "not ready"})
		return results
	}
	results = append(results, CheckResult{Section: "postgres", Name: "pg_isready", Status: "pass", Message: "accepting connections"})

	// Longest running query <60s
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "psql", "-U", "postgres", "-t", "-c",
		"SELECT COALESCE(EXTRACT(EPOCH FROM MAX(now() - query_start))::int, 0) FROM pg_stat_activity WHERE state='active' AND pid != pg_backend_pid();")
	out, err := cmd.Output()
	if err == nil {
		secs, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		if secs > 60 {
			results = append(results, CheckResult{Section: "postgres", Name: "Longest query", Status: "warn",
				Message: fmt.Sprintf("%ds (>60s)", secs)})
		} else {
			results = append(results, CheckResult{Section: "postgres", Name: "Longest query", Status: "pass",
				Message: fmt.Sprintf("%ds", secs)})
		}
	}

	// Dead tuple % <10%
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "psql", "-U", "postgres", "-t", "-c",
		"SELECT COALESCE(MAX(n_dead_tup::float / NULLIF(n_live_tup + n_dead_tup, 0) * 100)::int, 0) FROM pg_stat_user_tables;")
	out, err = cmd.Output()
	if err == nil {
		pct, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		if pct > 10 {
			results = append(results, CheckResult{Section: "postgres", Name: "Dead tuples", Status: "warn",
				Message: fmt.Sprintf("%d%% dead tuples (>10%%)", pct),
				FixCmd:  "docker exec nself_postgres psql -U postgres -c 'VACUUM ANALYZE;'"})
		} else {
			results = append(results, CheckResult{Section: "postgres", Name: "Dead tuples", Status: "pass",
				Message: fmt.Sprintf("%d%% dead tuples", pct)})
		}
	}

	// Last vacuum <24h
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "psql", "-U", "postgres", "-t", "-c",
		"SELECT COALESCE(EXTRACT(EPOCH FROM MIN(now() - last_autovacuum))::int, 0) FROM pg_stat_user_tables WHERE last_autovacuum IS NOT NULL;")
	out, err = cmd.Output()
	if err == nil {
		secs, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		hours := secs / 3600
		if hours > 24 {
			results = append(results, CheckResult{Section: "postgres", Name: "Last vacuum", Status: "warn",
				Message: fmt.Sprintf("%dh ago (>24h)", hours)})
		} else {
			results = append(results, CheckResult{Section: "postgres", Name: "Last vacuum", Status: "pass",
				Message: fmt.Sprintf("%dh ago", hours)})
		}
	}

	return results
}

// HasuraChecks verifies /healthz 200 and metadata consistency.
func HasuraChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// /healthz
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/healthz")
	if err != nil || resp.StatusCode != 200 {
		results = append(results, CheckResult{Section: "hasura", Name: "Hasura healthz", Status: "fail", Message: "unhealthy"})
	} else {
		results = append(results, CheckResult{Section: "hasura", Name: "Hasura healthz", Status: "pass", Message: "200 OK"})
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Metadata consistency — check via metadata API
	cmd := exec.CommandContext(ctx, "curl", "-sf", "-H", "Content-Type: application/json",
		"-d", `{"type":"get_inconsistent_metadata","args":{}}`,
		"http://127.0.0.1:8080/v1/metadata")
	out, err := cmd.Output()
	if err != nil {
		results = append(results, CheckResult{Section: "hasura", Name: "Metadata consistency", Status: "warn",
			Message: "cannot check (admin secret required)"})
	} else if strings.Contains(string(out), `"is_consistent":true`) {
		results = append(results, CheckResult{Section: "hasura", Name: "Metadata consistency", Status: "pass", Message: "consistent"})
	} else {
		results = append(results, CheckResult{Section: "hasura", Name: "Metadata consistency", Status: "fail",
			Message: "inconsistent metadata objects found"})
	}

	return results
}
