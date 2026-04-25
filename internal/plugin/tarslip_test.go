package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// buildTarGz creates an in-memory .tar.gz with the given entries.
// Each entry is (name string, content []byte). An empty content slice
// makes a directory entry.
func buildTarGz(t *testing.T, entries []struct{ name, content string }) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		if e.content == "" {
			hdr := &tar.Header{
				Typeflag: tar.TypeDir,
				Name:     e.name,
				Mode:     0o755,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatalf("WriteHeader dir %q: %v", e.name, err)
			}
		} else {
			hdr := &tar.Header{
				Typeflag: tar.TypeReg,
				Name:     e.name,
				Size:     int64(len(e.content)),
				Mode:     0o644,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatalf("WriteHeader file %q: %v", e.name, err)
			}
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatalf("Write file %q: %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func writeTempArchive(t *testing.T, data []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "archive-*.tar.gz")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		t.Fatalf("Write archive: %v", err)
	}
	return f.Name()
}

// TestExtractTarGz_HappyPath verifies normal extraction of a valid archive.
func TestExtractTarGz_HappyPath(t *testing.T) {
	entries := []struct{ name, content string }{
		{"plugin.json", `{"name":"test","version":"1.0.0"}`},
		{"lib/", ""},
		{"lib/helper.sh", "#!/bin/sh\necho ok"},
	}
	archivePath := writeTempArchive(t, buildTarGz(t, entries))
	destDir := t.TempDir()

	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("extractTarGz happy path: %v", err)
	}

	// Verify extracted files exist.
	wantFiles := []string{"plugin.json", "lib/helper.sh"}
	for _, rel := range wantFiles {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Errorf("expected extracted file %q: %v", rel, err)
		}
	}
}

// TestExtractTarGz_AbsolutePath ensures an archive containing an absolute
// path entry is rejected (tar slip prevention).
func TestExtractTarGz_AbsolutePath(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	// Write an entry with an absolute path directly into tar (bypassing normal helpers).
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "/etc/crontab",
		Size:     int64(len("evil")),
		Mode:     0o644,
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write([]byte("evil"))
	_ = tw.Close()
	_ = gw.Close()

	archivePath := writeTempArchive(t, buf.Bytes())
	destDir := t.TempDir()

	err := extractTarGz(archivePath, destDir)
	if err == nil {
		t.Fatal("expected error for absolute path tar entry, got nil")
	}
}

// TestExtractTarGz_DotDotTraversal ensures ../../../etc/passwd style paths
// are rejected.
func TestExtractTarGz_DotDotTraversal(t *testing.T) {
	cases := []struct {
		name    string
		tarName string
	}{
		{"double-dot prefix", "../../../etc/passwd"},
		{"dotdot after subdir", "subdir/../../.etc/secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			tw := tar.NewWriter(gw)
			hdr := &tar.Header{
				Typeflag: tar.TypeReg,
				Name:     tc.tarName,
				Size:     int64(len("payload")),
				Mode:     0o644,
			}
			_ = tw.WriteHeader(hdr)
			_, _ = tw.Write([]byte("payload"))
			_ = tw.Close()
			_ = gw.Close()

			archivePath := writeTempArchive(t, buf.Bytes())
			destDir := t.TempDir()

			err := extractTarGz(archivePath, destDir)
			if err == nil {
				t.Errorf("tc %q: expected path-traversal error, got nil", tc.name)
			}
		})
	}
}

// TestExtractTarGz_Symlink ensures symlink entries in the archive are rejected.
func TestExtractTarGz_Symlink(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Typeflag: tar.TypeSymlink,
		Name:     "link",
		Linkname: "/etc/passwd",
		Mode:     0o644,
	}
	_ = tw.WriteHeader(hdr)
	_ = tw.Close()
	_ = gw.Close()

	archivePath := writeTempArchive(t, buf.Bytes())
	destDir := t.TempDir()

	err := extractTarGz(archivePath, destDir)
	if err == nil {
		t.Fatal("expected error for symlink tar entry, got nil")
	}
}

// TestExtractTarGz_HardLink ensures hard-link entries in the archive are rejected.
func TestExtractTarGz_HardLink(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Typeflag: tar.TypeLink,
		Name:     "hardlink",
		Linkname: "/etc/passwd",
		Mode:     0o644,
	}
	_ = tw.WriteHeader(hdr)
	_ = tw.Close()
	_ = gw.Close()

	archivePath := writeTempArchive(t, buf.Bytes())
	destDir := t.TempDir()

	err := extractTarGz(archivePath, destDir)
	if err == nil {
		t.Fatal("expected error for hard-link tar entry, got nil")
	}
}

// TestExtractTarGz_SetuidBitStripped verifies setuid/setgid bits are stripped
// from extracted file permissions (privilege escalation prevention).
func TestExtractTarGz_SetuidBitStripped(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "suidfile",
		Size:     int64(len("data")),
		Mode:     0o4755, // setuid bit set
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write([]byte("data"))
	_ = tw.Close()
	_ = gw.Close()

	archivePath := writeTempArchive(t, buf.Bytes())
	destDir := t.TempDir()

	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	info, err := os.Stat(filepath.Join(destDir, "suidfile"))
	if err != nil {
		t.Fatalf("stat suidfile: %v", err)
	}
	if info.Mode()&os.ModeSetuid != 0 {
		t.Errorf("setuid bit not stripped from extracted file: mode=%v", info.Mode())
	}
}
