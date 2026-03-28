package security

import (
	"fmt"
	"os"
	"path/filepath"
)

// PermissionFinding describes a file whose current permissions differ from what
// the security policy requires.
type PermissionFinding struct {
	Path         string
	CurrentMode  os.FileMode
	RequiredMode os.FileMode
	Reason       string
}

// sensitivePatterns maps glob patterns (relative to the project root) to the
// required file mode. Evaluated in the order they are declared at walk time.
var sensitivePatterns = map[string]os.FileMode{
	"*.env":          0600,
	"*.env.*":        0600,
	"backups/*":      0600,
	"ssl/certs/*":    0640,
	".plugin-cache*": 0600,
}

// EnforceFilePermissions sets path to the given mode via os.Chmod.
// Returns an error if the file does not exist or chmod fails.
func EnforceFilePermissions(path string, mode os.FileMode) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("enforcing permissions on %s: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod %s to %04o: %w", path, mode, err)
	}
	return nil
}

// AuditProjectPermissions walks projectDir and checks each file against the
// sensitivePatterns map. Files whose current mode is more permissive than
// required are reported as PermissionFinding values.
//
// Directories are skipped; only regular files are evaluated.
func AuditProjectPermissions(projectDir string) ([]PermissionFinding, error) {
	var findings []PermissionFinding

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// Skip paths we cannot stat; don't abort the whole walk.
			return nil
		}
		if info.IsDir() {
			return nil
		}

		// Compute path relative to projectDir for pattern matching.
		rel, err := filepath.Rel(projectDir, path)
		if err != nil {
			return nil
		}

		for pattern, required := range sensitivePatterns {
			matched, err := filepath.Match(pattern, rel)
			if err != nil {
				// Malformed pattern — skip silently.
				continue
			}
			if !matched {
				continue
			}

			current := info.Mode().Perm()
			if current != required {
				findings = append(findings, PermissionFinding{
					Path:         path,
					CurrentMode:  current,
					RequiredMode: required,
					Reason:       fmt.Sprintf("matches pattern %q: want %04o, got %04o", pattern, required, current),
				})
			}
			// A file can only match one authoritative pattern; stop after first match.
			break
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("auditing project permissions in %s: %w", projectDir, err)
	}

	return findings, nil
}
