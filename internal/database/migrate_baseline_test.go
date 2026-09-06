package database

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveBaselineFiles(t *testing.T) {
	files := []string{
		filepath.Join("migrations", "20260101_a.sql"),
		filepath.Join("migrations", "20260102_b.sql"),
	}

	resolved, err := resolveBaselineFiles(files, []string{"20260102_b.sql"})
	if err != nil {
		t.Fatalf("resolveBaselineFiles: %v", err)
	}
	if len(resolved) != 1 || resolved[0] != files[1] {
		t.Fatalf("resolved = %v, want [%s]", resolved, files[1])
	}
}

func TestResolveBaselineFiles_UnmatchedNameErrors(t *testing.T) {
	files := []string{filepath.Join("migrations", "20260101_a.sql")}
	if _, err := resolveBaselineFiles(files, []string{"does_not_exist.sql"}); err == nil {
		t.Fatal("expected an error for an unmatched migration name")
	}
}

func TestBuildBaselinePlans(t *testing.T) {
	tmp := t.TempDir()
	withCwd(t, tmp)
	writeFile(t, tmp, "migrations/20260101_a.sql", "CREATE TABLE a (id int);\n")
	writeFile(t, tmp, "migrations/20260102_b.sql", "CREATE TABLE b (id int);\n")

	files := []string{
		filepath.Join(tmp, "migrations", "20260101_a.sql"),
		filepath.Join(tmp, "migrations", "20260102_b.sql"),
	}

	applied := map[string]time.Time{"20260101_a.sql": time.Now()}
	plans, err := buildBaselinePlans(files, applied)
	if err != nil {
		t.Fatalf("buildBaselinePlans: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("len(plans) = %d, want 2", len(plans))
	}
	if !plans[0].AlreadyApplied {
		t.Error("plans[0] (20260101_a.sql) should be AlreadyApplied")
	}
	if plans[1].AlreadyApplied {
		t.Error("plans[1] (20260102_b.sql) should NOT be AlreadyApplied")
	}
	if plans[1].Checksum == "" {
		t.Error("expected a non-empty checksum")
	}
	if plans[1].MigrationID != "20260102_b" {
		t.Errorf("MigrationID = %q, want 20260102_b", plans[1].MigrationID)
	}
}

// TestBaselineTxSQL_NeverExecutesMigrationBody pins down what "record without
// executing" means concretely: the transaction BaselineMigration sends
// contains only the two ledger INSERTs and can never contain the migration
// file's own DDL, because that DDL is never passed to baselineTxSQL at all.
func TestBaselineTxSQL_NeverExecutesMigrationBody(t *testing.T) {
	txSQL := baselineTxSQL("20260102_b", "20260102_b.sql", "deadbeef")

	if !strings.Contains(txSQL, "INSERT INTO np_common.schema_versions") {
		t.Error("expected the legacy ledger INSERT")
	}
	if !strings.Contains(txSQL, "INSERT INTO nself_ops.migrations") {
		t.Error("expected the ops ledger INSERT")
	}
	if strings.Contains(strings.ToUpper(txSQL), "CREATE TABLE") {
		t.Error("a baseline transaction must never contain migration DDL")
	}
	if !strings.HasPrefix(txSQL, "BEGIN;\n") || !strings.HasSuffix(txSQL, "COMMIT;\n") {
		t.Errorf("expected a single wrapping transaction, got %q", txSQL)
	}
}
