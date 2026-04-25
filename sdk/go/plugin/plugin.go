// Package plugin defines the core Plugin interface and lifecycle types used
// by every nSelf plugin service.
package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Info describes a plugin's identity and capabilities. Matches the fields in
// plugin.json that the CLI loader cares about.
type Info struct {
	Name        string            // e.g. "ai", "mux", "claw"
	Version     string            // SemVer, e.g. "1.2.0"
	Description string            // One-line human summary
	Tier        string            // "free" | "pro"
	Bundle      string            // e.g. "nClaw", "nChat"; empty for free plugins
	Category    string            // See F04-PLUGIN-INVENTORY-PRO.md categories
	MinCLI      string            // Minimum nSelf CLI version required
	MaxCLI      string            // "" means no upper bound
	MinSDK      string            // Minimum plugin-sdk-go version (e.g. "0.1.0")
	Metadata    map[string]string // Free-form extension slot
}

// Plugin is the common surface every nSelf plugin service satisfies.
// Concrete plugins return an Info, report liveness/readiness, and expose hooks
// for startup + shutdown.
type Plugin interface {
	// Info returns immutable identity metadata.
	Info() Info
	// Start runs any background workers, opens pools, and blocks until ctx
	// cancels or a fatal error occurs. If Start returns nil, ctx was cancelled
	// cleanly.
	Start(ctx context.Context) error
	// Ready reports whether the plugin is prepared to serve traffic.
	// Used by the /readyz endpoint. Should return an error describing the
	// first failing dependency check.
	Ready(ctx context.Context) error
	// Shutdown performs graceful drain. Invoked after ctx cancel. Must be
	// idempotent; the runner may call it multiple times.
	Shutdown(ctx context.Context) error
}

// Base provides a zero-value foundation for plugin implementations. Embed this
// in your concrete type to pick up sane defaults for Ready/Shutdown and a
// structured logger keyed to the plugin name.
//
//	type MyPlugin struct {
//	    sdkplugin.Base
//	    pool *pgxpool.Pool
//	}
type Base struct {
	PluginInfo Info
	Logger     *slog.Logger
	StartedAt  time.Time
}

// Info returns the embedded PluginInfo. Implement your own Info() to override.
func (b *Base) Info() Info { return b.PluginInfo }

// Ready is the default Ready() implementation — always nil. Override in your
// concrete plugin to check DB pools, upstream APIs, etc.
func (b *Base) Ready(ctx context.Context) error { return nil }

// Shutdown is the default Shutdown() — no-op. Override if you hold resources.
func (b *Base) Shutdown(ctx context.Context) error { return nil }

// Uptime reports how long the plugin has been running. Zero if never started.
func (b *Base) Uptime() time.Duration {
	if b.StartedAt.IsZero() {
		return 0
	}
	return time.Since(b.StartedAt)
}

// Validate returns an error if required Info fields are missing.
func (i Info) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("plugin: Info.Name is required")
	}
	if i.Version == "" {
		return fmt.Errorf("plugin %q: Info.Version is required", i.Name)
	}
	if i.Tier != "free" && i.Tier != "pro" {
		return fmt.Errorf("plugin %q: Info.Tier must be 'free' or 'pro', got %q", i.Name, i.Tier)
	}
	return nil
}
