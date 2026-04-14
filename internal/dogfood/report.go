package dogfood

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SaveReport writes the audit report as JSON to the project's .nself directory.
func SaveReport(projectDir string, report *AuditReport) error {
	dir := filepath.Join(projectDir, ".nself", "dogfood")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dogfood dir: %w", err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	filename := fmt.Sprintf("audit-%s.json", report.RanAt.Format("2006-01-02T150405"))
	return os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}

// GetLatestReport reads the most recent audit report.
func GetLatestReport(projectDir string) (*AuditReport, error) {
	dir := filepath.Join(projectDir, ".nself", "dogfood")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dogfood dir: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no audit reports found")
	}

	// Entries are sorted by name, last is newest due to timestamp format
	latest := entries[len(entries)-1]
	data, err := os.ReadFile(filepath.Join(dir, latest.Name()))
	if err != nil {
		return nil, err
	}

	var report AuditReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// GetReportsSince returns all reports since the given time.
func GetReportsSince(projectDir string, since time.Time) ([]*AuditReport, error) {
	dir := filepath.Join(projectDir, ".nself", "dogfood")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var reports []*AuditReport
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(since) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var report AuditReport
		if err := json.Unmarshal(data, &report); err != nil {
			continue
		}
		reports = append(reports, &report)
	}
	return reports, nil
}
