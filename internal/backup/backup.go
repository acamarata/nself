// Package backup provides path validation helpers for backup operations.
//
// Usage:
//
//	absPath, err := backup.ValidateBackupPath(cfg.Backup.Dir, userSuppliedPath)
package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateBackupPath resolves userPath relative to backupDir and verifies that
// the resolved absolute path is contained within backupDir. This prevents
// path traversal attacks (e.g. "../../../etc/passwd").
//
// backupDir is the trusted root directory for backups (e.g. "/var/backups").
// userPath is the caller-supplied relative or absolute path.
//
// Returns the resolved absolute path on success, or an error if the path
// escapes backupDir.
func ValidateBackupPath(backupDir, userPath string) (string, error) {
	absBase, err := filepath.Abs(backupDir)
	if err != nil {
		return "", fmt.Errorf("resolve backup directory: %w", err)
	}
	// Resolve symlinks on the base directory so a symlinked root cannot be used
	// to bypass prefix checks.
	if resolvedBase, err := filepath.EvalSymlinks(absBase); err == nil {
		absBase = resolvedBase
	}

	// Join userPath under backupDir before resolving, so that relative segments
	// like "2024/mybackup" are anchored to the base directory.
	joined := filepath.Join(absBase, userPath)

	absTarget, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolve backup path: %w", err)
	}
	// Resolve symlinks on the target. If the target does not yet exist,
	// resolve symlinks on its parent directory to catch symlink attacks on
	// directories that lead to the target.
	if resolvedTarget, err := filepath.EvalSymlinks(absTarget); err == nil {
		absTarget = resolvedTarget
	} else {
		// Target doesn't exist yet — resolve symlinks on parent.
		parent := filepath.Dir(absTarget)
		if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil {
			absTarget = filepath.Join(resolvedParent, filepath.Base(absTarget))
		}
	}

	// The resolved target must be the base itself or a descendant of it.
	// Use a trailing separator to prevent a prefix like "/var/backups-evil"
	// from matching "/var/backups".
	prefix := absBase + string(filepath.Separator)
	if absTarget != absBase && !strings.HasPrefix(absTarget, prefix) {
		return "", fmt.Errorf("path traversal detected: %q escapes backup directory %q", userPath, backupDir)
	}

	return absTarget, nil
}

// ListBackups returns the names of all regular files in backupDir. The returned
// slice is empty (not nil) when the directory contains no files. Directories
// and other non-regular entries are skipped.
func ListBackups(backupDir string) ([]string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return []string{}, fmt.Errorf("read backup directory: %w", err)
	}

	names := []string{}
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}
