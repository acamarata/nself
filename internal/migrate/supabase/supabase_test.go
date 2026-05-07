package supabase_test

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/migrate/supabase"
)

func TestGenerate_MigrationSQL_containsTable(t *testing.T) {
	result := &supabase.PullResult{
		Tables: []supabase.Table{
			{
				Schema: "public",
				Name:   "profiles",
				Columns: []supabase.Column{
					{Name: "id", DataType: "uuid", IsNullable: false},
					{Name: "email", DataType: "text", IsNullable: true},
				},
			},
		},
	}

	arts := supabase.Generate("test-ref", result)

	if !strings.Contains(arts.MigrationSQL, `"profiles"`) {
		t.Errorf("expected migration SQL to contain table name, got:\n%s", arts.MigrationSQL)
	}
	if !strings.Contains(arts.MigrationSQL, "uuid NOT NULL") {
		t.Errorf("expected NOT NULL column in migration SQL, got:\n%s", arts.MigrationSQL)
	}
}

func TestGenerate_HasuraMetadata_containsTable(t *testing.T) {
	result := &supabase.PullResult{
		Tables: []supabase.Table{
			{Schema: "public", Name: "items"},
		},
	}

	arts := supabase.Generate("test-ref", result)

	if !strings.Contains(arts.HasuraMetadata, "name: items") {
		t.Errorf("expected Hasura metadata to contain table, got:\n%s", arts.HasuraMetadata)
	}
	if !strings.Contains(arts.HasuraMetadata, "version: 3") {
		t.Errorf("expected metadata version header, got:\n%s", arts.HasuraMetadata)
	}
}

func TestGenerate_AuthImportScript_noUsers(t *testing.T) {
	result := &supabase.PullResult{}
	arts := supabase.Generate("test-ref", result)
	if !strings.Contains(arts.AuthImportScript, "No auth users") {
		t.Errorf("expected no-users message in import script, got:\n%s", arts.AuthImportScript)
	}
}

func TestGenerate_AuthImportScript_withUsers(t *testing.T) {
	result := &supabase.PullResult{
		Users: []supabase.AuthUser{
			{ID: "abc-123", Email: "alice@example.com"},
		},
	}
	arts := supabase.Generate("test-ref", result)
	if !strings.Contains(arts.AuthImportScript, "alice@example.com") {
		t.Errorf("expected user email in import script, got:\n%s", arts.AuthImportScript)
	}
}

func TestGenerate_Summary_counts(t *testing.T) {
	result := &supabase.PullResult{
		Tables:  []supabase.Table{{Name: "t1"}, {Name: "t2"}},
		Users:   []supabase.AuthUser{{Email: "u@example.com"}},
		Buckets: []supabase.StorageBucket{{Name: "avatars", Public: true}},
	}
	arts := supabase.Generate("my-project", result)
	if !strings.Contains(arts.Summary, "Tables:          2") {
		t.Errorf("expected table count in summary, got:\n%s", arts.Summary)
	}
	if !strings.Contains(arts.Summary, "Auth users:      1") {
		t.Errorf("expected user count in summary, got:\n%s", arts.Summary)
	}
	if !strings.Contains(arts.Summary, "avatars") {
		t.Errorf("expected bucket name in summary, got:\n%s", arts.Summary)
	}
}

func TestGenerate_RLSStub_note(t *testing.T) {
	result := &supabase.PullResult{
		Tables: []supabase.Table{{Schema: "public", Name: "posts"}},
		// no policies — should generate advisory comment
	}
	arts := supabase.Generate("proj", result)
	if !strings.Contains(arts.MigrationSQL, "RLS policies could not be fetched") {
		t.Errorf("expected RLS advisory note in migration SQL, got:\n%s", arts.MigrationSQL)
	}
}
