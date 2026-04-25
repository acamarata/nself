// Package logger provides a standardized slog.Logger factory used by every
// nSelf Go plugin. Output defaults to JSON on stderr with per-plugin keying.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options controls logger construction.
type Options struct {
	Plugin  string     // plugin name added to every record, e.g. "ai"
	Level   slog.Level // slog.LevelDebug / Info / Warn / Error
	Format  string     // "json" (default) or "text"
	Writer  io.Writer  // defaults to os.Stderr
	Version string     // plugin version added to every record
}

// New returns a *slog.Logger configured for nSelf plugins. Every record carries
// plugin=<name> and version=<version> attrs so multi-plugin log streams
// (docker logs, Loki, etc.) can be filtered by plugin.
func New(opts Options) *slog.Logger {
	if opts.Writer == nil {
		opts.Writer = os.Stderr
	}
	if opts.Format == "" {
		opts.Format = "json"
	}
	handlerOpts := &slog.HandlerOptions{
		Level: opts.Level,
	}

	var handler slog.Handler
	if strings.EqualFold(opts.Format, "text") {
		handler = slog.NewTextHandler(opts.Writer, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(opts.Writer, handlerOpts)
	}

	attrs := make([]slog.Attr, 0, 2)
	if opts.Plugin != "" {
		attrs = append(attrs, slog.String("plugin", opts.Plugin))
	}
	if opts.Version != "" {
		attrs = append(attrs, slog.String("version", opts.Version))
	}
	if len(attrs) > 0 {
		handler = handler.WithAttrs(attrs)
	}
	return slog.New(handler)
}

// ParseLevel converts a string like "debug", "info", "warn", "error" to
// slog.Level. Unknown values fall back to Info. Matches the convention used
// by every plugin's LOG_LEVEL env var.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error", "err":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
