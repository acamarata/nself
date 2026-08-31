package pitr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PruneOptions holds configuration for the retention pruning pass.
type PruneOptions struct {
	// RetentionDays is the number of days to retain WAL segments and base
	// backups. Segments older than Now()-RetentionDays are deleted.
	RetentionDays int

	// RemoteDeleteFn is called for each remote key that should be deleted.
	// Implementations must be idempotent (safe to call on already-deleted keys).
	// If nil, only the catalog is updated (no remote deletion).
	RemoteDeleteFn func(ctx context.Context, key string) error

	// LocalBaseBackupDir is the directory where local base backup archives are
	// stored. Files in this directory whose names match pruned entries are
	// removed. Set to "" to skip local file deletion.
	LocalBaseBackupDir string
}

// PruneRetention removes catalog entries and (optionally) remote/local files
// older than the configured retention window. Returns the number of entries pruned.
func PruneRetention(ctx context.Context, catalogPath string, opts PruneOptions) (int, error) {
	if opts.RetentionDays <= 0 {
		opts.RetentionDays = 7
	}

	catalog, err := LoadCatalog(catalogPath)
	if err != nil {
		return 0, fmt.Errorf("load catalog for pruning: %w", err)
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -opts.RetentionDays)

	// Collect remote keys to delete before modifying the catalog.
	var toDeleteRemote []string
	var toDeleteLocal []string

	for _, bb := range catalog.BaseBackups {
		if bb.Timestamp.Before(cutoff) {
			if bb.RemoteKey != "" {
				toDeleteRemote = append(toDeleteRemote, bb.RemoteKey)
			}
			if opts.LocalBaseBackupDir != "" && bb.RemoteKey != "" {
				// The remote key basename may differ from local path; keep it simple.
				toDeleteLocal = append(toDeleteLocal, bb.RemoteKey)
			}
		}
	}
	for _, seg := range catalog.WALSegments {
		if seg.End.Before(cutoff) && seg.RemoteKey != "" {
			toDeleteRemote = append(toDeleteRemote, seg.RemoteKey)
		}
	}

	pruned := catalog.PruneOlderThan(cutoff)

	// Persist catalog before attempting deletions — if a deletion fails the
	// catalog is still updated so the next prune does not re-add the entry.
	if err := catalog.Save(catalogPath); err != nil {
		return pruned, fmt.Errorf("save catalog after pruning: %w", err)
	}

	// Delete remote objects.
	if opts.RemoteDeleteFn != nil {
		for _, key := range toDeleteRemote {
			if delErr := opts.RemoteDeleteFn(ctx, key); delErr != nil {
				// Log but don't abort — partial success is better than none.
				fmt.Fprintf(os.Stderr, "warning: failed to delete remote segment %s: %v\n", key, delErr)
			}
		}
	}

	// Delete local base-backup archives.
	//
	// This loop was missing: toDeleteLocal was populated and then never read,
	// so LocalBaseBackupDir was a documented option that did nothing and local
	// base backups grew without bound. Found by staticcheck SA4010.
	//
	// Only the basename is used, because a remote key carries prefixes that do
	// not exist on disk. deleteLocalArchive refuses anything that would escape
	// LocalBaseBackupDir, and a file that is already gone is not an error —
	// pruning must stay idempotent, matching RemoteDeleteFn's contract.
	for _, key := range toDeleteLocal {
		if delErr := deleteLocalArchive(opts.LocalBaseBackupDir, key); delErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete local base backup %s: %v\n", key, delErr)
		}
	}

	return pruned, nil
}

// deleteLocalArchive removes one pruned base-backup archive from dir.
//
// Purpose: delete the on-disk counterpart of a pruned catalog entry.
// Inputs:  dir, the configured LocalBaseBackupDir; key, the entry's remote key.
// Outputs: nil when the file is deleted or already absent; an error otherwise.
// Constraints: only the basename of key is honoured, and the resolved path must
//
//	stay inside dir — a key is catalog data, so it is not trusted to
//	be a safe relative path. An empty dir disables deletion.
func deleteLocalArchive(dir, key string) error {
	if dir == "" {
		return nil
	}
	base := filepath.Base(key)
	if base == "." || base == string(filepath.Separator) || base == ".." {
		return fmt.Errorf("refusing to delete unsafe archive name %q", key)
	}
	path := filepath.Join(dir, base)

	cleanDir := filepath.Clean(dir) + string(filepath.Separator)
	if !strings.HasPrefix(filepath.Clean(path)+string(filepath.Separator), cleanDir) {
		return fmt.Errorf("refusing to delete %q outside %q", path, dir)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
