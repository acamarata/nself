package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestCheckHasuraMetadataYAML_MissingFilter verifies PERM-HASURA-01 fires when
// a np_* table is tracked in metadata but has no row filter for the user role.
// Acceptance criterion: FAIL result with "HASURA-FILTER-MISSING" in message.
func TestCheckHasuraMetadataYAML_MissingFilter(t *testing.T) {
	dir := t.TempDir()
	writeTableYAML(t, dir, "public_np_posts.yaml", `
table:
  schema: public
  name: np_posts
select_permissions:
  - role: user
    permission:
      filter: {}
      columns: [id, title]
`)
	results := CheckHasuraMetadataYAML(context.Background(), dir, false)
	found := false
	for _, r := range results {
		if containsStr(r.Message, "HASURA-FILTER-MISSING") && containsStr(r.Message, "np_posts") {
			found = true
			if r.Status != "warn" {
				t.Errorf("expected status=warn, got %q", r.Status)
			}
		}
	}
	if !found {
		t.Errorf("expected HASURA-FILTER-MISSING violation for np_posts; got results: %+v", results)
	}
}

// TestCheckHasuraMetadataYAML_CorrectSourceAccountID verifies PERM-HASURA-01
// passes when the user-role filter references source_account_id.
// Acceptance criterion: no violation for this table.
func TestCheckHasuraMetadataYAML_CorrectSourceAccountID(t *testing.T) {
	dir := t.TempDir()
	writeTableYAML(t, dir, "public_np_tasks.yaml", `
table:
  schema: public
  name: np_tasks
select_permissions:
  - role: user
    permission:
      filter:
        source_account_id:
          _eq: X-Source-Account
      columns: [id, title, done]
`)
	results := CheckHasuraMetadataYAML(context.Background(), dir, false)
	for _, r := range results {
		if containsStr(r.Name, "np_tasks") && containsStr(r.Message, "HASURA-FILTER-MISSING") {
			t.Errorf("unexpected violation for np_tasks: %+v", r)
		}
	}
	// Should have a single pass result.
	if len(results) == 1 && results[0].Status != "pass" {
		t.Errorf("expected pass result, got %+v", results[0])
	}
}

// TestCheckHasuraMetadataYAML_CorrectTenantID verifies PERM-HASURA-01 passes
// when the user-role filter references tenant_id.
// Acceptance criterion: no violation for this table.
func TestCheckHasuraMetadataYAML_CorrectTenantID(t *testing.T) {
	dir := t.TempDir()
	writeTableYAML(t, dir, "public_np_tenants.yaml", `
table:
  schema: public
  name: np_tenant_data
select_permissions:
  - role: user
    permission:
      filter:
        tenant_id:
          _eq: X-Hasura-Tenant-Id
      columns: [id, data]
`)
	results := CheckHasuraMetadataYAML(context.Background(), dir, false)
	for _, r := range results {
		if containsStr(r.Name, "np_tenant_data") && containsStr(r.Message, "HASURA-FILTER-MISSING") {
			t.Errorf("unexpected violation for np_tenant_data: %+v", r)
		}
	}
	if len(results) == 1 && results[0].Status != "pass" {
		t.Errorf("expected pass result, got %+v", results[0])
	}
}

// TestCheckHasuraMetadataYAML_NoneTable verifies PERM-HASURA-01 does NOT fire
// for a non-np_* table (no false positive for tenancy-none tables).
// Acceptance criterion: no HASURA-FILTER-MISSING result at all.
func TestCheckHasuraMetadataYAML_NoneTable(t *testing.T) {
	dir := t.TempDir()
	writeTableYAML(t, dir, "public_users.yaml", `
table:
  schema: public
  name: users
select_permissions:
  - role: user
    permission:
      filter: {}
      columns: [id, email]
`)
	results := CheckHasuraMetadataYAML(context.Background(), dir, false)
	for _, r := range results {
		if containsStr(r.Message, "HASURA-FILTER-MISSING") {
			t.Errorf("unexpected HASURA-FILTER-MISSING for non-np_ table: %+v", r)
		}
	}
}

// TestCheckHasuraMetadataYAML_NoMetadataDir verifies the check passes gracefully
// when no hasura/metadata directory exists (e.g. YAML metadata not yet exported).
func TestCheckHasuraMetadataYAML_NoMetadataDir(t *testing.T) {
	dir := t.TempDir()
	results := CheckHasuraMetadataYAML(context.Background(), dir, false)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass when no metadata dir, got %q: %s", results[0].Status, results[0].Message)
	}
}

// writeTableYAML is a test helper that creates hasura/metadata/ inside dir
// and writes the given YAML content to a per-table file.
func writeTableYAML(t *testing.T, projectDir, filename, content string) {
	t.Helper()
	metaDir := filepath.Join(projectDir, "hasura", "metadata")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("create metadata dir: %v", err)
	}
	path := filepath.Join(metaDir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}
