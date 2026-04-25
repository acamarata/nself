// Package sdk provides shared infrastructure for nSelf plugin services written in Go.
//
// Plugins import sub-packages to avoid rebuilding boilerplate:
//
//	sdk/plugin  - base plugin types, lifecycle, config
//	sdk/logger  - slog wrappers with consistent JSON output
//	sdk/config  - env + file config loader
//	sdk/db      - pgx pool helpers, health checks, migrations
//	sdk/httpx   - HTTP client with retries, timeouts, tracing
//	sdk/metrics - Prometheus /metrics endpoint with per-plugin counters
//	sdk/server  - chi router setup with middleware defaults
//	sdk/license - license grace period + offline validation helpers
//
// Every nSelf Go plugin (ai, claw, mux, voice, browser, notify, cron, chat,
// livekit, recording, bots, etc.) should depend on this module instead of
// reimplementing the same glue.
package sdk

// Version is the SDK release. Consumers can gate on this via compatibility.go.
const Version = "0.1.0"
