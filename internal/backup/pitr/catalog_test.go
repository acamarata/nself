package pitr

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal-catalog.json")

	c := &Catalog{}

	ts := time.Date(2026, 4, 22, 2, 0, 0, 0, time.UTC)
	bb := BaseBackup{
		Timestamp: ts,
		RemoteKey: "pitr/base/20260422T020000.tar.age",
	}
	if err := c.AddBaseBackup(path, bb); err != nil {
		t.Fatalf("AddBaseBackup: %v", err)
	}

	loaded, err := LoadCatalog(path)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(loaded.BaseBackups) != 1 {
		t.Fatalf("expected 1 base backup, got %d", len(loaded.BaseBackups))
	}
	if !loaded.BaseBackups[0].Timestamp.Equal(ts) {
		t.Errorf("timestamp mismatch: got %v, want %v", loaded.BaseBackups[0].Timestamp, ts)
	}
	if loaded.BaseBackups[0].RemoteKey != bb.RemoteKey {
		t.Errorf("remote key mismatch: got %q, want %q", loaded.BaseBackups[0].RemoteKey, bb.RemoteKey)
	}
}

func TestLoadCatalogMissing(t *testing.T) {
	c, err := LoadCatalog("/nonexistent/path/wal-catalog.json")
	if err != nil {
		t.Fatalf("expected no error for missing catalog, got: %v", err)
	}
	if len(c.BaseBackups) != 0 || len(c.WALSegments) != 0 {
		t.Errorf("expected empty catalog for missing file")
	}
}

func TestLatestBaseBackupBefore(t *testing.T) {
	c := &Catalog{
		BaseBackups: []BaseBackup{
			{Timestamp: time.Date(2026, 4, 20, 2, 0, 0, 0, time.UTC), RemoteKey: "base/old"},
			{Timestamp: time.Date(2026, 4, 22, 2, 0, 0, 0, time.UTC), RemoteKey: "base/new"},
			{Timestamp: time.Date(2026, 4, 23, 2, 0, 0, 0, time.UTC), RemoteKey: "base/future"},
		},
	}

	target := time.Date(2026, 4, 22, 15, 30, 0, 0, time.UTC)
	bb, err := c.LatestBaseBackupBefore(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bb.RemoteKey != "base/new" {
		t.Errorf("expected base/new, got %q", bb.RemoteKey)
	}
}

func TestLatestBaseBackupBeforeNoMatch(t *testing.T) {
	c := &Catalog{
		BaseBackups: []BaseBackup{
			{Timestamp: time.Date(2026, 4, 23, 2, 0, 0, 0, time.UTC), RemoteKey: "base/future"},
		},
	}

	target := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	_, err := c.LatestBaseBackupBefore(target)
	if err == nil {
		t.Fatal("expected error when no base backup exists before target")
	}
}

func TestOldestNewestTimes(t *testing.T) {
	c := &Catalog{
		WALSegments: []WALSegment{
			{
				Start:     time.Date(2026, 4, 22, 2, 0, 0, 0, time.UTC),
				End:       time.Date(2026, 4, 22, 2, 0, 5, 0, time.UTC),
				RemoteKey: "wal/seg1",
			},
			{
				Start:     time.Date(2026, 4, 22, 3, 0, 0, 0, time.UTC),
				End:       time.Date(2026, 4, 22, 3, 0, 5, 0, time.UTC),
				RemoteKey: "wal/seg2",
			},
		},
	}

	oldest, ok := c.OldestRecoverableTime()
	if !ok {
		t.Fatal("expected oldest recoverable time")
	}
	if !oldest.Equal(time.Date(2026, 4, 22, 2, 0, 0, 0, time.UTC)) {
		t.Errorf("oldest time mismatch: %v", oldest)
	}

	newest, ok := c.NewestWALTime()
	if !ok {
		t.Fatal("expected newest WAL time")
	}
	if !newest.Equal(time.Date(2026, 4, 22, 3, 0, 5, 0, time.UTC)) {
		t.Errorf("newest time mismatch: %v", newest)
	}
}

func TestPruneOlderThan(t *testing.T) {
	c := &Catalog{
		BaseBackups: []BaseBackup{
			{Timestamp: time.Date(2026, 4, 15, 2, 0, 0, 0, time.UTC), RemoteKey: "base/old"},
			{Timestamp: time.Date(2026, 4, 22, 2, 0, 0, 0, time.UTC), RemoteKey: "base/new"},
		},
		WALSegments: []WALSegment{
			{
				Start:     time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC),
				End:       time.Date(2026, 4, 14, 1, 0, 0, 0, time.UTC),
				RemoteKey: "wal/old",
			},
			{
				Start:     time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
				End:       time.Date(2026, 4, 22, 1, 0, 0, 0, time.UTC),
				RemoteKey: "wal/new",
			},
		},
	}

	cutoff := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	pruned := c.PruneOlderThan(cutoff)

	if pruned != 2 {
		t.Errorf("expected 2 pruned entries, got %d", pruned)
	}
	if len(c.BaseBackups) != 1 {
		t.Errorf("expected 1 base backup remaining, got %d", len(c.BaseBackups))
	}
	if len(c.WALSegments) != 1 {
		t.Errorf("expected 1 WAL segment remaining, got %d", len(c.WALSegments))
	}
}

func TestCatalogAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "wal-catalog.json")

	c := &Catalog{}
	bb := BaseBackup{
		Timestamp: time.Now().UTC(),
		RemoteKey: "pitr/base/test.tar",
	}
	if err := c.AddBaseBackup(path, bb); err != nil {
		t.Fatalf("AddBaseBackup in new directory: %v", err)
	}

	// Verify the temp file was removed.
	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful write")
	}

	// Verify the main file has correct perms.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat catalog: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %04o", info.Mode().Perm())
	}
}
