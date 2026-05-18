package controlplane

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// auditMu guards auditWriter against concurrent reads/writes. emitAudit is
// called from Run() which may be invoked concurrently in tests or from multiple
// goroutines; without this mutex the package-level variable assignment in
// setAuditWriter and the write in writeAuditLine form a data race.
var auditMu sync.Mutex

// auditWriter is the destination for control-plane audit lines. It defaults to
// os.Stderr and can be replaced in tests via setAuditWriter.
// All access must be guarded by auditMu.
var auditWriter io.Writer = os.Stderr

// setAuditWriter replaces the global audit output for tests. It returns a
// restore function that the test should call in a defer.
func setAuditWriter(w io.Writer) func() {
	auditMu.Lock()
	old := auditWriter
	auditWriter = w
	auditMu.Unlock()
	return func() {
		auditMu.Lock()
		auditWriter = old
		auditMu.Unlock()
	}
}

// auditLine represents one resolved-target audit record emitted by the
// control-plane resolver. Fields that could reveal host→key mappings are
// redacted before the record is created.
type auditLine struct {
	// Timestamp is when the capability was resolved.
	Timestamp time.Time

	// Env is the environment name.
	Env string

	// Server is the server name within the environment.
	Server string

	// Capability is the resolved access level ("manage", "read-only", "hidden").
	Capability string

	// Reason is the human-readable explanation (empty for manage).
	// Key file paths are redacted; only env-var names are retained.
	Reason string

	// SSHReachable, KeyPresent, DockerOK mirror the TargetStatus probe results.
	SSHReachable bool
	KeyPresent   bool
	DockerOK     bool

	// LatencyMS is the SSH probe round-trip in milliseconds.
	LatencyMS int
}

// emitAudit emits one structured audit record per entry in statuses.
//
// Each record is written as a key=value log line to auditWriter (os.Stderr by
// default). Host→key filesystem path mappings are redacted so that the audit
// trail contains only env-var names (safe to log) rather than key file paths
// (which may expose directory structure).
//
// Every call to emitAudit emits exactly one record per TargetStatus; callers
// can rely on this one-to-one guarantee.
func emitAudit(statuses []TargetStatus) {
	for _, ts := range statuses {
		line := toAuditLine(ts)
		writeAuditLine(line)
	}
}

// toAuditLine converts a TargetStatus to an auditLine with redacted paths.
func toAuditLine(ts TargetStatus) auditLine {
	return auditLine{
		Timestamp:    ts.ProbedAt,
		Env:          ts.Env,
		Server:       ts.Server,
		Capability:   string(ts.Capability),
		Reason:       redactKeyPaths(ts.Reason),
		SSHReachable: ts.SSHReachable,
		KeyPresent:   ts.KeyPresent,
		DockerOK:     ts.DockerOK,
		LatencyMS:    ts.LatencyMS,
	}
}

// writeAuditLine formats line as a key=value record and writes it to
// auditWriter. The format is compatible with structured log aggregators.
// auditMu is acquired for the duration of the write to prevent concurrent
// writers from interleaving partial lines.
func writeAuditLine(line auditLine) {
	msg := fmt.Sprintf(
		"level=info component=controlplane ts=%s env=%s server=%s capability=%s"+
			" ssh_reachable=%v key_present=%v docker_ok=%v latency_ms=%d",
		line.Timestamp.UTC().Format(time.RFC3339),
		sanitizeField(line.Env),
		sanitizeField(line.Server),
		sanitizeField(line.Capability),
		line.SSHReachable,
		line.KeyPresent,
		line.DockerOK,
		line.LatencyMS,
	)
	if line.Reason != "" {
		msg += fmt.Sprintf(" reason=%q", line.Reason)
	}
	msg += "\n"

	auditMu.Lock()
	_, _ = fmt.Fprint(auditWriter, msg)
	auditMu.Unlock()
}

// redactKeyPaths removes filesystem path segments from s. An env-var name
// (e.g. "NSELF_SSH_KEY_PROD") is retained. Absolute paths (starting with /)
// and tilde-paths are replaced with "<redacted>".
//
// Examples:
//
//	"no SSH key for NSELF_SSH_KEY_PROD"    → unchanged
//	"key /home/user/.ssh/id_ed25519 missing" → "key <redacted> missing"
func redactKeyPaths(s string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		if len(p) > 0 && (p[0] == '/' || p[0] == '~') {
			parts[i] = "<redacted>"
		}
	}
	return strings.Join(parts, " ")
}

// sanitizeField removes characters that could break the key=value log format.
func sanitizeField(s string) string {
	return strings.NewReplacer(" ", "_", "\n", "", "\r", "", "=", "_").Replace(s)
}
