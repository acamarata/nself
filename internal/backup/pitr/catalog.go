// Package pitr implements point-in-time recovery via pg_basebackup and WAL archiving.
//
// The catalog (~/.nself/wal-catalog.json) tracks base backup timestamps and WAL
// segment metadata. The recovery orchestrator uses the catalog to locate the
// correct base backup and WAL segments for a given target time.
package pitr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BaseBackup describes a pg_basebackup snapshot stored at a remote destination.
type BaseBackup struct {
	Timestamp time.Time `json:"timestamp"`
	RemoteKey string    `json:"remote_key"` // e.g. pitr/base/20260422T020000.tar.age
	SizeBytes int64     `json:"size_bytes,omitempty"`
}

// WALSegment describes a single archived WAL segment.
type WALSegment struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	RemoteKey string    `json:"remote_key"`
}

// Catalog is the on-disk PITR catalog stored at ~/.nself/wal-catalog.json.
type Catalog struct {
	BaseBackups []BaseBackup `json:"base_backups"`
	WALSegments []WALSegment `json:"wal_segments"`
}

// CatalogPath returns the default catalog path.
func CatalogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".nself", "wal-catalog.json"), nil
}

// LoadCatalog reads the catalog from disk. Returns an empty catalog if the file
// does not exist yet.
func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Catalog{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read catalog %s: %w", path, err)
	}

	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse catalog %s: %w", path, err)
	}
	return &c, nil
}

// Save persists the catalog to disk atomically (write to temp file, then rename).
func (c *Catalog) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create catalog directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write catalog temp file: %w", err)
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("chmod catalog: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename catalog: %w", err)
	}
	return nil
}

// AddBaseBackup appends a new base backup entry and saves the catalog.
func (c *Catalog) AddBaseBackup(path string, bb BaseBackup) error {
	c.BaseBackups = append(c.BaseBackups, bb)
	return c.Save(path)
}

// AddWALSegment appends a WAL segment entry and saves the catalog.
func (c *Catalog) AddWALSegment(path string, seg WALSegment) error {
	c.WALSegments = append(c.WALSegments, seg)
	return c.Save(path)
}

// LatestBaseBackupBefore returns the most recent base backup whose timestamp is
// at or before target. Returns an error if no suitable backup exists.
func (c *Catalog) LatestBaseBackupBefore(target time.Time) (BaseBackup, error) {
	var best BaseBackup
	found := false
	for _, bb := range c.BaseBackups {
		if !bb.Timestamp.After(target) {
			if !found || bb.Timestamp.After(best.Timestamp) {
				best = bb
				found = true
			}
		}
	}
	if !found {
		return BaseBackup{}, fmt.Errorf("no base backup found at or before %s", target.Format(time.RFC3339))
	}
	return best, nil
}

// WALSegmentsForRange returns WAL segments whose time range overlaps [start, end].
func (c *Catalog) WALSegmentsForRange(start, end time.Time) []WALSegment {
	var segs []WALSegment
	for _, s := range c.WALSegments {
		if s.End.After(start) && s.Start.Before(end) {
			segs = append(segs, s)
		}
	}
	return segs
}

// OldestRecoverableTime returns the start of the earliest WAL segment, or the
// earliest base backup timestamp — whichever is earlier.
func (c *Catalog) OldestRecoverableTime() (time.Time, bool) {
	var oldest time.Time
	found := false

	for _, bb := range c.BaseBackups {
		if !found || bb.Timestamp.Before(oldest) {
			oldest = bb.Timestamp
			found = true
		}
	}
	for _, seg := range c.WALSegments {
		if !found || seg.Start.Before(oldest) {
			oldest = seg.Start
			found = true
		}
	}
	return oldest, found
}

// NewestWALTime returns the end time of the latest WAL segment.
func (c *Catalog) NewestWALTime() (time.Time, bool) {
	var newest time.Time
	found := false
	for _, seg := range c.WALSegments {
		if !found || seg.End.After(newest) {
			newest = seg.End
			found = true
		}
	}
	return newest, found
}

// PruneOlderThan removes base backups and WAL segments older than the cutoff
// from the in-memory catalog. Call Save() after to persist.
func (c *Catalog) PruneOlderThan(cutoff time.Time) (pruned int) {
	var keptBBs []BaseBackup
	for _, bb := range c.BaseBackups {
		if bb.Timestamp.Before(cutoff) {
			pruned++
		} else {
			keptBBs = append(keptBBs, bb)
		}
	}
	c.BaseBackups = keptBBs

	var keptSegs []WALSegment
	for _, seg := range c.WALSegments {
		if seg.End.Before(cutoff) {
			pruned++
		} else {
			keptSegs = append(keptSegs, seg)
		}
	}
	c.WALSegments = keptSegs
	return pruned
}
