package database

// Purpose: fixture-tree unit tests for the !include resolution + JSON
// conversion machinery in hasura_apply.go, including the historical
// "list-wrapped !include'd file" incident (43/48 web/backend table files
// found list-wrapped since a9eb61bf, 2026-09-03) as an explicit, must-error
// regression case rather than a silent malformed-metadata send.
// Inputs: temp-dir fixture trees built per test via t.TempDir()/os.WriteFile.
// Outputs: pass/fail against resolveIncludes / convertYAMLToJSON /
// validateNoListWrappedEntries.
// Constraints: pure functions, no live Hasura/network required.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// writeFixtureFile writes content to dir/name, creating parent dirs as needed.
func writeFixtureFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestResolveIncludes_WellFormedTableTree resolves a two-table fixture tree
// (tables.yaml !include-ing two well-formed, bare-mapping table files) and
// verifies the JSON-safe result has both tables in order with no leftover
// list nesting.
func TestResolveIncludes_WellFormedTableTree(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "tables.yaml", "- !include public_np_foo.yaml\n- !include public_np_bar.yaml\n")
	writeFixtureFile(t, dir, "public_np_foo.yaml", "table:\n  schema: public\n  name: np_foo\nselect_permissions: []\n")
	writeFixtureFile(t, dir, "public_np_bar.yaml", "table:\n  schema: public\n  name: np_bar\nselect_permissions: []\n")

	resolved, err := resolveIncludes(dir, filepath.Join(dir, "tables.yaml"))
	if err != nil {
		t.Fatalf("resolveIncludes: %v", err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(resolved, &doc); err != nil {
		t.Fatalf("unmarshal resolved YAML: %v\n--- resolved ---\n%s", err, resolved)
	}

	items, ok := doc.([]interface{})
	if !ok {
		t.Fatalf("expected top-level list, got %T", doc)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 table entries, got %d", len(items))
	}
	for i, item := range items {
		if _, isMap := item.(map[string]interface{}); !isMap {
			t.Errorf("entry %d: expected a bare mapping, got %T", i, item)
		}
	}

	jsonBytes, err := json.Marshal(convertYAMLToJSON(doc))
	if err != nil {
		t.Fatalf("convertYAMLToJSON -> json.Marshal: %v", err)
	}
	var roundTrip []map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &roundTrip); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if got := roundTrip[0]["table"].(map[string]interface{})["name"]; got != "np_foo" {
		t.Errorf("entry 0 table.name = %v, want np_foo", got)
	}
}

// TestResolveIncludes_NestedConfigTree resolves the standard Hasura CLI
// shape (config.yaml !include-ing tables.yaml, which itself !include-s
// per-table files) to confirm recursive !include resolution works, not just
// one level.
func TestResolveIncludes_NestedConfigTree(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "config.yaml", "version: 3\ntables: !include tables.yaml\n")
	writeFixtureFile(t, dir, "tables.yaml", "- !include public_np_foo.yaml\n")
	writeFixtureFile(t, dir, "public_np_foo.yaml", "table:\n  schema: public\n  name: np_foo\n")

	resolved, err := resolveIncludes(dir, filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("resolveIncludes: %v", err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(resolved, &doc); err != nil {
		t.Fatalf("unmarshal resolved YAML: %v\n--- resolved ---\n%s", err, resolved)
	}
	root, ok := doc.(map[string]interface{})
	if !ok {
		t.Fatalf("expected top-level mapping, got %T", doc)
	}
	tables, ok := root["tables"].([]interface{})
	if !ok || len(tables) != 1 {
		t.Fatalf("expected tables to be a 1-element list, got %#v", root["tables"])
	}
	if _, isMap := tables[0].(map[string]interface{}); !isMap {
		t.Errorf("tables[0]: expected a bare mapping, got %T", tables[0])
	}
}

// TestValidateNoListWrappedEntries_CatchesHistoricalIncident reproduces the
// exact 2026-09-03 web/backend incident: an !include'd table file that is
// itself wrapped in a one-element YAML list ("- table: ...") instead of a
// bare mapping ("table: ..."). Before this fix, resolveIncludes' textual
// splice silently produced a nested-list entry that would have reached the
// Hasura API as malformed metadata; validateNoListWrappedEntries must now
// catch it with a clear, actionable error naming the section and index.
func TestValidateNoListWrappedEntries_CatchesHistoricalIncident(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "tables.yaml", "- !include public_np_foo.yaml\n- !include public_np_bar.yaml\n")
	// public_np_foo.yaml is well-formed; public_np_bar.yaml is list-wrapped
	// (the historical bug shape) — the error must name index 1.
	writeFixtureFile(t, dir, "public_np_foo.yaml", "table:\n  schema: public\n  name: np_foo\n")
	writeFixtureFile(t, dir, "public_np_bar.yaml", "- table:\n    schema: public\n    name: np_bar\n")

	resolved, err := resolveIncludes(dir, filepath.Join(dir, "tables.yaml"))
	if err != nil {
		t.Fatalf("resolveIncludes: %v", err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(resolved, &doc); err != nil {
		t.Fatalf("unmarshal resolved YAML: %v\n--- resolved ---\n%s", err, resolved)
	}

	err = validateNoListWrappedEntries(doc)
	if err == nil {
		t.Fatal("expected validateNoListWrappedEntries to return a clear error for the list-wrapped file, got nil")
	}
	const want = "tables[1] is a YAML list, not an object"
	if got := err.Error(); !strings.Contains(got, want) {
		t.Errorf("error = %q, want it to contain %q", got, want)
	}
}

// TestValidateNoListWrappedEntries_NonMapDocIsNoop covers metadata.json's
// legacy JSON-derived shape and any other non-object top-level document:
// there is nothing to validate, so it must return nil rather than panic on
// the type assertion.
func TestValidateNoListWrappedEntries_NonMapDocIsNoop(t *testing.T) {
	if err := validateNoListWrappedEntries([]interface{}{"unexpected"}); err != nil {
		t.Errorf("expected nil for a non-map document, got %v", err)
	}
	if err := validateNoListWrappedEntries(nil); err != nil {
		t.Errorf("expected nil for a nil document, got %v", err)
	}
}
