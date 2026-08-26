package license

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureDir_MovesAsideLegacyFile pins the upgrade path that was broken.
//
// An older nSelf stored the license as a regular file at ~/.nself/license.
// Every current license operation calls MkdirAll on that same path, which
// returns ENOTDIR against a file, so on an upgraded machine `nself license set`
// failed with "remove .../license/key: not a directory" and no way forward.
func TestEnsureDir_MovesAsideLegacyFile(t *testing.T) {
	root := t.TempDir()
	// Must be the real license path: the migration is scoped to it on purpose,
	// so that an arbitrary non-directory elsewhere is reported rather than
	// silently renamed.
	dir := filepath.Join(root, licenseDir)
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		t.Fatalf("seeding parent: %v", err)
	}

	const legacy = "legacy license payload"
	if err := os.WriteFile(dir, []byte(legacy), 0600); err != nil {
		t.Fatalf("seeding legacy file: %v", err)
	}

	if err := ensureDir(dir); err != nil {
		t.Fatalf("ensureDir over a legacy file: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat after ensureDir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("path is still not a directory")
	}

	// The legacy file must survive: it is the user's license data.
	entries, err := os.ReadDir(filepath.Dir(dir))
	if err != nil {
		t.Fatalf("reading root: %v", err)
	}
	var kept string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "license.legacy-") {
			kept = filepath.Join(filepath.Dir(dir), e.Name())
		}
	}
	if kept == "" {
		t.Fatal("legacy file was not preserved: it holds the user's license and must never be deleted")
	}
	data, err := os.ReadFile(kept)
	if err != nil || string(data) != legacy {
		t.Errorf("preserved file does not hold the original contents: %q, err=%v", data, err)
	}
}

// TestEnsureDir_NormalCases covers the paths that must keep behaving as MkdirAll.
func TestEnsureDir_NormalCases(t *testing.T) {
	root := t.TempDir()

	fresh := filepath.Join(root, "a", "b")
	if err := ensureDir(fresh); err != nil {
		t.Fatalf("creating a fresh nested dir: %v", err)
	}
	if info, err := os.Stat(fresh); err != nil || !info.IsDir() {
		t.Fatalf("fresh dir not created: err=%v", err)
	}

	// Idempotent: an existing directory is left exactly as it is.
	marker := filepath.Join(fresh, "keep")
	if err := os.WriteFile(marker, []byte("x"), 0600); err != nil {
		t.Fatalf("writing marker: %v", err)
	}
	if err := ensureDir(fresh); err != nil {
		t.Fatalf("ensureDir on an existing dir: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("existing directory contents were disturbed: %v", err)
	}
}

// TestEnsureDir_RefusesToMoveUnrelatedFile pins the scoping decision.
//
// Callers pass directories other than the license dir here, including
// $HOME/.nself. Renaming whatever happens to sit at an arbitrary path would be
// more destructive than the bug the migration fixes, so anything that is not
// the known legacy license file must be reported instead of moved.
func TestEnsureDir_RefusesToMoveUnrelatedFile(t *testing.T) {
	root := t.TempDir()
	other := filepath.Join(root, ".nself")
	if err := os.WriteFile(other, []byte("not a license"), 0600); err != nil {
		t.Fatalf("seeding: %v", err)
	}

	err := ensureDir(other)
	if err == nil {
		t.Fatal("ensureDir silently moved a file it has no business moving")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error should say the path is not a directory, got: %v", err)
	}
	if _, statErr := os.Stat(other); statErr != nil {
		t.Errorf("the unrelated file was moved or removed: %v", statErr)
	}
}
