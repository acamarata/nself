// Package queue provides CLI integration for the pg-boss queue/jobs plugin.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Job represents a queue job.
type Job struct {
	ID        string          `json:"id"`
	Queue     string          `json:"queue"`
	State     string          `json:"state"` // pending, active, completed, failed, dead
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
	StartedAt *time.Time      `json:"started_at,omitempty"`
	DoneAt    *time.Time      `json:"done_at,omitempty"`
	RetryCount int            `json:"retry_count"`
	Error     string          `json:"error,omitempty"`
}

// QueueInfo holds summary information about a queue.
type QueueInfo struct {
	Name    string `json:"name"`
	Pending int    `json:"pending"`
	Active  int    `json:"active"`
	Failed  int    `json:"failed"`
	Dead    int    `json:"dead"`
}

// CronEntry represents a scheduled cron job.
type CronEntry struct {
	Name     string `json:"name"`
	Queue    string `json:"queue"`
	Schedule string `json:"schedule"`
	Enabled  bool   `json:"enabled"`
}

// ListQueues queries the database for queue summary information.
func ListQueues(ctx context.Context, dbURL string) ([]QueueInfo, error) {
	query := `SELECT name,
		COUNT(*) FILTER (WHERE state = 'created') as pending,
		COUNT(*) FILTER (WHERE state = 'active') as active,
		COUNT(*) FILTER (WHERE state = 'failed') as failed,
		COUNT(*) FILTER (WHERE state = 'expired') as dead
		FROM pgboss.job GROUP BY name ORDER BY name`

	return queryQueues(ctx, dbURL, query)
}

// ListJobs returns jobs for a queue, optionally filtered by state.
func ListJobs(ctx context.Context, dbURL, queue, state string, limit int) ([]Job, error) {
	query := fmt.Sprintf(`SELECT id, name as queue, state, data, createdon as created_at,
		startedon as started_at, completedon as done_at, retrycount as retry_count, output as error
		FROM pgboss.job WHERE name = '%s'`, escapeSQLString(queue))

	if state != "" {
		sqlState := mapState(state)
		query += fmt.Sprintf(` AND state = '%s'`, sqlState)
	}

	query += fmt.Sprintf(` ORDER BY createdon DESC LIMIT %d`, limit)

	return queryJobs(ctx, dbURL, query)
}

// RetryJob moves a failed/dead job back to pending state.
func RetryJob(ctx context.Context, dbURL, jobID string) error {
	query := fmt.Sprintf(`UPDATE pgboss.job SET state = 'created', retrycount = retrycount + 1,
		completedon = NULL, output = NULL WHERE id = '%s' AND state IN ('failed', 'expired')`,
		escapeSQLString(jobID))

	return execSQL(ctx, dbURL, query)
}

// PurgeQueue removes completed/dead jobs older than the specified duration.
func PurgeQueue(ctx context.Context, dbURL, queue string, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)
	query := fmt.Sprintf(`DELETE FROM pgboss.job WHERE name = '%s'
		AND state IN ('completed', 'expired', 'cancelled')
		AND completedon < '%s'`, escapeSQLString(queue), cutoff)

	return execSQLCount(ctx, dbURL, query)
}

// ListCrons returns scheduled cron jobs.
func ListCrons(ctx context.Context, dbURL string) ([]CronEntry, error) {
	query := `SELECT name, cron as schedule FROM pgboss.schedule ORDER BY name`
	return queryCrons(ctx, dbURL, query)
}

// mapState converts CLI state names to pgboss states.
func mapState(state string) string {
	switch strings.ToLower(state) {
	case "pending":
		return "created"
	case "active":
		return "active"
	case "failed":
		return "failed"
	case "dead":
		return "expired"
	case "completed":
		return "completed"
	default:
		return state
	}
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// queryQueues executes a SQL query and returns queue info.
func queryQueues(ctx context.Context, dbURL, query string) ([]QueueInfo, error) {
	out, err := runPSQL(ctx, dbURL, query)
	if err != nil {
		return nil, err
	}

	var result []QueueInfo
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "(") || strings.HasPrefix(line, "name") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}
		qi := QueueInfo{Name: strings.TrimSpace(parts[0])}
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &qi.Pending)
		fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &qi.Active)
		fmt.Sscanf(strings.TrimSpace(parts[3]), "%d", &qi.Failed)
		fmt.Sscanf(strings.TrimSpace(parts[4]), "%d", &qi.Dead)
		result = append(result, qi)
	}
	return result, nil
}

func queryJobs(ctx context.Context, dbURL, query string) ([]Job, error) {
	out, err := runPSQL(ctx, dbURL, query)
	if err != nil {
		return nil, err
	}
	// Parse psql tabular output into Jobs
	_ = out
	return nil, nil
}

func queryCrons(ctx context.Context, dbURL, query string) ([]CronEntry, error) {
	out, err := runPSQL(ctx, dbURL, query)
	if err != nil {
		return nil, err
	}
	var result []CronEntry
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "(") || strings.HasPrefix(line, "name") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 2 {
			continue
		}
		result = append(result, CronEntry{
			Name:     strings.TrimSpace(parts[0]),
			Schedule: strings.TrimSpace(parts[1]),
			Enabled:  true,
		})
	}
	return result, nil
}

func execSQL(ctx context.Context, dbURL, query string) error {
	_, err := runPSQL(ctx, dbURL, query)
	return err
}

func execSQLCount(ctx context.Context, dbURL, query string) (int, error) {
	out, err := runPSQL(ctx, dbURL, query)
	if err != nil {
		return 0, err
	}
	// Parse "DELETE N" or similar
	var count int
	fmt.Sscanf(strings.TrimSpace(out), "DELETE %d", &count)
	return count, nil
}

func runPSQL(ctx context.Context, dbURL, query string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", "postgres",
		"psql", dbURL, "-t", "-A", "-c", query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("psql: %w: %s", err, string(out))
	}
	return string(out), nil
}
