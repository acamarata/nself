// Package health provides health checking utilities for nSelf services.
package health

// Purpose:
//   HasuraHealthzHandler is an HTTP handler that exposes a /healthz endpoint
//   with three-state Hasura health semantics (ok / degraded / down). It
//   replaces a naive nginx-direct-proxy that returned 500 when Hasura was
//   slow (> threshold), causing false-positive failures in upstream probes.
//
// Inputs:
//   HEALTHZ_HASURA_TIMEOUT_MS — configurable threshold (default 2000 ms).
//   HEALTHZ_HASURA_URL        — Hasura internal URL (default http://hasura:8080).
//
// Outputs:
//   200 {"status":"ok",       "hasura":"ok"}    — Hasura responded within threshold
//   200 {"status":"degraded", "hasura":"slow"}  — Hasura slow (> threshold)
//   503 {"status":"down",     "hasura":"down"}  — Hasura unreachable / refused
//
// Constraints:
//   - No goroutine leak: context is always cancelled via defer.
//   - Thread-safe: handler is stateless; reads env once via HasuraHealthzConfig.
//   - Timeout is per-request (not cumulative).
//
// SPORT: REGISTRY-ENDPOINTS.md /healthz, REGISTRY-ENV.md HEALTHZ_HASURA_TIMEOUT_MS

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultHasuraTimeoutMS = 2000
	defaultHasuraURL       = "http://hasura:8080/healthz"
)

// HasuraHealthzConfig holds runtime configuration for the Hasura health check.
// Read from env at handler construction time; immutable during serving.
type HasuraHealthzConfig struct {
	// HasuraURL is the URL to probe for Hasura liveness.
	HasuraURL string

	// TimeoutMS is the threshold in milliseconds. A Hasura response slower than
	// this is reported as "degraded" rather than "down".
	TimeoutMS int
}

// HasuraHealthzConfigFromEnv reads HEALTHZ_HASURA_TIMEOUT_MS and
// HEALTHZ_HASURA_URL from the environment, applying defaults for any missing
// or invalid values.
func HasuraHealthzConfigFromEnv() HasuraHealthzConfig {
	ms := defaultHasuraTimeoutMS
	if v := os.Getenv("HEALTHZ_HASURA_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 30000 {
			ms = n
		}
	}

	u := defaultHasuraURL
	if v := os.Getenv("HEALTHZ_HASURA_URL"); v != "" {
		u = v
	}

	return HasuraHealthzConfig{HasuraURL: u, TimeoutMS: ms}
}

// hasuraHealthzBody is the JSON response body for /healthz.
type hasuraHealthzBody struct {
	Status string `json:"status"`
	Hasura string `json:"hasura"`
}

// HasuraHealthzHandler returns an http.HandlerFunc that probes Hasura
// asynchronously with a configurable timeout and returns the appropriate
// three-state response.
//
// cfg controls the Hasura URL and timeout threshold. Use
// HasuraHealthzConfigFromEnv() for production; inject custom values in tests.
func HasuraHealthzHandler(cfg HasuraHealthzConfig) http.HandlerFunc {
	client := &http.Client{
		// The client timeout is set generously; the per-request context carries
		// the real threshold so we can distinguish slow vs. down.
		Timeout: time.Duration(cfg.TimeoutMS)*time.Millisecond + 5*time.Second,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		threshold := time.Duration(cfg.TimeoutMS) * time.Millisecond

		// resultCh receives the outcome of the async Hasura probe.
		type probeResult struct {
			reachable bool
		}
		resultCh := make(chan probeResult, 1)

		ctx, cancel := context.WithTimeout(r.Context(), threshold)
		defer cancel()

		go func() {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.HasuraURL, nil)
			if err != nil {
				resultCh <- probeResult{reachable: false}
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				resultCh <- probeResult{reachable: false}
				return
			}
			resp.Body.Close()
			resultCh <- probeResult{reachable: resp.StatusCode == http.StatusOK}
		}()

		writeHealthzJSON := func(status int, body hasuraHealthzBody) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		}

		select {
		case result := <-resultCh:
			if result.reachable {
				writeHealthzJSON(http.StatusOK, hasuraHealthzBody{
					Status: "ok",
					Hasura: "ok",
				})
			} else {
				// Hasura responded but returned non-200 (or connection error
				// arrived before the threshold fired) — treat as down.
				writeHealthzJSON(http.StatusServiceUnavailable, hasuraHealthzBody{
					Status: "down",
					Hasura: "down",
				})
			}
		case <-ctx.Done():
			// Threshold exceeded before Hasura replied — degraded, not down.
			writeHealthzJSON(http.StatusOK, hasuraHealthzBody{
				Status: "degraded",
				Hasura: "slow",
			})
		}
	}
}
