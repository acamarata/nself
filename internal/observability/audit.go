package observability

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// AuditEvent represents a structured audit log entry.
type AuditEvent struct {
	Event    string            // e.g. "user.login", "admin.delete", "permission.denied"
	Actor    string            // who performed the action (user ID or system identifier)
	Target   string            // what was acted upon (resource type:id)
	Metadata map[string]string // additional context
}

// auditLogger is a dedicated logger for audit events, configured to emit
// to stdout with job=audit labels for Promtail to route to the audit Loki tenant.
var auditLogger *slog.Logger

func init() {
	env := os.Getenv("NSELF_ENV")
	if env == "" {
		env = "dev"
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // audit logs are always info+
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = "ts"
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	auditLogger = slog.New(handler).With(
		"job", "audit",
		"service", "audit",
		"env", env,
	)
}

// AuditLog emits a structured audit event. These events are routed by Promtail
// to the audit-specific Loki tenant (audit-$ENV) based on the job=audit label.
func AuditLog(ctx context.Context, event AuditEvent) {
	attrs := []slog.Attr{
		slog.String("audit_event", event.Event),
		slog.String("actor", event.Actor),
		slog.String("target", event.Target),
		slog.String("audit_ts", time.Now().UTC().Format(time.RFC3339Nano)),
	}

	for k, v := range event.Metadata {
		attrs = append(attrs, slog.String("meta."+k, v))
	}

	// Extract trace context if available.
	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}

	auditLogger.LogAttrs(ctx, slog.LevelInfo, "audit: "+event.Event, attrs...)
}
