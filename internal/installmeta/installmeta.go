// Package installmeta persists install-time metadata to ~/.nself/install-meta.json.
//
// The install source (NSELF_INSTALL_REF env var) is captured at install time
// by install.sh and persisted here. The value is sanitized to prevent injection.
//
// Schema (v1):
//
//	{
//	  "install_source": "hn",       // sanitized ref from ?ref= query param or NSELF_INSTALL_REF env
//	  "install_date":  "2026-04-20T10:00:00Z",
//	  "cli_version":   "1.0.9",
//	  "schema_version": 1,
//	  "_p95_note": "install_source included in ping_api telemetry from v1.1.0 (S54-T05)"
//	}
//
// P95 integration point: the telemetry package reads InstallSource() and sends it
// in the ping_api telemetry payload as "install_source" (S54-T05, v1.1.0).
//
// S57-T11 — 2026-04-20
package installmeta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/version"
)

const (
	metaFile      = "install-meta.json"
	schemaVersion = 1
	// maxRefLen caps the ref value to prevent oversized JSON.
	maxRefLen = 64
)

// allowedRefPattern matches safe ref values: alphanumeric, hyphen, underscore, dot.
// Any other character is stripped during sanitization.
var allowedRefPattern = regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)

// Meta holds install-time metadata.
type Meta struct {
	InstallSource string `json:"install_source"`
	InstallDate   string `json:"install_date"`
	CLIVersion    string `json:"cli_version"`
	SchemaVersion int    `json:"schema_version"`
	P95Note       string `json:"_p95_note"`
}

// metaPath returns the path to install-meta.json in the nself home dir.
func metaPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home dir: %w", err)
	}
	return filepath.Join(home, ".nself", metaFile), nil
}

// SanitizeRef strips disallowed characters and caps length.
// Input: raw ref from NSELF_INSTALL_REF or ?ref= query param.
// Output: safe ASCII string suitable for JSON and logging.
func SanitizeRef(raw string) string {
	// Strip disallowed characters
	clean := allowedRefPattern.ReplaceAllString(raw, "")
	// Cap length
	if len(clean) > maxRefLen {
		clean = clean[:maxRefLen]
	}
	// Lowercase for canonical form
	return strings.ToLower(clean)
}

// Write creates or overwrites ~/.nself/install-meta.json with the given ref.
// The nself home directory is created if it does not exist.
func Write(rawRef string) error {
	path, err := metaPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cannot create .nself dir: %w", err)
	}

	meta := Meta{
		InstallSource: SanitizeRef(rawRef),
		InstallDate:   time.Now().UTC().Format(time.RFC3339),
		CLIVersion:    version.Version,
		SchemaVersion: schemaVersion,
		P95Note:       "install_source included in ping_api telemetry from v1.1.0 (S54-T05)",
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal install-meta: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write install-meta: %w", err)
	}

	return nil
}

// Read reads the current install-meta.json. Returns zero Meta if the file does not exist.
func Read() (Meta, error) {
	path, err := metaPath()
	if err != nil {
		return Meta{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Meta{}, nil
	}
	if err != nil {
		return Meta{}, fmt.Errorf("cannot read install-meta: %w", err)
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return Meta{}, fmt.Errorf("cannot parse install-meta: %w", err)
	}

	return meta, nil
}

// InstallSource returns the sanitized install source string, or "unknown" if not set.
// Safe to call at any time — never errors.
func InstallSource() string {
	meta, err := Read()
	if err != nil || meta.InstallSource == "" {
		return "unknown"
	}
	return meta.InstallSource
}
