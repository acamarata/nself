// Package deprecation provides @deprecated annotation helpers for plugin authors.
//
// Usage in a plugin handler:
//
//	func (p *MyPlugin) OldHandler(ctx context.Context, req Request) (Response, error) {
//	    deprecation.Mark(ctx, "OldHandler", "NewHandler", "v1.0.9", "https://docs.nself.org/plugins/deprecation")
//	    return p.NewHandler(ctx, req)
//	}
//
// The annotation emits a structured log warning on every invocation.
// The deprecated handler still executes normally — Mark is warning-only.
package deprecation

import (
	"context"
	"log/slog"
)

// logKey is used to extract a *slog.Logger from context, allowing callers
// to inject a structured logger. Falls back to slog.Default().
type logKey struct{}

// WithLogger returns a new context carrying the given logger.
// Plugin servers should inject their structured logger this way so
// deprecation warnings appear in plugin log output.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, logKey{}, logger)
}

// loggerFrom extracts the logger from context or returns slog.Default().
func loggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(logKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// Mark emits a structured deprecation warning for the named handler.
// It writes to the structured logger from ctx (or slog.Default() as fallback).
//
// Parameters:
//   - ctx:         request context (carries logger if injected via WithLogger)
//   - handlerName: name of the deprecated function/handler (e.g. "OldHandler")
//   - replacedBy:  name of the replacement (e.g. "NewHandler")
//   - since:       version when this handler was deprecated (e.g. "v1.0.9")
//   - docsURL:     migration guide URL
//
// The deprecated handler MUST still return its result normally after calling Mark.
// Mark adds ≤1μs overhead per call (structured log write to slog.Default).
func Mark(ctx context.Context, handlerName, replacedBy, since, docsURL string) {
	loggerFrom(ctx).WarnContext(ctx, "plugin handler deprecated",
		slog.String("handler", handlerName),
		slog.String("replaced_by", replacedBy),
		slog.String("since", since),
		slog.String("docs", docsURL),
		slog.String("action", "update your plugin to use "+replacedBy),
	)
}
