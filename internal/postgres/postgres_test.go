package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateInitScript_CreatesFile verifies that GenerateInitScript creates
// the expected SQL init file.
func TestGenerateInitScript_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateInitScript(dir); err != nil {
		t.Fatalf("GenerateInitScript: %v", err)
	}

	initPath := filepath.Join(dir, "postgres", "init", "01-init.sql")
	if _, err := os.Stat(initPath); err != nil {
		t.Fatalf("init script not created at %s: %v", initPath, err)
	}
}

// TestGenerateInitScript_ContainsRequiredSchemas verifies the generated SQL
// includes the required schemas.
func TestGenerateInitScript_ContainsRequiredSchemas(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateInitScript(dir); err != nil {
		t.Fatalf("GenerateInitScript: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "postgres", "init", "01-init.sql"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	required := []string{"CREATE SCHEMA IF NOT EXISTS auth", "CREATE SCHEMA IF NOT EXISTS storage"}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Errorf("init SQL missing: %q", want)
		}
	}
}

// TestGenerateInitScript_Idempotent verifies calling GenerateInitScript twice
// does not error (existing file is overwritten without error).
func TestGenerateInitScript_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateInitScript(dir); err != nil {
		t.Fatalf("first GenerateInitScript: %v", err)
	}
	if err := GenerateInitScript(dir); err != nil {
		t.Fatalf("second GenerateInitScript: %v", err)
	}
}

// TestDedupStrings_RemovesDuplicates verifies the internal helper used by
// InstallExtensions deduplicates extension lists.
func TestDedupStrings_RemovesDuplicates(t *testing.T) {
	input := []string{"pgcrypto", "uuid-ossp", "pgcrypto", "citext", "uuid-ossp"}
	got := dedupStrings(input)

	seen := make(map[string]int)
	for _, s := range got {
		seen[s]++
		if seen[s] > 1 {
			t.Errorf("dedupStrings: duplicate %q in output", s)
		}
	}
	if len(got) != 3 {
		t.Errorf("expected 3 unique items, got %d: %v", len(got), got)
	}
}

// TestGenerateInitScript_CreatesNpPluginsScript verifies that GenerateInitScript
// writes the 04-np-plugins.sql file alongside the main init script.
func TestGenerateInitScript_CreatesNpPluginsScript(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateInitScript(dir); err != nil {
		t.Fatalf("GenerateInitScript: %v", err)
	}

	pluginsPath := filepath.Join(dir, "postgres", "init", "04-np-plugins.sql")
	data, err := os.ReadFile(pluginsPath)
	if err != nil {
		t.Fatalf("04-np-plugins.sql not created: %v", err)
	}

	content := string(data)
	// Must use CREATE TABLE IF NOT EXISTS for idempotency.
	if !strings.Contains(content, "CREATE TABLE IF NOT EXISTS np_plugins") {
		t.Errorf("04-np-plugins.sql missing CREATE TABLE IF NOT EXISTS np_plugins; got:\n%s", content)
	}
	// Must include the index with IF NOT EXISTS for idempotency.
	if !strings.Contains(content, "CREATE INDEX IF NOT EXISTS") {
		t.Errorf("04-np-plugins.sql missing CREATE INDEX IF NOT EXISTS; got:\n%s", content)
	}
}

// TestGenerateInitScript_NpPluginsIdempotent verifies that calling
// GenerateInitScript twice in a row on the same directory does not error.
// This models the "nself build run twice" scenario from the spec.
func TestGenerateInitScript_NpPluginsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateInitScript(dir); err != nil {
		t.Fatalf("first GenerateInitScript: %v", err)
	}
	// Second call must not error — simulates re-running nself build.
	if err := GenerateInitScript(dir); err != nil {
		t.Fatalf("second GenerateInitScript (idempotency check): %v", err)
	}

	// Verify the file is still present and well-formed after the second call.
	pluginsPath := filepath.Join(dir, "postgres", "init", "04-np-plugins.sql")
	data, err := os.ReadFile(pluginsPath)
	if err != nil {
		t.Fatalf("04-np-plugins.sql missing after second call: %v", err)
	}
	if !strings.Contains(string(data), "CREATE TABLE IF NOT EXISTS np_plugins") {
		t.Errorf("04-np-plugins.sql content wrong after second call")
	}
}

// TestEnsureNpPluginsTable_ContainsIfNotExists verifies EnsureNpPluginsTable
// returns SQL that is unconditionally safe to execute more than once.
func TestEnsureNpPluginsTable_ContainsIfNotExists(t *testing.T) {
	sql := EnsureNpPluginsTable()
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS np_plugins") {
		t.Errorf("EnsureNpPluginsTable SQL missing IF NOT EXISTS guard; got:\n%s", sql)
	}
	if !strings.Contains(sql, "CREATE INDEX IF NOT EXISTS") {
		t.Errorf("EnsureNpPluginsTable SQL missing IF NOT EXISTS index guard; got:\n%s", sql)
	}
}

// TestDefaultExtensions_NonEmpty verifies DefaultExtensions contains at least
// the minimum required extensions.
func TestDefaultExtensions_NonEmpty(t *testing.T) {
	if len(DefaultExtensions) == 0 {
		t.Error("DefaultExtensions should not be empty")
	}
	required := []string{"uuid-ossp", "pgcrypto"}
	for _, ext := range required {
		found := false
		for _, de := range DefaultExtensions {
			if de == ext {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultExtensions missing required extension: %q", ext)
		}
	}
}
