// Package promote implements staging-to-production promotion with diff preview and rollback.
package promote

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DiffResult holds the comparison between two environments.
type DiffResult struct {
	Source     string      `json:"source"`
	Target     string      `json:"target"`
	Migrations []DiffEntry `json:"migrations"`
	EnvVars    []DiffEntry `json:"env_vars"`
	Images     []DiffEntry `json:"images"`
	Metadata   []DiffEntry `json:"metadata"`
	Timestamp  time.Time   `json:"timestamp"`
}

// DiffEntry represents a single difference between environments.
type DiffEntry struct {
	Key         string `json:"key"`
	SourceValue string `json:"source_value"`
	TargetValue string `json:"target_value"`
	Status      string `json:"status"` // added, removed, changed, same
}

// PromotionRecord records an executed promotion.
type PromotionRecord struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	Target      string    `json:"target"`
	ApproveID   string    `json:"approve_id"`
	BackupTag   string    `json:"backup_tag"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"` // success, failed, rolled_back
	Error       string    `json:"error,omitempty"`
}

// secretPatterns are env var names whose values should be redacted in diffs.
var secretPatterns = []string{
	"PASSWORD", "SECRET", "KEY", "TOKEN", "CREDENTIAL",
}

// DryRun computes the diff between source and target environments.
func DryRun(ctx context.Context, projectDir, source, target string) (*DiffResult, error) {
	result := &DiffResult{
		Source:    source,
		Target:    target,
		Timestamp: time.Now(),
	}

	// Compare env files
	sourceEnv, err := loadEnvFile(filepath.Join(projectDir, fmt.Sprintf(".env.%s", source)))
	if err != nil {
		return nil, fmt.Errorf("load source env: %w", err)
	}
	targetEnv, err := loadEnvFile(filepath.Join(projectDir, fmt.Sprintf(".env.%s", target)))
	if err != nil {
		return nil, fmt.Errorf("load target env: %w", err)
	}

	result.EnvVars = diffMaps(sourceEnv, targetEnv)

	// Compare migration directories
	sourceMigDir := filepath.Join(projectDir, "migrations", source)
	targetMigDir := filepath.Join(projectDir, "migrations", target)
	result.Migrations = diffDirectories(sourceMigDir, targetMigDir)

	// Compare Docker image digests
	result.Images = diffDockerImages(ctx, projectDir, source, target)

	return result, nil
}

// Execute runs the full promotion pipeline.
func Execute(ctx context.Context, projectDir, source, target, approveID string) (*PromotionRecord, error) {
	record := &PromotionRecord{
		ID:        fmt.Sprintf("promo-%d", time.Now().UnixNano()),
		Source:    source,
		Target:    target,
		ApproveID: approveID,
		StartedAt: time.Now(),
	}

	// 1. Pre-flight: run nself doctor --deep
	cmd := exec.CommandContext(ctx, "nself", "doctor", "--full")
	if out, err := cmd.CombinedOutput(); err != nil {
		record.Status = "failed"
		record.Error = fmt.Sprintf("pre-flight doctor failed: %s", string(out))
		record.CompletedAt = time.Now()
		return record, fmt.Errorf("pre-flight check failed: %w", err)
	}

	// 2. Create backup with promotion tag
	backupTag := fmt.Sprintf("pre-promote-%s-%d", target, time.Now().Unix())
	record.BackupTag = backupTag
	cmd = exec.CommandContext(ctx, "nself", "backup", "create", "--tag", backupTag)
	if out, err := cmd.CombinedOutput(); err != nil {
		record.Status = "failed"
		record.Error = fmt.Sprintf("backup failed: %s", string(out))
		record.CompletedAt = time.Now()
		return record, fmt.Errorf("backup failed: %w", err)
	}

	// 3. Apply migrations
	cmd = exec.CommandContext(ctx, "nself", "db", "migrate", "--env", target)
	if out, err := cmd.CombinedOutput(); err != nil {
		record.Status = "failed"
		record.Error = fmt.Sprintf("migration failed: %s", string(out))
		record.CompletedAt = time.Now()
		return record, fmt.Errorf("migration failed: %w", err)
	}

	// 4. Rebuild and restart services
	cmd = exec.CommandContext(ctx, "nself", "build", "--env", target)
	if out, err := cmd.CombinedOutput(); err != nil {
		record.Status = "failed"
		record.Error = fmt.Sprintf("build failed: %s", string(out))
		record.CompletedAt = time.Now()
		return record, fmt.Errorf("build failed: %w", err)
	}

	cmd = exec.CommandContext(ctx, "nself", "restart", "--env", target)
	if out, err := cmd.CombinedOutput(); err != nil {
		record.Status = "failed"
		record.Error = fmt.Sprintf("restart failed: %s", string(out))
		record.CompletedAt = time.Now()
		return record, fmt.Errorf("restart failed: %w", err)
	}

	// 5. Post-flight: run doctor again
	cmd = exec.CommandContext(ctx, "nself", "doctor")
	if out, err := cmd.CombinedOutput(); err != nil {
		record.Status = "failed"
		record.Error = fmt.Sprintf("post-flight doctor failed: %s", string(out))
		record.CompletedAt = time.Now()
		return record, fmt.Errorf("post-flight check failed: %w", err)
	}

	record.Status = "success"
	record.CompletedAt = time.Now()

	// Save promotion record
	if err := saveRecord(projectDir, record); err != nil {
		return record, fmt.Errorf("save record: %w", err)
	}

	return record, nil
}

// Rollback restores to the pre-promotion backup tag.
func Rollback(ctx context.Context, projectDir, backupTag string) error {
	if backupTag == "" {
		// Find latest pre-promote backup
		tag, err := findLatestPromoteBackup(projectDir)
		if err != nil {
			return fmt.Errorf("find backup: %w", err)
		}
		backupTag = tag
	}

	cmd := exec.CommandContext(ctx, "nself", "backup", "restore", "--tag", backupTag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore failed: %s: %w", string(out), err)
	}
	return nil
}

// loadEnvFile reads a .env file into a key-value map.
func loadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, nil
}

// diffMaps compares two key-value maps and returns diff entries with redaction.
func diffMaps(source, target map[string]string) []DiffEntry {
	var entries []DiffEntry
	seen := make(map[string]bool)

	for k, sv := range source {
		seen[k] = true
		tv, exists := target[k]
		entry := DiffEntry{Key: k, SourceValue: redactIfSecret(k, sv)}
		if !exists {
			entry.Status = "added"
			entry.TargetValue = "(not set)"
		} else if sv == tv {
			entry.Status = "same"
			entry.TargetValue = redactIfSecret(k, tv)
		} else {
			entry.Status = "changed"
			entry.TargetValue = redactIfSecret(k, tv)
		}
		entries = append(entries, entry)
	}

	for k, tv := range target {
		if !seen[k] {
			entries = append(entries, DiffEntry{
				Key:         k,
				SourceValue: "(not set)",
				TargetValue: redactIfSecret(k, tv),
				Status:      "removed",
			})
		}
	}
	return entries
}

func redactIfSecret(key, value string) string {
	upper := strings.ToUpper(key)
	for _, pat := range secretPatterns {
		if strings.Contains(upper, pat) {
			if len(value) > 4 {
				return value[:2] + "***" + value[len(value)-2:]
			}
			return "***"
		}
	}
	return value
}

func diffDirectories(source, target string) []DiffEntry {
	var entries []DiffEntry
	sourceFiles := listFiles(source)
	targetFiles := listFiles(target)

	seen := make(map[string]bool)
	for _, f := range sourceFiles {
		seen[f] = true
		found := false
		for _, tf := range targetFiles {
			if tf == f {
				found = true
				break
			}
		}
		if found {
			entries = append(entries, DiffEntry{Key: f, Status: "same"})
		} else {
			entries = append(entries, DiffEntry{Key: f, Status: "added"})
		}
	}
	for _, f := range targetFiles {
		if !seen[f] {
			entries = append(entries, DiffEntry{Key: f, Status: "removed"})
		}
	}
	return entries
}

func listFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files
}

func diffDockerImages(_ context.Context, _ string, _, _ string) []DiffEntry {
	// Docker image diff requires reading docker-compose files for each env
	// and comparing image tags/digests. Stubbed for now with basic structure.
	return nil
}

func saveRecord(projectDir string, record *PromotionRecord) error {
	dir := filepath.Join(projectDir, ".nself", "promotions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, record.ID+".json"), data, 0o644)
}

func findLatestPromoteBackup(projectDir string) (string, error) {
	dir := filepath.Join(projectDir, ".nself", "promotions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("no promotion records found")
	}
	// Find the most recent promotion record
	var latest *PromotionRecord
	for i := len(entries) - 1; i >= 0; i-- {
		data, err := os.ReadFile(filepath.Join(dir, entries[i].Name()))
		if err != nil {
			continue
		}
		var rec PromotionRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		if rec.BackupTag != "" {
			latest = &rec
			break
		}
	}
	if latest == nil {
		return "", fmt.Errorf("no promotion backup tags found")
	}
	return latest.BackupTag, nil
}
