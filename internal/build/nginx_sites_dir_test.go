package build

// nginx_sites_dir_test.go — cli#385: proves resolveNginxSitesDir targets
// the fronting stack's own nginx/sites/ when NGINX_FRONTED_BY's structural
// convention can be confirmed, leaves the unfronted default untouched, and
// refuses (naming both directories) when the convention cannot be
// confirmed rather than silently writing a tree nginx never reads.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveNginxSitesDir_Unfronted proves the default (FrontedBy unset)
// case is unchanged: the project's own workdir/nginx/sites.
func TestResolveNginxSitesDir_Unfronted(t *testing.T) {
	workdir := t.TempDir()

	got, err := resolveNginxSitesDir(workdir, "")
	if err != nil {
		t.Fatalf("resolveNginxSitesDir: %v", err)
	}
	want := filepath.Join(workdir, "nginx", "sites")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestResolveNginxSitesDir_FrontedResolved proves that when this project
// lives as "backend" directly under the fronting stack's own directory
// (basename == FrontedBy), route confs are targeted at the fronting
// stack's own nginx/sites — the directory its nginx actually serves from —
// not this project's own (unread) tree.
func TestResolveNginxSitesDir_FrontedResolved(t *testing.T) {
	base := t.TempDir()
	frontingDir := filepath.Join(base, "nself-web")
	workdir := filepath.Join(frontingDir, "backend")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}

	got, err := resolveNginxSitesDir(workdir, "nself-web")
	if err != nil {
		t.Fatalf("resolveNginxSitesDir: %v", err)
	}
	want := filepath.Join(frontingDir, "nginx", "sites")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Must NOT be this project's own (unserved) directory.
	if got == filepath.Join(workdir, "nginx", "sites") {
		t.Error("resolved to this project's own nginx/sites — that tree is never read by any running nginx when fronted")
	}
}

// TestResolveNginxSitesDir_FrontedUnresolvedRefuses is the refusal path:
// FrontedBy is set but the parent directory's basename does not match it,
// so the fronting stack's directory cannot be confirmed. The build must
// refuse, and the error must name BOTH this project's own directory and
// the (mismatched) parent directory that was actually checked, so an
// operator can act on the message without re-deriving the topology.
func TestResolveNginxSitesDir_FrontedUnresolvedRefuses(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "some-other-name")
	workdir := filepath.Join(parent, "backend")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}

	_, err := resolveNginxSitesDir(workdir, "nself-web")
	if err == nil {
		t.Fatal("expected an error when the fronting stack's directory cannot be confirmed, got nil")
	}

	ownDir := filepath.Join(workdir, "nginx", "sites")
	msg := err.Error()
	if !strings.Contains(msg, ownDir) {
		t.Errorf("error must name this project's own directory %q, got: %s", ownDir, msg)
	}
	if !strings.Contains(msg, parent) {
		t.Errorf("error must name the checked (mismatched) parent directory %q, got: %s", parent, msg)
	}
	if !strings.Contains(msg, "nself-web") {
		t.Errorf("error must name the NGINX_FRONTED_BY stack, got: %s", msg)
	}
}

// TestResolveNginxSitesDir_FrontedAtFilesystemRoot guards the edge case
// where workdir has no parent (filepath.Dir returns itself) — must refuse,
// never loop or panic.
func TestResolveNginxSitesDir_FrontedAtFilesystemRoot(t *testing.T) {
	root := string(filepath.Separator)
	if _, err := resolveNginxSitesDir(root, "nself-web"); err == nil {
		t.Error("expected an error resolving a fronting dir at filesystem root, got nil")
	}
}
