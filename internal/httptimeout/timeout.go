// Package httptimeout provides timeout-bounded *http.Client values per scope.
//
// All HTTP traffic from CLI internals MUST use a client from this package.
// Direct use of http.DefaultClient, http.Get, http.Post, or &http.Client{}
// (without an explicit Timeout) is forbidden by the forbidigo linter and the
// http-timeout-check CI workflow.
//
// Per-scope timeouts can be overridden via environment variables:
//
//	NSELF_HTTP_TIMEOUT_DEFAULT    (default 30s)
//	NSELF_HTTP_TIMEOUT_INSTALLER  (default 300s)
//	NSELF_HTTP_TIMEOUT_HEALTH     (default 5s)
//	NSELF_HTTP_TIMEOUT_LICENSE    (default 10s)
//	NSELF_HTTP_TIMEOUT_PLUGIN     (default 15s)
//	NSELF_HTTP_TIMEOUT_BACKUP     (default 600s)
//
// Values are integer seconds. Invalid or non-positive values fall back to the
// hard-coded default. Env vars are read at package init time.
package httptimeout

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

// envDuration reads an integer-second env var and returns it as a
// time.Duration. Returns fallback when the env var is unset, non-numeric,
// or non-positive.
func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}

// Per-scope clients. Each client has Timeout set; callers SHOULD also pass
// a context.Context with a deadline for fine-grained cancellation.
var (
	// Default is the general-purpose client (30s).
	Default = &http.Client{Timeout: envDuration("NSELF_HTTP_TIMEOUT_DEFAULT", 30*time.Second)}

	// Installer is for long downloads such as Docker image pulls or binary fetches (300s).
	Installer = &http.Client{Timeout: envDuration("NSELF_HTTP_TIMEOUT_INSTALLER", 300*time.Second)}

	// Health is for liveness/readiness probes — fail fast (5s).
	Health = &http.Client{Timeout: envDuration("NSELF_HTTP_TIMEOUT_HEALTH", 5*time.Second)}

	// License is for ping_api license validation (10s).
	License = &http.Client{Timeout: envDuration("NSELF_HTTP_TIMEOUT_LICENSE", 10*time.Second)}

	// Plugin is for plugin registry queries (15s).
	Plugin = &http.Client{Timeout: envDuration("NSELF_HTTP_TIMEOUT_PLUGIN", 15*time.Second)}

	// Backup is for long-running backup uploads/downloads (600s).
	Backup = &http.Client{Timeout: envDuration("NSELF_HTTP_TIMEOUT_BACKUP", 600*time.Second)}
)

// WithTimeout returns a fresh *http.Client with the given timeout. Use sparingly;
// prefer one of the scoped clients above so timeouts stay tunable via env.
func WithTimeout(d time.Duration) *http.Client {
	return &http.Client{Timeout: d}
}
