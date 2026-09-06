package nginxtopo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveFrontingDir_Confirmed proves the one structural convention this
// package trusts: projectDir living directly under a directory whose
// basename equals frontedBy resolves to that parent.
func TestResolveFrontingDir_Confirmed(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "nself-web")
	projectDir := filepath.Join(parent, "backend")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	dir, ok := ResolveFrontingDir(projectDir, "nself-web")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if dir != parent {
		t.Errorf("got %q, want %q", dir, parent)
	}
}

// TestResolveFrontingDir_MismatchedBasename proves resolution refuses
// (ok=false) rather than guessing when the parent's basename doesn't match.
func TestResolveFrontingDir_MismatchedBasename(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "some-other-name")
	projectDir := filepath.Join(parent, "backend")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, ok := ResolveFrontingDir(projectDir, "nself-web"); ok {
		t.Error("expected ok=false for a mismatched parent basename")
	}
}

// TestResolveFrontingDir_FilesystemRoot guards against an infinite loop or
// false positive when projectDir has no parent.
func TestResolveFrontingDir_FilesystemRoot(t *testing.T) {
	root := string(filepath.Separator)
	if _, ok := ResolveFrontingDir(root, "nself-web"); ok {
		t.Error("expected ok=false at filesystem root")
	}
}
