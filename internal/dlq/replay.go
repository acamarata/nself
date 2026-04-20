// Package dlq provides helpers for nself dlq replay.
package dlq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultMaxRows = 100

// ErrNoRows is returned when the DLQ contains no rows matching the filter.
var ErrNoRows = errors.New("no DLQ rows matched the filter criteria")

// ReplayOptions holds parsed flags for dlq replay.
type ReplayOptions struct {
	// Plugin name to replay (required).
	Plugin string
	// MaxRows caps the number of rows replayed (default 100).
	MaxRows int
	// Filter is a map of field=value pairs (e.g. "status=quarantined").
	Filter map[string]string
	// DryRun previews without executing.
	DryRun bool
	// BaseURL is the nSelf API base URL. Defaults to NSELF_API_URL env var or http://localhost:8080.
	BaseURL string
	// Stdout/Stderr for output.
	Stdout io.Writer
	Stderr io.Writer
	// HTTPClient allows injection in tests.
	HTTPClient *http.Client
}

// RowResult holds the per-row outcome of a replay attempt.
type RowResult struct {
	ID      string
	Success bool
	Err     error
}

// Replay queries a plugin's DLQ table and re-enqueues rows back to the work queue.
//
// Safe defaults: --max-rows 100 prevents floods.
// --dry-run previews without executing.
// A prominent warning is emitted to remind the operator to fix the upstream bug first.
func Replay(ctx context.Context, opts ReplayOptions) ([]RowResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	if opts.MaxRows <= 0 {
		opts.MaxRows = defaultMaxRows
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("NSELF_API_URL")
	}
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	// Emit upstream-bug warning prominently.
	fmt.Fprintf(stderr,
		"WARNING: replaying DLQ rows before fixing the upstream bug causes them to re-DLQ immediately.\n"+
			"         Confirm the root cause is resolved before continuing.\n\n")

	// Fetch DLQ rows.
	rows, err := fetchDLQRows(ctx, client, baseURL, opts.Plugin, opts.MaxRows, opts.Filter)
	if err != nil {
		return nil, fmt.Errorf("fetching DLQ rows for plugin %q: %w", opts.Plugin, err)
	}
	if len(rows) == 0 {
		fmt.Fprintf(stdout, "No DLQ rows found for plugin %q with current filters.\n", opts.Plugin)
		return nil, ErrNoRows
	}

	fmt.Fprintf(stdout, "Found %d DLQ row(s) for plugin %q (max-rows=%d)\n",
		len(rows), opts.Plugin, opts.MaxRows)

	if opts.DryRun {
		fmt.Fprintln(stdout, "--- DRY RUN: no rows will be re-enqueued ---")
		for _, r := range rows {
			fmt.Fprintf(stdout, "  would replay row id=%s\n", r["id"])
		}
		fmt.Fprintln(stdout, "No changes made (dry run).")
		return nil, nil
	}

	// Re-enqueue each row; report per-row success/failure.
	var results []RowResult
	var failCount int
	for _, row := range rows {
		id, _ := row["id"].(string)
		if id == "" {
			id = "(unknown)"
		}

		rErr := replayRow(ctx, client, baseURL, opts.Plugin, id)
		if rErr != nil {
			fmt.Fprintf(stderr, "  FAIL row=%s: %v\n", id, rErr)
			results = append(results, RowResult{ID: id, Success: false, Err: rErr})
			failCount++
		} else {
			fmt.Fprintf(stdout, "  OK   row=%s\n", id)
			results = append(results, RowResult{ID: id, Success: true})
		}
	}

	successCount := len(results) - failCount
	fmt.Fprintf(stdout, "\nReplayed %d/%d rows (%d failed)\n",
		successCount, len(results), failCount)

	if failCount > 0 {
		return results, fmt.Errorf("%d row(s) failed to replay", failCount)
	}
	return results, nil
}

// dlqRow is a map representing a single DLQ row.
type dlqRow = map[string]interface{}

// fetchDLQRows queries the plugin's DLQ endpoint and returns matching rows.
func fetchDLQRows(ctx context.Context, client *http.Client, baseURL, plugin string, maxRows int, filter map[string]string) ([]dlqRow, error) {
	url := fmt.Sprintf("%s/api/plugins/%s/dlq?max=%d", baseURL, plugin, maxRows)
	for k, v := range filter {
		url += fmt.Sprintf("&%s=%s", k, v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		// In test environments the endpoint may not exist — return empty rows.
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("plugin %q DLQ endpoint not found (is the plugin installed?)", plugin)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DLQ fetch returned HTTP %d", resp.StatusCode)
	}

	var rows []dlqRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decoding DLQ rows: %w", err)
	}
	return rows, nil
}

// replayRow re-enqueues a single DLQ row via the plugin's replay endpoint.
func replayRow(ctx context.Context, client *http.Client, baseURL, plugin, rowID string) error {
	url := fmt.Sprintf("%s/api/plugins/%s/dlq/%s/replay", baseURL, plugin, rowID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("replay returned HTTP %d", resp.StatusCode)
	}
	return nil
}
