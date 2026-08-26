package access

// Purpose: a Transport backed by a plain local file, used two ways: (1) as
// the fixture every test in this package and cmd/commands/access_test.go
// drives, standing in for a remote authorized_keys file with no network
// involved; (2) as the real implementation when an operator points `nself
// access` at a local path directly (e.g. a mounted volume or a dry-run
// against a scratch copy).
// Inputs: a filesystem path.
// Outputs: file reads/writes/backups at that path.
// Constraints: Write always leaves the file at 0600; Backup never errors on
// a missing source file, since granting the very first key on a host is the
// common case.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LocalFileTransport implements Transport against a file on the local
// filesystem.
type LocalFileTransport struct {
	Path string

	// now is overridable in tests for deterministic backup filenames.
	now func() time.Time
}

// NewLocalFileTransport returns a Transport rooted at path.
func NewLocalFileTransport(path string) *LocalFileTransport {
	return &LocalFileTransport{Path: path, now: time.Now}
}

func (t *LocalFileTransport) Describe() string { return t.Path }

// Read returns the file's content, or (nil, nil) if it does not exist yet.
func (t *LocalFileTransport) Read(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(t.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", t.Path, err)
	}
	return data, nil
}

// Backup copies the current file to "<path>.bak.<UTC timestamp>" and returns
// that path, or "" if there was nothing to back up.
func (t *LocalFileTransport) Backup(ctx context.Context) (string, error) {
	data, err := os.ReadFile(t.Path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s for backup: %w", t.Path, err)
	}

	nowFn := t.now
	if nowFn == nil {
		nowFn = time.Now
	}
	backupPath := fmt.Sprintf("%s.bak.%s", t.Path, nowFn().UTC().Format("20060102T150405Z"))
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write backup %s: %w", backupPath, err)
	}
	return backupPath, nil
}

// Write replaces the file's content, creating its parent directory (0700) if
// needed, and leaves the file at 0600.
func (t *LocalFileTransport) Write(ctx context.Context, content []byte) error {
	dir := filepath.Dir(t.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(t.Path, content, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", t.Path, err)
	}
	// os.WriteFile applies the given mode only when creating the file; an
	// existing file keeps its prior permissions. Chmod explicitly so a
	// pre-existing 0644 authorized_keys is corrected on every write.
	return os.Chmod(t.Path, 0o600)
}
