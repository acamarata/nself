package access

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalFileTransport_ReadMissingFile(t *testing.T) {
	tp := NewLocalFileTransport(filepath.Join(t.TempDir(), "authorized_keys"))
	data, err := tp.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data for a missing file, got %q", data)
	}
}

func TestLocalFileTransport_BackupMissingFileIsNoop(t *testing.T) {
	tp := NewLocalFileTransport(filepath.Join(t.TempDir(), "authorized_keys"))
	path, err := tp.Backup(context.Background())
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if path != "" {
		t.Errorf("expected no backup path for a missing source file, got %q", path)
	}
}

func TestLocalFileTransport_WriteCreatesParentAnd0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "authorized_keys")
	tp := NewLocalFileTransport(path)

	if err := tp.Write(context.Background(), []byte("ssh-ed25519 AAAA test\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Windows has no Unix permission bits: Mode().Perm() always reports 0666
	// there regardless of what was requested. Only assert the real invariant
	// on platforms that have one.
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("mode = %o, want 0600", perm)
		}
	}
}

func TestLocalFileTransport_BackupThenWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authorized_keys")
	tp := NewLocalFileTransport(path)
	ctx := context.Background()

	original := "ssh-ed25519 AAAA original\n"
	if err := tp.Write(ctx, []byte(original)); err != nil {
		t.Fatalf("initial write: %v", err)
	}

	backupPath, err := tp.Backup(ctx)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if backupPath == "" {
		t.Fatal("expected a non-empty backup path")
	}
	if !strings.HasPrefix(backupPath, path+".bak.") {
		t.Errorf("backupPath = %q, want prefix %q", backupPath, path+".bak.")
	}

	backedUp, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backedUp) != original {
		t.Errorf("backup content = %q, want %q", backedUp, original)
	}

	// Writing new content must not disturb the backup already taken.
	if err := tp.Write(ctx, []byte("ssh-ed25519 AAAA replaced\n")); err != nil {
		t.Fatalf("second write: %v", err)
	}
	stillThere, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("re-read backup: %v", err)
	}
	if string(stillThere) != original {
		t.Error("backup content changed after a later write to the live file")
	}
}

func TestLocalFileTransport_Describe(t *testing.T) {
	tp := NewLocalFileTransport("/tmp/fixture/authorized_keys")
	if tp.Describe() != "/tmp/fixture/authorized_keys" {
		t.Errorf("Describe() = %q", tp.Describe())
	}
}
