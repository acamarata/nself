package pitr

// retention_local_test.go — Coverage for local base-backup deletion.
//
// Purpose: PruneRetention documented LocalBaseBackupDir as "files in this
//          directory whose names match pruned entries are removed", but the
//          list of files to delete was built and then never read. The option
//          did nothing and local base backups grew without bound. staticcheck
//          SA4010 ("this result of append is never used") was the only thing
//          reporting it. These tests pin the behavior so it cannot silently
//          regress to a no-op again.
// Inputs:  a temp dir standing in for LocalBaseBackupDir; remote-key strings.
// Outputs: assertions on which files survive.
// Constraints: filesystem only — no catalog, no network.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteLocalArchive_RemovesByBasename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "base-20260801.tar.gz")
	if err := os.WriteFile(path, []byte("archive"), 0o600); err != nil {
		t.Fatal(err)
	}

	// A remote key carries prefixes that do not exist on disk; only the
	// basename should be used to locate the local file.
	if err := deleteLocalArchive(dir, "pitr/base/base-20260801.tar.gz"); err != nil {
		t.Fatalf("deleteLocalArchive: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("archive still present after prune; LocalBaseBackupDir is a no-op again")
	}
}

// TestDeleteLocalArchive_MissingFileIsNotAnError pins idempotency, matching the
// contract RemoteDeleteFn already documents. A prune that errors on an
// already-deleted file cannot be safely re-run.
func TestDeleteLocalArchive_MissingFileIsNotAnError(t *testing.T) {
	if err := deleteLocalArchive(t.TempDir(), "never-existed.tar.gz"); err != nil {
		t.Errorf("missing file returned %v, want nil", err)
	}
}

func TestDeleteLocalArchive_EmptyDirDisablesDeletion(t *testing.T) {
	if err := deleteLocalArchive("", "anything.tar.gz"); err != nil {
		t.Errorf("empty dir returned %v, want nil", err)
	}
}

// TestDeleteLocalArchive_RefusesEscapingPath matters because a key comes from
// the catalog file, which is data on disk rather than a trusted constant. A
// crafted key must not be able to delete outside the configured directory.
func TestDeleteLocalArchive_RefusesEscapingPath(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(filepath.Dir(dir), "outside.tar.gz")
	if err := os.WriteFile(outside, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(outside) }()

	for _, key := range []string{"../outside.tar.gz", "../../outside.tar.gz", ".."} {
		_ = deleteLocalArchive(dir, key)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("a key escaped LocalBaseBackupDir and deleted %s", outside)
	}
}
