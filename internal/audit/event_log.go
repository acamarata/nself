package audit

// Purpose: Security event log writer — appends structured JSON entries to
//          ~/.nself/audit.log for searchable, tamper-evident security records.
// Inputs:  event string (e.g. "plugin-install-bypass"), fields map of key/value pairs.
// Outputs: Appends one JSON line to ~/.nself/audit.log; returns error on failure.
// Constraints: File is created with 0600 permissions if it does not exist.
//              Falls back to /tmp/.nself/audit.log when home dir is unavailable.
//              write is append-only — never truncates or rewrites existing entries.
// SPORT: security-event-log; callers: internal/plugin/installer.go (bypass audit)

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// auditLogPath returns the path to the security audit log file.
// Normally ~/.nself/audit.log; falls back to /tmp/.nself/audit.log.
// Checks the HOME env var first so tests can override the path on all platforms
// (os.UserHomeDir on Windows reads USERPROFILE/HOMEDRIVE+HOMEPATH, not HOME).
func auditLogPath() string {
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".nself", "audit.log")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "audit.log")
	}
	return filepath.Join(home, ".nself", "audit.log")
}

// Write appends a structured JSON entry to the security audit log.
// event is a machine-readable event type (e.g. "plugin-install-bypass").
// fields contains any key/value pairs relevant to the event.
// A "timestamp" key is always added automatically; callers must not include it.
// The file is created with 0600 permissions if it does not exist.
func Write(event string, fields map[string]string) error {
	logPath := auditLogPath()

	// Ensure the parent directory exists with restrictive permissions.
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("audit: creating log directory %s: %w", dir, err)
	}

	// Build the JSON record. Merge caller fields then add mandatory fields.
	record := make(map[string]string, len(fields)+2)
	for k, v := range fields {
		record[k] = v
	}
	record["event"] = event
	record["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("audit: marshalling event %q: %w", event, err)
	}
	line = append(line, '\n')

	// Open for append, create with 0600 if missing.
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("audit: opening log %s: %w", logPath, err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("audit: writing to log %s: %w", logPath, err)
	}
	return nil
}
