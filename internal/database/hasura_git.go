package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// MetadataDrift describes the difference between live Hasura metadata and committed files.
type MetadataDrift struct {
	// TablesAdded are tracked tables in live Hasura not in committed metadata.
	TablesAdded []string
	// TablesRemoved are tables in committed metadata not in live Hasura.
	TablesRemoved []string
	// PermissionsChanged are tables where permissions differ.
	PermissionsChanged []string
	// RelationshipsChanged are tables where relationships differ.
	RelationshipsChanged []string
	// IsClean is true when live matches committed exactly.
	IsClean bool
}

// HasuraGitStatus describes the git state of metadata files.
type HasuraGitStatus struct {
	Branch       string
	CommitHash   string
	LastExportAt time.Time
	Modified     []string // metadata files with uncommitted changes
	IsClean      bool
}

// metadataTableEntry holds the table name and schema extracted from live metadata.
type metadataTableEntry struct {
	Schema string
	Name   string
}

// extractTablesFromMetadata parses a raw Hasura export_metadata JSON response
// and returns the list of tracked tables across all sources.
func extractTablesFromMetadata(data []byte) ([]metadataTableEntry, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse metadata JSON: %w", err)
	}

	sources, ok := raw["sources"]
	if !ok {
		return nil, nil
	}
	sourceList, ok := sources.([]interface{})
	if !ok {
		return nil, nil
	}

	var entries []metadataTableEntry
	for _, src := range sourceList {
		srcMap, ok := src.(map[string]interface{})
		if !ok {
			continue
		}
		tables, ok := srcMap["tables"]
		if !ok {
			continue
		}
		tableList, ok := tables.([]interface{})
		if !ok {
			continue
		}
		for _, tbl := range tableList {
			tblMap, ok := tbl.(map[string]interface{})
			if !ok {
				continue
			}
			tableObj, ok := tblMap["table"]
			if !ok {
				continue
			}
			tblDef, ok := tableObj.(map[string]interface{})
			if !ok {
				continue
			}
			schema, _ := tblDef["schema"].(string)
			name, _ := tblDef["name"].(string)
			if name != "" {
				entries = append(entries, metadataTableEntry{Schema: schema, Name: name})
			}
		}
	}
	return entries, nil
}

// extractTableNamesFromYAMLDir scans the on-disk YAML table files under
// {tablesDir} and returns the table names encoded in each filename.
// Hasura CLI stores tables as {schema}_{name}.yaml files.
func extractTableNamesFromYAMLDir(tablesDir string) ([]string, error) {
	entries, err := os.ReadDir(tablesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tables dir %s: %w", tablesDir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		// Strip .yaml suffix; the remainder is used as the table identifier.
		names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
	}
	return names, nil
}

// DetectMetadataDrift compares live Hasura metadata against committed files in hasura/metadata/.
// It exports current metadata via the Hasura API and diffs against the on-disk files.
func DetectMetadataDrift(ctx context.Context, cfg *config.Config, projectDir string) (MetadataDrift, error) {
	liveData, err := postMetadata(ctx, cfg, metadataRequest{
		Type: "export_metadata",
		Args: map[string]interface{}{},
	})
	if err != nil {
		return MetadataDrift{}, fmt.Errorf("export live metadata: %w", err)
	}

	liveTables, err := extractTablesFromMetadata(liveData)
	if err != nil {
		return MetadataDrift{}, fmt.Errorf("parse live metadata tables: %w", err)
	}

	tablesDir := filepath.Join(projectDir, "hasura", "metadata", "databases", "default", "tables")
	diskNames, err := extractTableNamesFromYAMLDir(tablesDir)
	if err != nil {
		return MetadataDrift{}, err
	}

	// Build lookup sets.
	liveSet := make(map[string]bool, len(liveTables))
	for _, t := range liveTables {
		key := t.Schema + "_" + t.Name
		liveSet[key] = true
	}

	diskSet := make(map[string]bool, len(diskNames))
	for _, n := range diskNames {
		diskSet[n] = true
	}

	var added, removed []string
	for key := range liveSet {
		if !diskSet[key] {
			added = append(added, key)
		}
	}
	for key := range diskSet {
		if !liveSet[key] {
			removed = append(removed, key)
		}
	}

	drift := MetadataDrift{
		TablesAdded:   added,
		TablesRemoved: removed,
		IsClean:       len(added) == 0 && len(removed) == 0,
	}
	return drift, nil
}

// ExportAndCommitMetadata exports Hasura metadata and creates a git commit.
// commitMsg is the commit message; if empty, a default message is generated.
// Returns the commit hash on success.
func ExportAndCommitMetadata(ctx context.Context, cfg *config.Config, projectDir string, commitMsg string) (string, error) {
	if _, err := HasuraExportToYAML(ctx, cfg, projectDir); err != nil {
		return "", fmt.Errorf("export metadata to YAML: %w", err)
	}

	if commitMsg == "" {
		commitMsg = fmt.Sprintf("chore(hasura): export metadata %s", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	}

	addCmd := exec.CommandContext(ctx, "git", "-C", projectDir, "add", "hasura/metadata/")
	if out, err := addCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add hasura/metadata/: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	commitCmd := exec.CommandContext(ctx, "git", "-C", projectDir, "commit", "-m", commitMsg)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	hashOut, err := exec.CommandContext(ctx, "git", "-C", projectDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(hashOut)), nil
}

// GetGitStatus returns the git status of the hasura/metadata/ directory.
func GetGitStatus(projectDir string) (HasuraGitStatus, error) {
	branchOut, err := exec.Command("git", "-C", projectDir, "branch", "--show-current").Output()
	if err != nil {
		return HasuraGitStatus{}, fmt.Errorf("git branch --show-current: %w", err)
	}
	branch := strings.TrimSpace(string(branchOut))

	hashOut, err := exec.Command("git", "-C", projectDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return HasuraGitStatus{}, fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	commitHash := strings.TrimSpace(string(hashOut))
	if len(commitHash) > 7 {
		commitHash = commitHash[:7]
	}

	statusOut, err := exec.Command("git", "-C", projectDir, "status", "--porcelain", "hasura/metadata/").Output()
	if err != nil {
		return HasuraGitStatus{}, fmt.Errorf("git status hasura/metadata/: %w", err)
	}

	var modified []string
	for _, line := range strings.Split(string(statusOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// porcelain format: XY filename
		if len(line) > 3 {
			modified = append(modified, line[3:])
		}
	}

	return HasuraGitStatus{
		Branch:     branch,
		CommitHash: commitHash,
		Modified:   modified,
		IsClean:    len(modified) == 0,
	}, nil
}

// ApplyMetadataFromGit checks out the metadata files at the given git ref and
// then applies them via the standard metadata apply mechanism.
// ref can be a branch name, tag, or commit hash.
func ApplyMetadataFromGit(ctx context.Context, cfg *config.Config, projectDir string, ref string) error {
	checkoutCmd := exec.CommandContext(ctx, "git", "-C", projectDir, "checkout", ref, "--", "hasura/metadata/")
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s -- hasura/metadata/: %w\n%s", ref, err, strings.TrimSpace(string(out)))
	}

	if err := HasuraApplyMetadata(ctx, cfg, projectDir); err != nil {
		return fmt.Errorf("apply metadata from ref %s: %w", ref, err)
	}
	return nil
}

// GenerateTrackTableSQL generates a comment block describing how to track a new
// table in Hasura metadata. It returns a YAML snippet and CLI hint rather than
// raw SQL because table tracking is a Hasura metadata operation.
func GenerateTrackTableSQL(schema, table string) string {
	return fmt.Sprintf(`-- Track table in Hasura
-- Run: nself db hasura metadata apply
-- Or add to hasura/metadata/databases/default/tables/%s_%s.yaml:
--
-- table:
--   schema: %s
--   name: %s
`, schema, table, schema, table)
}

// MetadataSnapshot captures the current live metadata as a JSON snapshot.
// Useful for comparing before/after schema migrations.
func MetadataSnapshot(ctx context.Context, cfg *config.Config) ([]byte, error) {
	data, err := postMetadata(ctx, cfg, metadataRequest{
		Type: "export_metadata",
		Args: map[string]interface{}{},
	})
	if err != nil {
		return nil, fmt.Errorf("capture metadata snapshot: %w", err)
	}
	return data, nil
}

// CompareSnapshots returns a human-readable diff between two metadata snapshots.
// It compares table lists extracted from the sources and reports additions and removals.
func CompareSnapshots(before, after []byte) (string, error) {
	beforeTables, err := extractTablesFromMetadata(before)
	if err != nil {
		return "", fmt.Errorf("parse before snapshot: %w", err)
	}
	afterTables, err := extractTablesFromMetadata(after)
	if err != nil {
		return "", fmt.Errorf("parse after snapshot: %w", err)
	}

	beforeSet := make(map[string]bool, len(beforeTables))
	for _, t := range beforeTables {
		beforeSet[t.Schema+"."+t.Name] = true
	}
	afterSet := make(map[string]bool, len(afterTables))
	for _, t := range afterTables {
		afterSet[t.Schema+"."+t.Name] = true
	}

	var added, removed []string
	for k := range afterSet {
		if !beforeSet[k] {
			added = append(added, k)
		}
	}
	for k := range beforeSet {
		if !afterSet[k] {
			removed = append(removed, k)
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		return "No changes between snapshots.", nil
	}

	var sb strings.Builder
	sb.WriteString("Snapshot diff:\n")
	for _, t := range added {
		sb.WriteString(fmt.Sprintf("  + %s\n", t))
	}
	for _, t := range removed {
		sb.WriteString(fmt.Sprintf("  - %s\n", t))
	}
	return sb.String(), nil
}
