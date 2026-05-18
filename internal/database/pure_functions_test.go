package database

import (
	"strings"
	"testing"
)

// TestChecksumBytes_KnownInput verifies SHA-256 output for a known payload.
func TestChecksumBytes_KnownInput(t *testing.T) {
	// echo -n "" | sha256sum -> e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	got, err := checksumBytes([]byte{})
	if err != nil {
		t.Fatalf("checksumBytes(empty): unexpected error: %v", err)
	}
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("checksumBytes(empty) = %q, want %q", got, want)
	}
}

// TestChecksumBytes_NonEmpty verifies a non-empty payload produces a 64-char hex digest.
func TestChecksumBytes_NonEmpty(t *testing.T) {
	got, err := checksumBytes([]byte("hello, nself"))
	if err != nil {
		t.Fatalf("checksumBytes: unexpected error: %v", err)
	}
	if len(got) != 64 {
		t.Errorf("checksumBytes: got len=%d, want 64", len(got))
	}
	if !sha256HexRegex.MatchString(got) {
		t.Errorf("checksumBytes: %q does not match sha256HexRegex", got)
	}
}

// TestChecksumBytes_Deterministic verifies two calls produce the same output.
func TestChecksumBytes_Deterministic(t *testing.T) {
	data := []byte("migration-20260101-nself")
	a, _ := checksumBytes(data)
	b, _ := checksumBytes(data)
	if a != b {
		t.Errorf("checksumBytes is non-deterministic: %q != %q", a, b)
	}
}

// TestExtractMigrationID_FlatLayout verifies YYYYMMDDHHMMSS_name.sql parsing.
func TestExtractMigrationID_FlatLayout(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"migrations/20260101000000_create_users.sql", "20260101000000"},
		{"migrations/20251231235959_add_index.sql", "20251231235959"},
		{"migrations/20260115120000_init.sql", "20260115120000"},
		{"/absolute/path/20260501083000_plugin_schema.sql", "20260501083000"},
	}
	for _, tc := range cases {
		got := extractMigrationID(tc.path)
		if got != tc.want {
			t.Errorf("extractMigrationID(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestExtractMigrationID_NestedLayout verifies YYYYMMDDHHMMSS_name/up.sql parsing.
func TestExtractMigrationID_NestedLayout(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"migrations/20260101000000_create_users/up.sql", "20260101000000"},
		{"migrations/20260515090000_add_column/up.sql", "20260515090000"},
	}
	for _, tc := range cases {
		got := extractMigrationID(tc.path)
		if got != tc.want {
			t.Errorf("extractMigrationID(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestGenerateSoftDeleteMigration_UpSQL verifies the UP migration SQL is correct.
func TestGenerateSoftDeleteMigration_UpSQL(t *testing.T) {
	upSQL, _ := GenerateSoftDeleteMigration("np_test", "users")
	if !strings.Contains(upSQL, "ALTER TABLE np_test.users ADD COLUMN IF NOT EXISTS deleted_at") {
		t.Errorf("upSQL missing ADD COLUMN deleted_at: %q", upSQL)
	}
	if !strings.Contains(upSQL, "CREATE INDEX IF NOT EXISTS idx_users_deleted_at") {
		t.Errorf("upSQL missing CREATE INDEX: %q", upSQL)
	}
	if !strings.Contains(upSQL, "CREATE OR REPLACE VIEW np_test.users_active") {
		t.Errorf("upSQL missing CREATE VIEW users_active: %q", upSQL)
	}
	if !strings.Contains(upSQL, "WHERE deleted_at IS NULL") {
		t.Errorf("upSQL missing WHERE deleted_at IS NULL: %q", upSQL)
	}
}

// TestGenerateSoftDeleteMigration_DownSQL verifies the DOWN migration SQL reverses the UP.
func TestGenerateSoftDeleteMigration_DownSQL(t *testing.T) {
	_, downSQL := GenerateSoftDeleteMigration("np_test", "users")
	if !strings.Contains(downSQL, "DROP VIEW IF EXISTS np_test.users_active") {
		t.Errorf("downSQL missing DROP VIEW: %q", downSQL)
	}
	if !strings.Contains(downSQL, "DROP INDEX IF EXISTS") {
		t.Errorf("downSQL missing DROP INDEX: %q", downSQL)
	}
	if !strings.Contains(downSQL, "ALTER TABLE np_test.users DROP COLUMN IF EXISTS deleted_at") {
		t.Errorf("downSQL missing DROP COLUMN: %q", downSQL)
	}
}

// TestGenerateSoftDeleteMigration_DifferentSchemas verifies schema/table substitution.
func TestGenerateSoftDeleteMigration_DifferentSchemas(t *testing.T) {
	cases := []struct {
		schema string
		table  string
	}{
		{"np_chat", "messages"},
		{"np_claw", "notes"},
		{"public", "sessions"},
	}
	for _, tc := range cases {
		up, down := GenerateSoftDeleteMigration(tc.schema, tc.table)
		if !strings.Contains(up, tc.schema+"."+tc.table) {
			t.Errorf("upSQL for %s.%s missing correct schema.table reference", tc.schema, tc.table)
		}
		if !strings.Contains(down, tc.schema+"."+tc.table) {
			t.Errorf("downSQL for %s.%s missing correct schema.table reference", tc.schema, tc.table)
		}
	}
}

// TestGenerateFKIndexSQL_Format verifies the generated index SQL format.
func TestGenerateFKIndexSQL_Format(t *testing.T) {
	info := FKIndexInfo{
		TableSchema: "np_test",
		TableName:   "orders",
		ColumnName:  "user_id",
		HasIndex:    false,
	}
	got := GenerateFKIndexSQL(info)
	if !strings.Contains(got, "CREATE INDEX IF NOT EXISTS idx_orders_user_id") {
		t.Errorf("GenerateFKIndexSQL: missing index name: %q", got)
	}
	if !strings.Contains(got, "ON np_test.orders (user_id)") {
		t.Errorf("GenerateFKIndexSQL: missing ON clause: %q", got)
	}
}

// TestGenerateFKIndexSQL_NamingConvention verifies idx_{table}_{column} naming.
func TestGenerateFKIndexSQL_NamingConvention(t *testing.T) {
	cases := []struct {
		schema  string
		table   string
		column  string
		wantIdx string
	}{
		{"np_chat", "messages", "user_id", "idx_messages_user_id"},
		{"np_claw", "notes", "folder_id", "idx_notes_folder_id"},
		{"public", "sessions", "account_id", "idx_sessions_account_id"},
	}
	for _, tc := range cases {
		info := FKIndexInfo{TableSchema: tc.schema, TableName: tc.table, ColumnName: tc.column}
		got := GenerateFKIndexSQL(info)
		if !strings.Contains(got, tc.wantIdx) {
			t.Errorf("GenerateFKIndexSQL: index name %q not in %q", tc.wantIdx, got)
		}
	}
}

// TestGenerateDriftMigration_MissingColumns verifies UP SQL is generated for each column.
func TestGenerateDriftMigration_MissingColumns(t *testing.T) {
	result := SchemaDriftResult{
		TableSchema: "np_test",
		TableName:   "docs",
		MissingColumns: []RequiredColumn{
			{Name: "id", Required: true},
			{Name: "created_at", Required: true},
			{Name: "updated_at", Required: true},
		},
	}
	upSQL, downSQL := GenerateDriftMigration(result)
	if !strings.Contains(upSQL, "ADD COLUMN IF NOT EXISTS id UUID") {
		t.Errorf("upSQL missing id column: %q", upSQL)
	}
	if !strings.Contains(upSQL, "ADD COLUMN IF NOT EXISTS created_at") {
		t.Errorf("upSQL missing created_at column: %q", upSQL)
	}
	if !strings.Contains(upSQL, "ADD COLUMN IF NOT EXISTS updated_at") {
		t.Errorf("upSQL missing updated_at column: %q", upSQL)
	}
	if !strings.Contains(downSQL, "DROP COLUMN IF EXISTS id") {
		t.Errorf("downSQL missing DROP id: %q", downSQL)
	}
}

// TestGenerateDriftMigration_EmptyMissingColumns verifies empty input produces empty output.
func TestGenerateDriftMigration_EmptyMissingColumns(t *testing.T) {
	result := SchemaDriftResult{
		TableSchema:    "np_test",
		TableName:      "fully_compliant",
		MissingColumns: nil,
	}
	upSQL, downSQL := GenerateDriftMigration(result)
	if upSQL != "" {
		t.Errorf("GenerateDriftMigration(no missing): upSQL = %q, want empty", upSQL)
	}
	if downSQL != "" {
		t.Errorf("GenerateDriftMigration(no missing): downSQL = %q, want empty", downSQL)
	}
}

// TestSummarizeDrift_AllCompliant verifies a fully compliant table set.
func TestSummarizeDrift_AllCompliant(t *testing.T) {
	results := []SchemaDriftResult{
		{TableSchema: "np_test", TableName: "users", MissingColumns: nil},
		{TableSchema: "np_test", TableName: "orders", MissingColumns: nil},
	}
	summary := SummarizeDrift(results)
	if summary.TotalTables != 2 {
		t.Errorf("TotalTables: got %d, want 2", summary.TotalTables)
	}
	if summary.CompliantTables != 2 {
		t.Errorf("CompliantTables: got %d, want 2", summary.CompliantTables)
	}
	if summary.DriftedTables != 0 {
		t.Errorf("DriftedTables: got %d, want 0", summary.DriftedTables)
	}
	if summary.OverallScore != 100 {
		t.Errorf("OverallScore: got %d, want 100", summary.OverallScore)
	}
}

// TestSummarizeDrift_AllDrifted verifies a fully drifted table set.
func TestSummarizeDrift_AllDrifted(t *testing.T) {
	results := []SchemaDriftResult{
		{
			TableSchema:    "np_test",
			TableName:      "old_table",
			MissingColumns: []RequiredColumn{{Name: "id", Required: true}},
		},
		{
			TableSchema:    "np_test",
			TableName:      "legacy",
			MissingColumns: []RequiredColumn{{Name: "created_at", Required: false}},
		},
	}
	summary := SummarizeDrift(results)
	if summary.DriftedTables != 2 {
		t.Errorf("DriftedTables: got %d, want 2", summary.DriftedTables)
	}
	if summary.CompliantTables != 0 {
		t.Errorf("CompliantTables: got %d, want 0", summary.CompliantTables)
	}
	if summary.OverallScore != 0 {
		t.Errorf("OverallScore: got %d, want 0", summary.OverallScore)
	}
	// Only the table with Required:true increments MissingRequired.
	if summary.MissingRequired != 1 {
		t.Errorf("MissingRequired: got %d, want 1", summary.MissingRequired)
	}
}

// TestSummarizeDrift_Mixed verifies partial drift.
func TestSummarizeDrift_Mixed(t *testing.T) {
	results := []SchemaDriftResult{
		{TableSchema: "np_test", TableName: "compliant", MissingColumns: nil},
		{
			TableSchema:    "np_test",
			TableName:      "drifted",
			MissingColumns: []RequiredColumn{{Name: "updated_at", Required: true}},
		},
	}
	summary := SummarizeDrift(results)
	if summary.TotalTables != 2 {
		t.Errorf("TotalTables: got %d, want 2", summary.TotalTables)
	}
	if summary.CompliantTables != 1 {
		t.Errorf("CompliantTables: got %d, want 1", summary.CompliantTables)
	}
	if summary.DriftedTables != 1 {
		t.Errorf("DriftedTables: got %d, want 1", summary.DriftedTables)
	}
	// 1 of 2 tables drifted → score = 100 - 50 = 50
	if summary.OverallScore != 50 {
		t.Errorf("OverallScore: got %d, want 50", summary.OverallScore)
	}
}

// TestSummarizeDrift_Empty verifies zero-table input.
func TestSummarizeDrift_Empty(t *testing.T) {
	summary := SummarizeDrift(nil)
	if summary.TotalTables != 0 {
		t.Errorf("TotalTables: got %d, want 0", summary.TotalTables)
	}
	if summary.OverallScore != 100 {
		t.Errorf("OverallScore: got %d, want 100 (no tables = no drift)", summary.OverallScore)
	}
}
