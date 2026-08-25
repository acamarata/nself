package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Purpose: backup-freshness checks — last full backup age and the Hasura
// metadata backup age (BACKUP-METADATA-01).
// Inputs: a context (unused by BackupChecks today) and the project directory.
// Outputs: []CheckResult, or a single CheckResult for the metadata backup.
// Constraints: split out of system.go (CLI-R12) as a pure move; no behavior
// changed.

// BackupChecks verifies backup health.
func BackupChecks(_ context.Context, projectDir string) []CheckResult {
	var results []CheckResult

	// Check last backup age
	backupDir := filepath.Join(projectDir, ".nself", "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		results = append(results, CheckResult{Section: "backups", Name: "Backup directory", Status: "warn",
			Message: "no backup directory found", FixCmd: "nself backup create"})
		return results
	}

	if len(entries) == 0 {
		results = append(results, CheckResult{Section: "backups", Name: "Last backup", Status: "fail",
			Message: "no backups found", FixCmd: "nself backup create"})
		return results
	}

	// Check most recent backup file modification time
	var newest time.Time
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}

	age := time.Since(newest)
	if age > 26*time.Hour {
		results = append(results, CheckResult{Section: "backups", Name: "Last backup", Status: "fail",
			Message: fmt.Sprintf("last backup is %s old (>26h)", age.Round(time.Minute)),
			FixCmd:  "nself backup create"})
	} else {
		results = append(results, CheckResult{Section: "backups", Name: "Last backup", Status: "pass",
			Message: fmt.Sprintf("last backup %s ago", age.Round(time.Minute))})
	}

	// BACKUP-METADATA-01: verify a Hasura metadata backup exists and is < 36h old.
	results = append(results, CheckHasuraMetadataBackup(backupDir))

	return results
}

// CheckHasuraMetadataBackup verifies that a hasura-metadata-*.json file exists
// in backupDir and is less than 36 hours old (implements BACKUP-METADATA-01).
func CheckHasuraMetadataBackup(backupDir string) CheckResult {
	const checkName = "Hasura metadata backup (BACKUP-METADATA-01)"
	const maxAge = 36 * time.Hour

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return CheckResult{
			Section: "backups",
			Name:    checkName,
			Status:  "warn",
			Message: "backup directory not found; run nself backup hasura-metadata",
			FixCmd:  "nself backup hasura-metadata",
		}
	}

	var newest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Match files like hasura-metadata-2026-04-30.json
		if len(name) < len("hasura-metadata-") || name[:len("hasura-metadata-")] != "hasura-metadata-" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}

	if newest.IsZero() {
		return CheckResult{
			Section: "backups",
			Name:    checkName,
			Status:  "fail",
			Message: "no Hasura metadata backup found",
			FixCmd:  "nself backup hasura-metadata",
		}
	}

	age := time.Since(newest)
	if age > maxAge {
		return CheckResult{
			Section: "backups",
			Name:    checkName,
			Status:  "fail",
			Message: fmt.Sprintf("last Hasura metadata backup is %s old (>36h)", age.Round(time.Minute)),
			FixCmd:  "nself backup hasura-metadata",
		}
	}

	return CheckResult{
		Section: "backups",
		Name:    checkName,
		Status:  "pass",
		Message: fmt.Sprintf("last Hasura metadata backup %s ago", age.Round(time.Minute)),
	}
}
