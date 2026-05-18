package firebase

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeJSON writes v as a JSON file at dir/name and returns the full path.
func writeJSON(t *testing.T, dir, name string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// writeJSONSub creates a subdirectory named collName under dir, writes a JSON
// file there, and returns the file path. This matches the Firestore export layout
// where collections are directories.
func writeJSONSub(t *testing.T, dir, collName, fileName string, v any) string {
	t.Helper()
	sub := filepath.Join(dir, collName)
	if err := os.MkdirAll(sub, 0750); err != nil {
		t.Fatalf("mkdir %s: %v", sub, err)
	}
	return writeJSON(t, sub, fileName, v)
}

// ---------------------------------------------------------------------------
// toTableName / toColumnName
// ---------------------------------------------------------------------------

func TestToTableName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "users"},
		{"UserProfiles", "user_profiles"},
		{"myCollection", "my_collection"},
		{"already_snake", "already_snake"},
		// empty string → normalizeName falls back to "field"
		{"", "field"},
		{"A", "a"},
	}
	for _, tc := range tests {
		got := toTableName(tc.input)
		if got != tc.want {
			t.Errorf("toTableName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestToColumnName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"createdAt", "created_at"},
		{"userId", "user_id"},
		{"name", "name"},
		// Leading/trailing underscores are stripped; "__name__" → "name"
		{"__name__", "name"},
	}
	for _, tc := range tests {
		got := toColumnName(tc.input)
		if got != tc.want {
			t.Errorf("toColumnName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// jsonTypeName
// ---------------------------------------------------------------------------

func TestJSONTypeName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"true", "boolean"},
		{"false", "boolean"},
		{"42", "integer"},
		// Floats return "float", not "number"
		{"3.14", "float"},
		{`"hello"`, "string"},
		{`"2024-01-15T10:30:00Z"`, "timestamp"},
		{`{"a":1}`, "object"},
		{"[1,2]", "array"},
		{"null", "null"},
	}
	for _, tc := range tests {
		got := jsonTypeName(json.RawMessage(tc.raw))
		if got != tc.want {
			t.Errorf("jsonTypeName(%s) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// inferSQLType
// ---------------------------------------------------------------------------

func TestInferSQLType(t *testing.T) {
	tests := []struct {
		counts map[string]int
		want   string
	}{
		{map[string]int{"integer": 5}, "BIGINT"},
		// The actual JSON number type for floats is "float", not "number"
		{map[string]int{"float": 3}, "DOUBLE PRECISION"},
		{map[string]int{"boolean": 2}, "BOOLEAN"},
		{map[string]int{"timestamp": 4}, "TIMESTAMPTZ"},
		{map[string]int{"string": 10}, "TEXT"},
		{map[string]int{"object": 1}, "JSONB"},
		{map[string]int{"array": 1}, "JSONB"},
		// mixed types → JSONB
		{map[string]int{"string": 3, "integer": 2}, "JSONB"},
		// null only → TEXT (safe default)
		{map[string]int{"null": 5}, "TEXT"},
	}
	for _, tc := range tests {
		got := inferSQLType(tc.counts)
		if got != tc.want {
			t.Errorf("inferSQLType(%v) = %q, want %q", tc.counts, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// looksLikeTimestamp
// ---------------------------------------------------------------------------

func TestLooksLikeTimestamp(t *testing.T) {
	yes := []string{
		"2024-01-15T10:30:00Z",
		"2024-01-15T10:30:00.000Z",
		"2024-01-15T10:30:00+05:00",
	}
	no := []string{
		"hello",
		"2024-01-15",
		"10:30:00",
		"",
	}
	for _, s := range yes {
		if !looksLikeTimestamp(s) {
			t.Errorf("looksLikeTimestamp(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if looksLikeTimestamp(s) {
			t.Errorf("looksLikeTimestamp(%q) = true, want false", s)
		}
	}
}

// ---------------------------------------------------------------------------
// inferSchema — collection-as-subdirectory layout (canonical Firestore export)
// ---------------------------------------------------------------------------

func TestInferSchema_SubdirFormat(t *testing.T) {
	dir := t.TempDir()

	// Write documents into a "users" subdirectory — this is the canonical
	// Firestore export layout where collection name = parent dir name.
	docs := []map[string]any{
		{"name": "Alice", "age": 30, "active": true},
		{"name": "Bob", "age": 25, "active": false},
	}
	writeJSONSub(t, dir, "users", "docs.json", docs)

	collections, err := inferSchema(dir)
	if err != nil {
		t.Fatalf("inferSchema: %v", err)
	}
	if len(collections) != 1 {
		t.Fatalf("got %d collections, want 1", len(collections))
	}
	c := collections[0]
	if c.TableName != "users" {
		t.Errorf("TableName = %q, want %q", c.TableName, "users")
	}
	if c.SampleCount != 2 {
		t.Errorf("SampleCount = %d, want 2", c.SampleCount)
	}
	// id column must always be first.
	if c.Columns[0].Name != "id" {
		t.Errorf("first column = %q, want \"id\"", c.Columns[0].Name)
	}
	// Verify inferred types for the remaining columns.
	colMap := make(map[string]string)
	for _, col := range c.Columns {
		colMap[col.Name] = col.SQLType
	}
	if colMap["active"] != "BOOLEAN" {
		t.Errorf("active SQLType = %q, want BOOLEAN", colMap["active"])
	}
	if colMap["age"] != "BIGINT" {
		t.Errorf("age SQLType = %q, want BIGINT", colMap["age"])
	}
	if colMap["name"] != "TEXT" {
		t.Errorf("name SQLType = %q, want TEXT", colMap["name"])
	}
}

// ---------------------------------------------------------------------------
// inferSchema — full Firebase __name__ path format
// ---------------------------------------------------------------------------

func TestInferSchema_FullNamePath(t *testing.T) {
	dir := t.TempDir()

	// Full Firebase __name__ path: "projects/p/databases/d/documents/<collection>/<docID>"
	docs := []map[string]any{
		{"__name__": "projects/p/databases/d/documents/orders/o1", "total": 99.99, "status": "shipped"},
	}
	writeJSON(t, dir, "export.json", docs)

	collections, err := inferSchema(dir)
	if err != nil {
		t.Fatalf("inferSchema: %v", err)
	}
	if len(collections) != 1 {
		t.Fatalf("got %d collections, want 1", len(collections))
	}
	if collections[0].TableName != "orders" {
		t.Errorf("TableName = %q, want \"orders\"", collections[0].TableName)
	}
}

// ---------------------------------------------------------------------------
// inferSchema — wrapped "documents" key format
// ---------------------------------------------------------------------------

func TestInferSchema_WrappedFormat(t *testing.T) {
	dir := t.TempDir()

	wrapped := map[string]any{
		"documents": []map[string]any{
			{"__name__": "projects/p/databases/d/documents/products/p1", "price": 9.99},
		},
	}
	writeJSON(t, dir, "export.json", wrapped)

	collections, err := inferSchema(dir)
	if err != nil {
		t.Fatalf("inferSchema: %v", err)
	}
	if len(collections) != 1 {
		t.Fatalf("got %d collections, want 1", len(collections))
	}
	if collections[0].TableName != "products" {
		t.Errorf("TableName = %q, want \"products\"", collections[0].TableName)
	}
}

// ---------------------------------------------------------------------------
// buildSchemaMigration — smoke test
// ---------------------------------------------------------------------------

func TestBuildSchemaMigration(t *testing.T) {
	collections := []CollectionInfo{
		{
			Name:      "users",
			TableName: "users",
			Columns: []ColumnDef{
				{Name: "id", SQLType: "TEXT", Nullable: false},
				{Name: "name", SQLType: "TEXT", Nullable: true},
				{Name: "age", SQLType: "BIGINT", Nullable: true},
			},
			SampleCount: 5,
		},
	}
	sql := string(buildSchemaMigration(collections, "myproject"))
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS") {
		t.Error("missing CREATE TABLE statement")
	}
	if !strings.Contains(sql, "users") {
		t.Error("missing table name in SQL")
	}
	if !strings.Contains(sql, "ENABLE ROW LEVEL SECURITY") {
		t.Error("missing RLS statement")
	}
}

// ---------------------------------------------------------------------------
// buildHasuraMetadata — smoke test
// ---------------------------------------------------------------------------

func TestBuildHasuraMetadata(t *testing.T) {
	collections := []CollectionInfo{
		{Name: "posts", TableName: "posts", SampleCount: 1},
	}
	yaml := string(buildHasuraMetadata(collections))
	if !strings.Contains(yaml, "posts") {
		t.Error("missing table name in Hasura metadata")
	}
	// Hasura metadata YAML should track the table
	if !strings.Contains(yaml, "table") {
		t.Error("missing 'table' key in Hasura metadata")
	}
}

// ---------------------------------------------------------------------------
// Run — DryRun happy path
// ---------------------------------------------------------------------------

func TestRun_DryRun(t *testing.T) {
	dir := t.TempDir()
	// Use full __name__ path so collection is inferred correctly.
	docs := []map[string]any{
		{"__name__": "projects/p/databases/d/documents/items/1", "label": "foo", "qty": 3},
	}
	writeJSON(t, dir, "items.json", docs)

	opts := Options{
		ExportDir:   dir,
		ProjectName: "testproj",
		DryRun:      true,
	}
	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run(DryRun): %v", err)
	}
	if len(result.Tables) == 0 {
		t.Error("expected at least one table in result")
	}
	// DryRun: no files written, paths are empty.
	if result.SchemaSQL != "" {
		t.Errorf("DryRun SchemaSQL should be empty, got %q", result.SchemaSQL)
	}
}

// ---------------------------------------------------------------------------
// Run — writes files with correct permissions
// ---------------------------------------------------------------------------

func TestRun_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	docs := []map[string]any{
		{"__name__": "projects/p/databases/d/documents/products/p1", "name": "Widget", "price": 9.99},
	}
	writeJSON(t, dir, "products.json", docs)

	outDir := filepath.Join(dir, "out")
	opts := Options{
		ExportDir:   dir,
		OutputDir:   outDir,
		ProjectName: "shopproj",
	}
	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, p := range []string{result.SchemaSQL, result.HasuraYAML, result.Summary} {
		if p == "" {
			t.Error("expected non-empty path in result")
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", p, err)
			continue
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("file %s has perm %o, want 0600", p, info.Mode().Perm())
		}
	}
}

// ---------------------------------------------------------------------------
// Run — empty export dir returns error
// ---------------------------------------------------------------------------

func TestRun_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := Run(context.Background(), Options{ExportDir: dir})
	if err == nil {
		t.Error("expected error for empty export dir")
	}
}

// ---------------------------------------------------------------------------
// Run — missing export dir returns error
// ---------------------------------------------------------------------------

func TestRun_MissingDir(t *testing.T) {
	_, err := Run(context.Background(), Options{ExportDir: "/nonexistent/path/xyz"})
	if err == nil {
		t.Error("expected error for nonexistent export dir")
	}
}

// ---------------------------------------------------------------------------
// Run — table name is stable across invocations
// ---------------------------------------------------------------------------

func TestRun_StableTableNames(t *testing.T) {
	dir := t.TempDir()
	docs := []map[string]any{
		{"__name__": "projects/p/databases/d/documents/userProfiles/u1", "email": "a@b.com"},
	}
	writeJSON(t, dir, "up.json", docs)

	opts := Options{ExportDir: dir, ProjectName: "p", DryRun: true}
	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Tables) != 1 {
		t.Fatalf("want 1 table, got %d: %v", len(result.Tables), result.Tables)
	}
	if result.Tables[0] != "user_profiles" {
		t.Errorf("table name = %q, want \"user_profiles\"", result.Tables[0])
	}
}
