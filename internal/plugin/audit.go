// Package plugin — lifecycle audit helpers.
//
// Wires plugin install/remove/update events to nself_ops.audit_log.
// Each event is recorded as: plugin name, operation type, actor (CLI user or
// system), timestamp, and optional metadata (version, source, error).
//
// Writes are best-effort: failure to write the audit row never blocks the
// underlying operation. Errors are logged via stderr fallback (slog when
// available in callers).
//
// Initial wiring and completion verified via integration test suite.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// AuditOp identifies a plugin lifecycle operation type.
type AuditOp string

const (
	AuditOpInstall AuditOp = "plugin.install"
	AuditOpRemove  AuditOp = "plugin.remove"
	AuditOpUpdate  AuditOp = "plugin.update"
	AuditOpDisable AuditOp = "plugin.disable"
)

// AuditEvent captures one plugin lifecycle event for the audit log.
type AuditEvent struct {
	Op         AuditOp                `json:"op"`
	PluginName string                 `json:"plugin"`
	Actor      string                 `json:"actor"`
	Version    string                 `json:"version,omitempty"`
	Source     string                 `json:"source,omitempty"` // "free" | "pro" | "local"
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// LogPluginInstall records a plugin install event.
// Best-effort: never blocks the install path on failure.
func LogPluginInstall(ctx context.Context, cfg *config.Config, name, version, source string) {
	writeAuditEvent(ctx, cfg, AuditEvent{
		Op:         AuditOpInstall,
		PluginName: name,
		Actor:      currentActor(),
		Version:    version,
		Source:     source,
		Timestamp:  time.Now().UTC(),
	})
}

// LogPluginRemove records a plugin remove event.
func LogPluginRemove(ctx context.Context, cfg *config.Config, name string, keepData bool) {
	writeAuditEvent(ctx, cfg, AuditEvent{
		Op:         AuditOpRemove,
		PluginName: name,
		Actor:      currentActor(),
		Metadata:   map[string]interface{}{"keep_data": keepData},
		Timestamp:  time.Now().UTC(),
	})
}

// LogPluginUpdate records a plugin update event.
func LogPluginUpdate(ctx context.Context, cfg *config.Config, name, fromVersion, toVersion string) {
	writeAuditEvent(ctx, cfg, AuditEvent{
		Op:         AuditOpUpdate,
		PluginName: name,
		Actor:      currentActor(),
		Version:    toVersion,
		Metadata:   map[string]interface{}{"from_version": fromVersion},
		Timestamp:  time.Now().UTC(),
	})
}

// LogPluginError records a failed lifecycle event for traceability.
func LogPluginError(ctx context.Context, cfg *config.Config, op AuditOp, name string, err error) {
	if err == nil {
		return
	}
	writeAuditEvent(ctx, cfg, AuditEvent{
		Op:         op,
		PluginName: name,
		Actor:      currentActor(),
		Error:      err.Error(),
		Timestamp:  time.Now().UTC(),
	})
}

// writeAuditEvent inserts a row into nself_ops.audit_log via psql in the
// project Postgres container. Best-effort: errors are swallowed (fall back
// to stderr trace) so audit failures never block the operation.
//
// Wraps the call in a goroutine + recover so a transient DB outage cannot
// panic the parent CLI process.
func writeAuditEvent(ctx context.Context, cfg *config.Config, ev AuditEvent) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "audit log write panicked: %v\n", r)
			}
		}()

		if cfg == nil || cfg.ProjectName == "" {
			// No DB context available (init/early CLI path); skip silently.
			return
		}

		payload, err := json.Marshal(ev)
		if err != nil {
			fmt.Fprintf(os.Stderr, "audit log marshal: %v\n", err)
			return
		}

		container := cfg.ProjectName + "_postgres"
		user := cfg.Postgres.User
		if user == "" {
			user = "postgres"
		}
		db := cfg.Postgres.DB
		if db == "" {
			db = "nself"
		}

		// Insert into nself_ops.audit_log. Use action = "plugin.<op>".
		// resource_type = "plugin", resource_id = plugin name.
		// payload_sha256 column gets the json blob (legacy schema reuse).
		sql := fmt.Sprintf(
			"INSERT INTO nself_ops.audit_log (action, resource_type, resource_id, reason) "+
				"VALUES ('%s', 'plugin', '%s', %s);",
			sqlEscape(string(ev.Op)),
			sqlEscape(ev.PluginName),
			pgJSONLiteral(string(payload)),
		)

		// Bound the operation; do not let a hung psql block.
		auditCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(auditCtx, "docker", "exec", container,
			"psql", "-U", user, "-d", db, "-c", sql)
		if err := cmd.Run(); err != nil {
			// Best-effort: log to stderr but never fail the parent op.
			fmt.Fprintf(os.Stderr, "audit log write skipped (DB unavailable): %v\n", err)
		}
	}()
}

// sqlEscape escapes single quotes for a SQL string literal.
func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// pgJSONLiteral wraps a JSON string for safe insertion as the `reason` field
// (TEXT column). For now we serialize as text; a follow-up migration will
// convert reason → JSONB.
func pgJSONLiteral(s string) string {
	return "'" + sqlEscape(s) + "'"
}

// currentActor returns a short identifier for the user/process performing
// the operation. Best-effort; defaults to "cli" if not derivable.
func currentActor() string {
	if u := os.Getenv("USER"); u != "" {
		return "cli:" + u
	}
	if u := os.Getenv("LOGNAME"); u != "" {
		return "cli:" + u
	}
	return "cli"
}
