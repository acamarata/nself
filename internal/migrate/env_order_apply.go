package migrate

// env_order_apply.go — CLI-R18 migration shim: applying fixes.
//
// Purpose: Perform the writes DetectEnvOrder (env_order.go) decided are safe:
//          consolidate each preserved value into .env.secrets, and archive a
//          fully-folded legacy .env.ai file. Migrate combines detect+apply in
//          the shape `nself migrate` calls.
// Inputs:  projectDir and an *EnvOrderReport from DetectEnvOrder.
// Outputs: file writes to .env.secrets (mode 0600) and, when applicable, a
//          rename of .env.ai to .env.ai.migrated.
// Constraints: Only ever writes ActionFixed entries — ActionManualReview
//              entries are never touched by design (see env_order.go for
//              why). Idempotent: re-running Apply on an already-applied
//              report is a no-op. Never regenerates NSELF_MASTER_SECRET;
//              upsertEnvVar only replaces/appends the exact key requested.
// SPORT:   cli/internal/migrate — CLI-R18.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Apply performs the writes for every change in report marked ActionFixed,
// consolidating each preserved value into .env.secrets (created if absent).
// It is idempotent: applying an already-applied report is a no-op, and a
// fresh Detect() afterward reports NoChangeNeeded for everything Apply fixed.
// Manual-review entries are left untouched — Apply never guesses on your
// behalf for those.
func Apply(projectDir string, report *EnvOrderReport) error {
	secretsPath := filepath.Join(projectDir, ".env.secrets")

	for i := range report.Changes {
		c := &report.Changes[i]
		if c.Action != ActionFixed {
			continue
		}
		if err := upsertEnvVar(secretsPath, c.Var, c.OldValue); err != nil {
			return fmt.Errorf("writing %s to .env.secrets: %w", c.Var, err)
		}
	}

	// Archive a fully-folded .env.ai (every one of its keys that drifted was
	// ActionFixed — none needed manual review) so it no longer confuses
	// operators, without deleting data.
	aiPath := filepath.Join(projectDir, ".env.ai")
	if _, err := os.Stat(aiPath); err == nil {
		if !hasUnresolvedEnvAI(report) {
			migratedPath := filepath.Join(projectDir, ".env.ai.migrated")
			if err := os.Rename(aiPath, migratedPath); err != nil {
				return fmt.Errorf("archiving .env.ai: %w", err)
			}
			report.AIArchived = true
		}
	}

	return nil
}

// Migrate runs DetectEnvOrder followed by Apply in one call — the shape
// `nself migrate` uses. Returns the report either way so the caller can print
// a summary even when nothing needed fixing.
func Migrate(projectDir string) (*EnvOrderReport, error) {
	report, err := DetectEnvOrder(projectDir)
	if err != nil {
		return nil, err
	}
	if report.NoChangeNeeded {
		return report, nil
	}
	if err := Apply(projectDir, report); err != nil {
		return report, err
	}
	return report, nil
}

// hasUnresolvedEnvAI reports whether any manual-review change originated
// from .env.ai — if so, the file must not be archived yet.
func hasUnresolvedEnvAI(report *EnvOrderReport) bool {
	for _, c := range report.Changes {
		if c.Action == ActionManualReview && c.OldWinner == ".env.ai" {
			return true
		}
	}
	return false
}

// upsertEnvVar writes key=value into the .env file at path, replacing an
// existing unquoted `key=` line if present, or appending one otherwise.
// Creates the file (mode 0600) if it does not exist. Always leaves the file
// at mode 0600 — .env.secrets must never be group/world readable (P15).
func upsertEnvVar(path, key, value string) error {
	line := fmt.Sprintf("%s=%s\n", key, value)

	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		existing = nil
	}

	replaced := false
	var out bytes.Buffer
	if len(existing) > 0 {
		lines := bytes.Split(existing, []byte("\n"))
		// A trailing newline produces one spurious empty split element —
		// drop it so we don't append a blank line on every rewrite.
		if n := len(lines); n > 0 && len(lines[n-1]) == 0 {
			lines = lines[:n-1]
		}
		for _, raw := range lines {
			if !replaced && bytes.HasPrefix(raw, []byte(key+"=")) {
				out.WriteString(line)
				replaced = true
				continue
			}
			out.Write(raw)
			out.WriteByte('\n')
		}
	}
	if !replaced {
		if out.Len() > 0 && out.Bytes()[out.Len()-1] != '\n' {
			out.WriteByte('\n')
		}
		out.WriteString("# CLI-R18 migration: preserved effective value from the legacy cascade order\n")
		out.WriteString(line)
	}

	if err := os.WriteFile(path, out.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return os.Chmod(path, 0o600)
}
