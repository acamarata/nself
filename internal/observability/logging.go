package observability

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

// LogConfig holds configuration for the structured JSON logger.
type LogConfig struct {
	Service   string // e.g. "claw", "ai", "mux"
	Env       string // "dev", "staging", "prod"
	Version   string // e.g. "1.0.3"
	Tenant    string // multi-tenant isolation key
	RequestID string // per-request correlation ID
	UserID    string // optional, PII-gated
}

// NewJSONLogger creates a slog.Logger with JSON output on stdout that includes
// all 12 standard fields per the observability spec: ts, level, service, env,
// version, msg, trace_id, span_id, request_id, user_id, tenant, caller.
//
// The returned logger wraps a TraceLogHandler (for trace_id/span_id injection)
// around a PII-redacting handler around the base slog.JSONHandler.
func NewJSONLogger(cfg LogConfig) *slog.Logger {
	level := resolveLogLevel(cfg.Env)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename "time" → "ts" for spec compliance.
			if a.Key == slog.TimeKey {
				a.Key = "ts"
			}
			// Rename "source" → "caller" and format as file:line.
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a = slog.String("caller", formatCaller(src))
				}
			}
			return a
		},
	}

	base := slog.NewJSONHandler(os.Stdout, opts)

	// Stack: JSONHandler → RedactHandler → TraceLogHandler
	redacted := NewRedactHandler(base)
	traced := NewTraceLogHandler(redacted)

	// Pre-populate standard fields.
	attrs := []slog.Attr{
		slog.String("service", cfg.Service),
		slog.String("env", resolveEnv(cfg.Env)),
		slog.String("version", cfg.Version),
	}
	if cfg.Tenant != "" {
		attrs = append(attrs, slog.String("tenant", cfg.Tenant))
	}
	if cfg.RequestID != "" {
		attrs = append(attrs, slog.String("request_id", cfg.RequestID))
	}
	if cfg.UserID != "" {
		attrs = append(attrs, slog.String("user_id", cfg.UserID))
	}

	handler := traced.WithAttrs(attrs)
	return slog.New(handler)
}

// SetDefaultLogger creates and sets the global slog default logger.
func SetDefaultLogger(cfg LogConfig) {
	logger := NewJSONLogger(cfg)
	slog.SetDefault(logger)
}

// resolveLogLevel returns the slog.Level based on LOG_LEVEL env var or env defaults.
// Defaults: dev=debug, staging=info, prod=info.
func resolveLogLevel(env string) slog.Level {
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		switch strings.ToLower(v) {
		case "debug":
			return slog.LevelDebug
		case "info":
			return slog.LevelInfo
		case "warn", "warning":
			return slog.LevelWarn
		case "error":
			return slog.LevelError
		}
	}

	switch strings.ToLower(env) {
	case "dev", "development", "":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func resolveEnv(env string) string {
	if env != "" {
		return env
	}
	if v := os.Getenv("NSELF_ENV"); v != "" {
		return v
	}
	return "dev"
}

func formatCaller(src *slog.Source) string {
	if src == nil {
		return ""
	}
	// Shorten path: keep only last two segments.
	file := src.File
	if idx := strings.LastIndex(file, "/"); idx > 0 {
		if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
			file = file[idx2+1:]
		}
	}
	return file + ":" + itoa(src.Line)
}

func itoa(i int) string {
	// Simple int-to-string without importing strconv.
	if i == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// StackTraceHandler is a slog.Handler that captures stack traces for warn+ levels.
// It wraps any inner handler.
type StackTraceHandler struct {
	inner slog.Handler
}

// NewStackTraceHandler wraps a handler to add err.stack for warn and above.
func NewStackTraceHandler(inner slog.Handler) *StackTraceHandler {
	return &StackTraceHandler{inner: inner}
}

func (h *StackTraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *StackTraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelWarn {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		r.AddAttrs(slog.String("err.stack", string(buf[:n])))
	}
	return h.inner.Handle(ctx, r)
}

func (h *StackTraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &StackTraceHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *StackTraceHandler) WithGroup(name string) slog.Handler {
	return &StackTraceHandler{inner: h.inner.WithGroup(name)}
}
