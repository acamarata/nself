package repoqa

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the current working directory until it finds go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found walking up from working directory")
		}
		dir = parent
	}
}

// TestGofmtClean fails when any first-party Go file is not gofmt-formatted.
// vendor/ and testdata/ are excluded: vendored code is upstream's to format and
// testdata fixtures are deliberately allowed to be non-canonical.
func TestGofmtClean(t *testing.T) {
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("gofmt not on PATH")
	}
	root := repoRoot(t)

	var targets []string
	for _, d := range []string{"cmd", "internal", "tools"} {
		if _, err := os.Stat(filepath.Join(root, d)); err == nil {
			targets = append(targets, d)
		}
	}
	if len(targets) == 0 {
		t.Fatal("no target directories found")
	}

	cmd := exec.Command("gofmt", append([]string{"-l"}, targets...)...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("gofmt -l failed: %v", err)
	}

	var bad []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "/testdata/") {
			continue
		}
		bad = append(bad, line)
	}

	if len(bad) > 0 {
		t.Fatalf("%d file(s) are not gofmt-formatted. Run `make fmt`:\n  %s",
			len(bad), strings.Join(bad, "\n  "))
	}
}
