package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunDBMigrateLint_RefusesUnguardedBulk(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tasks_schema.sql")
	sql := "CREATE TABLE a (id int);\nCREATE TABLE b (id int);\nCREATE TABLE c (id int);\n"
	if err := os.WriteFile(path, []byte(sql), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := &cobra.Command{}
	if err := runDBMigrateLint(cmd, []string{path}); err == nil {
		t.Fatal("expected lint to refuse (return an error) for the bulk-unguarded fixture")
	}
}

func TestRunDBMigrateLint_PassesGuarded(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "guarded.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE IF NOT EXISTS a (id int);\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := &cobra.Command{}
	if err := runDBMigrateLint(cmd, []string{path}); err != nil {
		t.Fatalf("expected lint to pass, got error: %v", err)
	}
}
