package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindNSelfRoot_AcceptsPerEnvironmentFiles is the regression guard for the
// third finding of the ntask clean-fork self-host drill (2026-08-24): a repo
// that commits .env.dev and no bare .env is a complete project, but the CLI
// only recognised `.env` and told the user to run `nself init` — which would
// have overwritten a working configuration.
func TestFindNSelfRoot_AcceptsPerEnvironmentFiles(t *testing.T) {
	for _, marker := range []string{".env", ".env.dev", ".env.staging", ".env.prod"} {
		t.Run(marker, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, marker), []byte("PROJECT_NAME=x\n"), 0o600); err != nil {
				t.Fatalf("write %s: %v", marker, err)
			}

			got, err := FindNSelfRoot(dir)
			if err != nil {
				t.Fatalf("%s should mark a project root: %v", marker, err)
			}
			if got != dir {
				t.Errorf("got root %q, want %q", got, dir)
			}
		})
	}
}

// TestFindNSelfRoot_IgnoresLocalOnlyOverlays keeps the change narrow. A
// directory holding only a never-committed overlay is not a project someone
// checked out, and treating it as one would make `nself build` act on a stray
// secrets file.
func TestFindNSelfRoot_IgnoresLocalOnlyOverlays(t *testing.T) {
	for _, overlay := range []string{".env.secrets", ".env.local"} {
		t.Run(overlay, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, overlay), []byte("SECRET=x\n"), 0o600); err != nil {
				t.Fatalf("write %s: %v", overlay, err)
			}
			if _, err := FindNSelfRoot(dir); err == nil {
				t.Errorf("%s alone should not mark a project root", overlay)
			}
		})
	}
}

// TestFindNSelfRoot_ErrorNamesWhatItLookedFor checks the other half of the
// report: "no nself project found. Run 'nself init'" gave no way to work out
// why, on a directory that plainly had configuration in it.
func TestFindNSelfRoot_ErrorNamesWhatItLookedFor(t *testing.T) {
	_, err := FindNSelfRoot(t.TempDir())
	if err == nil {
		t.Fatal("expected an error for an empty directory")
	}
	for _, want := range []string{".env", ".env.dev"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name %s as a file it looked for: %v", want, err)
		}
	}
}
