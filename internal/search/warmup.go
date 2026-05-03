// Package search provides MeiliSearch integration helpers for the nSelf CLI.
package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// EnvWarmupQueries is the environment variable that holds a comma-separated
	// list of representative search query strings used to warm up the MeiliSearch
	// index cache on startup. Warm-up is skipped when the variable is empty or
	// unset.
	EnvWarmupQueries = "MEILISEARCH_WARMUP_QUERIES"

	defaultWarmupTimeout = 10 * time.Second
	warmupRequestTimeout = 3 * time.Second
)

// WarmupConfig holds the parameters needed to run a warm-up sweep.
type WarmupConfig struct {
	// Host is the MeiliSearch hostname (default "localhost").
	Host string
	// Port is the MeiliSearch HTTP port (default 7700).
	Port int
	// MasterKey is the MeiliSearch master key, used as the Authorization
	// header. May be empty for instances configured without auth.
	MasterKey string
	// IndexPrefix is prepended to each index name (e.g. "np_").
	IndexPrefix string
	// Queries is the list of warm-up query strings. When empty, Warmup is a
	// no-op and returns immediately without any network calls.
	Queries []string
}

// QueriesFromEnv reads MEILISEARCH_WARMUP_QUERIES and splits it on commas.
// It trims whitespace from each element and discards empties. Returns nil when
// the variable is unset or contains only whitespace.
func QueriesFromEnv() []string {
	raw := os.Getenv(EnvWarmupQueries)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if q := strings.TrimSpace(p); q != "" {
			out = append(out, q)
		}
	}
	return out
}

// WarmupResult summarises a completed warm-up run.
type WarmupResult struct {
	QueriesAttempted int
	QueriesSucceeded int
	Errors           []string
}

// Warmup issues one search request per query string against the first
// discoverable index on the MeiliSearch instance.  It uses the multi-index
// search endpoint (POST /multi-search) so no specific index name is required
// in advance.
//
// When cfg.Queries is empty the function returns immediately with a zero
// WarmupResult and a nil error.  The caller should treat a warm-up as
// best-effort — errors are collected and returned but never fatal.
func Warmup(ctx context.Context, cfg WarmupConfig) (WarmupResult, error) {
	if len(cfg.Queries) == 0 {
		return WarmupResult{}, nil
	}

	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 7700
	}

	baseURL := fmt.Sprintf("http://%s:%d", host, port)
	client := &http.Client{Timeout: warmupRequestTimeout}

	warmupCtx, cancel := context.WithTimeout(ctx, defaultWarmupTimeout)
	defer cancel()

	result := WarmupResult{QueriesAttempted: len(cfg.Queries)}

	for _, q := range cfg.Queries {
		if warmupCtx.Err() != nil {
			result.Errors = append(result.Errors, "context deadline exceeded, remaining queries skipped")
			break
		}
		if err := issueWarmupQuery(warmupCtx, client, baseURL, cfg.MasterKey, cfg.IndexPrefix, q); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("query %q: %v", q, err))
		} else {
			result.QueriesSucceeded++
		}
	}

	return result, nil
}

// issueWarmupQuery posts a single search request to the MeiliSearch multi-search
// endpoint.  Any 2xx or 404 (index not yet created) response is treated as
// success because the goal is cache population, not correctness validation.
func issueWarmupQuery(ctx context.Context, client *http.Client, baseURL, masterKey, indexPrefix, query string) error {
	// Build a minimal multi-search payload.  indexPrefix may be empty.
	indexUID := indexPrefix + "*" // wildcard accepted by Meili >= v1.5
	body := fmt.Sprintf(`{"queries":[{"indexUid":%q,"q":%q,"limit":1}]}`,
		indexUID, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/multi-search", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if masterKey != "" {
		req.Header.Set("Authorization", "Bearer "+masterKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); resp.Body.Close() }()

	// 2xx = success.  404 = index absent (brand-new instance) — treat as OK
	// because the warm-up still exercised the Meili request-parsing path.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return fmt.Errorf("unexpected status %d", resp.StatusCode)
}
