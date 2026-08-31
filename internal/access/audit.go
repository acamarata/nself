package access

// Purpose: a local, append-only audit trail of every grant/revoke mutation,
// so "was this key actually granted" never again depends on someone's
// memory of a manual ssh session (the failure mode issue #238 was filed
// against). The log lives on the operator's own machine, not the remote
// host — it records who ran what, not a remote-side event.
// Inputs: an auditRecord per mutation.
// Outputs: one key=value line appended to ~/.nself/access-audit.log.
// Constraints: never writes private key material, only fingerprints; a
// failure to write the audit line is reported to the caller but must never
// block or reverse the grant/revoke it is recording.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// auditRecord is one grant/revoke event.
type auditRecord struct {
	Action      string // "grant" or "revoke"
	Host        string
	User        string
	Fingerprint string
	Sudo        bool
	Docker      bool
	Expires     *time.Time
}

// currentAuditLogPath is a function variable so tests can redirect audit
// output to a temp file instead of the operator's real home directory.
// SetAuditLogPathForTest is the only intended caller of the setter half.
var currentAuditLogPath = defaultAuditLogPath

// defaultAuditLogPath returns ~/.nself/access-audit.log, falling back to a
// temp-dir path when the home directory can't be resolved (mirrors the
// fallback convention in internal/plugin/paths.go).
func defaultAuditLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".nself", "access-audit.log")
	}
	return filepath.Join(home, ".nself", "access-audit.log")
}

// SetAuditLogPathForTest redirects the audit log to path for the duration of
// a test and returns a restore function to call in a defer.
func SetAuditLogPathForTest(path string) func() {
	old := currentAuditLogPath
	currentAuditLogPath = func() string { return path }
	return func() { currentAuditLogPath = old }
}

// writeAudit appends one line to the local access audit log.
func writeAudit(r auditRecord) error {
	path := currentAuditLogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create audit log dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer func() { _ = f.Close() }()

	line := fmt.Sprintf("ts=%s action=%s host=%s user=%s fingerprint=%s sudo=%t docker=%t",
		time.Now().UTC().Format(time.RFC3339),
		sanitizeField(r.Action), sanitizeField(r.Host), sanitizeField(r.User),
		r.Fingerprint, r.Sudo, r.Docker)
	if r.Expires != nil {
		line += " expires=" + r.Expires.UTC().Format("2006-01-02")
	}
	line += "\n"

	_, err = f.WriteString(line)
	return err
}

// sanitizeField removes characters that could break the key=value log format.
func sanitizeField(s string) string {
	return strings.NewReplacer(" ", "_", "\n", "", "\r", "", "=", "_").Replace(s)
}
