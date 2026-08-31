package migrationai_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/migrationai"
)

// ── proposal.go tests ────────────────────────────────────────────────────────

func TestSanitizeSlug_Basic(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"add user soft delete", "add_user_soft_delete"},
		{"Add User Table", "add_user_table"},
		{"my--migration!!", "my_migration"},
		{"", "migration"},
		{"  spaces  ", "spaces"},
	}
	for _, c := range cases {
		got := migrationai.ExportedSanitizeSlug(c.input)
		if got != c.want {
			t.Errorf("sanitizeSlug(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}

func TestWriteMigrationFile_AdditiveOnly(t *testing.T) {
	dir := t.TempDir()

	sql := `ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
DROP TABLE old_users;
ALTER TABLE users DROP COLUMN legacy_field;`

	path, err := migrationai.WriteMigrationFile(dir, "test_migration", sql, "")
	if err != nil {
		t.Fatalf("WriteMigrationFile: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	body := string(content)

	// ALTER TABLE ADD COLUMN must be present as safe SQL.
	if !strings.Contains(body, "ADD COLUMN") {
		t.Error("expected ADD COLUMN to be present in safe SQL")
	}

	// DROP TABLE must be blocked (converted to comment).
	if strings.Contains(body, "DROP TABLE") && !strings.Contains(body, "-- DROP TABLE") {
		t.Error("expected DROP TABLE to be blocked (commented out)")
	}

	// ALTER TABLE DROP COLUMN must be blocked.
	if strings.Contains(body, "DROP COLUMN") && !strings.Contains(body, "-- ") {
		t.Error("expected DROP COLUMN to be blocked")
	}
}

func TestWriteMigrationFile_FileExists(t *testing.T) {
	dir := t.TempDir()
	_, err := migrationai.WriteMigrationFile(dir, "simple", "SELECT 1;", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 migration file, got %d", len(entries))
	}
	if !strings.HasSuffix(entries[0].Name(), "_auto_simple.sql") {
		t.Errorf("unexpected filename: %s", entries[0].Name())
	}
}

func TestDeleteMigrationFile_NoOp(t *testing.T) {
	// Deleting a non-existent file must be a no-op.
	if err := migrationai.DeleteMigrationFile("/tmp/nonexistent_nself_test.sql"); err != nil {
		t.Errorf("expected no error on non-existent file, got: %v", err)
	}
}

func TestDeleteMigrationFile_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql")
	if err := os.WriteFile(path, []byte("-- test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := migrationai.DeleteMigrationFile(path); err != nil {
		t.Fatalf("DeleteMigrationFile: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

// ── ast_go.go tests ──────────────────────────────────────────────────────────

func TestParseGoStructs_Basic(t *testing.T) {
	src := `package models
type User struct {
	ID        int    ` + "`db:\"id\"`" + `
	Name      string ` + "`db:\"name\"`" + `
	Email     string
	DeletedAt *time.Time
}
type internal struct {
	hidden string
}
`
	structs, err := migrationai.ParseGoStructs(src)
	if err != nil {
		t.Fatalf("ParseGoStructs: %v", err)
	}
	if len(structs) != 1 {
		t.Fatalf("expected 1 exported struct, got %d", len(structs))
	}
	if structs[0].Name != "User" {
		t.Errorf("expected struct name User, got %s", structs[0].Name)
	}
	if len(structs[0].Fields) != 4 {
		t.Errorf("expected 4 fields, got %d: %+v", len(structs[0].Fields), structs[0].Fields)
	}
}

func TestDiffGoStructs_AddField(t *testing.T) {
	oldSrc := `package m
type User struct { ID int; Name string }`
	newSrc := `package m
type User struct { ID int; Name string; DeletedAt *time.Time }`

	old, _ := migrationai.ParseGoStructs(oldSrc)
	newS, _ := migrationai.ParseGoStructs(newSrc)

	deltas := migrationai.DiffGoStructs(old, newS)
	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d: %+v", len(deltas), deltas)
	}
	if deltas[0].FieldName != "DeletedAt" {
		t.Errorf("expected DeletedAt, got %s", deltas[0].FieldName)
	}
	if deltas[0].Op != migrationai.OpAdd {
		t.Error("expected OpAdd")
	}
}

func TestDiffGoStructs_RemoveField(t *testing.T) {
	oldSrc := `package m
type User struct { ID int; Name string; Legacy string }`
	newSrc := `package m
type User struct { ID int; Name string }`

	old, _ := migrationai.ParseGoStructs(oldSrc)
	newS, _ := migrationai.ParseGoStructs(newSrc)

	deltas := migrationai.DiffGoStructs(old, newS)
	var removes int
	for _, d := range deltas {
		if d.Op == migrationai.OpRemove && d.FieldName == "Legacy" {
			removes++
		}
	}
	if removes != 1 {
		t.Errorf("expected 1 remove delta for Legacy, got %d (all: %+v)", removes, deltas)
	}
}

// ── ast_ts.go tests ──────────────────────────────────────────────────────────

func TestParseTSInterfaces_Basic(t *testing.T) {
	src := `export interface User {
  id: string;
  name: string;
  email?: string;
}
`
	ifaces := migrationai.ParseTSInterfaces(src)
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}
	if ifaces[0].Name != "User" {
		t.Errorf("expected name User, got %s", ifaces[0].Name)
	}
	if len(ifaces[0].Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(ifaces[0].Fields))
	}
	for _, f := range ifaces[0].Fields {
		if f.Name == "email" && !f.Optional {
			t.Error("expected email to be optional")
		}
	}
}

func TestDiffTSInterfaces_AddField(t *testing.T) {
	oldSrc := `interface User { id: string; name: string; }`
	newSrc := `interface User { id: string; name: string; deletedAt?: string; }`

	old := migrationai.ParseTSInterfaces(oldSrc)
	newI := migrationai.ParseTSInterfaces(newSrc)

	deltas := migrationai.DiffTSInterfaces(old, newI)
	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d", len(deltas))
	}
	if deltas[0].FieldName != "deletedAt" {
		t.Errorf("expected deletedAt, got %s", deltas[0].FieldName)
	}
}

// ── generator.go tests ───────────────────────────────────────────────────────

func TestGenerator_Generate_HappyPath(t *testing.T) {
	// Spin up a mock AI server.
	mockResp := migrationai.GenerateResponse{
		SQL:        "ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;",
		Confidence: 0.92,
		AIModel:    "test-model",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/migration/generate" || r.Method != http.MethodPost {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := migrationai.Config{
		AIEndpoint:    srv.URL,
		ConfidenceMin: 0.75,
		MigrationsDir: dir,
		AutoApply:     false,
		Timeout:       5000000000, // 5s
	}

	gen := migrationai.NewGenerator(cfg)
	proposal, err := gen.Generate(t.Context(), "add soft delete to users", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if proposal.FilePath == "" {
		t.Error("expected FilePath to be set")
	}
	if proposal.Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got %.2f", proposal.Confidence)
	}

	// Verify file was written.
	if _, err := os.Stat(proposal.FilePath); err != nil {
		t.Errorf("migration file not found at %s: %v", proposal.FilePath, err)
	}
}

func TestGenerator_Generate_LowConfidence(t *testing.T) {
	mockResp := migrationai.GenerateResponse{
		SQL:        "ALTER TABLE foo ADD COLUMN bar INT;",
		Confidence: 0.30, // below 0.75
		AIModel:    "test-model",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := migrationai.Config{
		AIEndpoint:    srv.URL,
		ConfidenceMin: 0.75,
		MigrationsDir: dir,
		AutoApply:     false,
		Timeout:       5000000000,
	}

	gen := migrationai.NewGenerator(cfg)
	_, err := gen.Generate(t.Context(), "add bar", nil)
	if err == nil {
		t.Fatal("expected ErrLowConfidence, got nil")
	}
	lcErr, ok := err.(*migrationai.ErrLowConfidence)
	if !ok {
		t.Fatalf("expected *ErrLowConfidence, got %T: %v", err, err)
	}
	if lcErr.Got != 0.30 {
		t.Errorf("expected Got=0.30, got %.2f", lcErr.Got)
	}
}

func TestAutoApply_NeverTrue(t *testing.T) {
	cfg := migrationai.DefaultConfig()
	if cfg.AutoApply {
		t.Error("DefaultConfig().AutoApply must always be false — never auto-apply in production")
	}
}
