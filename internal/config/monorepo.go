package config

import (
	"os"
	"path/filepath"
)

// backendDirCandidates lists the directory names to probe when checking whether
// a given directory is a monorepo root that contains an nself backend project.
// "backend" matches the nself-web Turborepo layout (committed, no dot prefix).
// ".backend" matches consumer app layout (gitignored, dot prefix).
var backendDirCandidates = []string{"backend", ".backend"}

// DetectMonorepoRoot checks whether dir is a monorepo root by looking for an
// nself backend sub-directory containing a .env file. It probes, in order:
//
//   - <dir>/backend/.env
//   - <dir>/.backend/.env
//
// Returns the full path to the backend directory if found, or "" if dir does
// not appear to be a monorepo root. The function never returns an error: a
// missing or unreadable path is treated as "not found".
func DetectMonorepoRoot(dir string) string {
	for _, candidate := range backendDirCandidates {
		backendDir := filepath.Join(dir, candidate)
		envFile := filepath.Join(backendDir, ".env")
		if _, err := os.Stat(envFile); err == nil {
			return backendDir
		}
	}
	return ""
}
