package repoqa

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// trackedFixtures are files that live in the working tree, are committed, and
// must never be rewritten by a test run.
//
// cmd/commands/.env.example is the one that caused trouble: a test that
// disabled the source-repo guard and ran a real command from inside
// cmd/commands treated the source tree as a project and wrote to it. Nothing
// noticed until `git add -A` swept the change into a commit and Doc-Sync went
// red two pushes later.
//
// Checking git's own view rather than a golden hash means the test keeps
// working when the fixture is deliberately edited and committed.
var trackedFixtures = []string{
	"cmd/commands/.env.example",
	"cmd/commands/.gitignore",
}

// TestTrackedFixturesAreUnmodified fails when a tracked fixture differs from
// its committed content.
//
// Run at the end of a local `go test ./...` this catches the corruption
// immediately, in the working copy, with a message naming the file — instead of
// letting it reach a commit. It is skipped outside a git checkout so it cannot
// fail in a source tarball.
func TestTrackedFixturesAreUnmodified(t *testing.T) {
	root := repoRoot(t)

	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("not a git checkout")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	var dirty []string
	for _, rel := range trackedFixtures {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			continue // deleted on purpose is a separate concern
		}

		cmd := exec.Command("git", "diff", "--quiet", "--", rel)
		cmd.Dir = root
		err := cmd.Run()
		if err == nil {
			continue
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("git diff %s: %v", rel, err)
		}
		if exitErr.ExitCode() != 1 {
			t.Fatalf("git diff %s exited %d: %v", rel, exitErr.ExitCode(), err)
		}
		dirty = append(dirty, rel)
	}

	if len(dirty) > 0 {
		t.Fatalf("%d tracked fixture(s) were modified in the working tree:\n  %s\n\n"+
			"A test almost certainly wrote to the source tree. Find it and give it a\n"+
			"t.TempDir() to work in — do NOT commit the churn, and do not add the file\n"+
			"to .gitignore (it is a committed fixture). Restore with:\n"+
			"  git checkout -- %s",
			len(dirty), strings.Join(dirty, "\n  "), strings.Join(dirty, " "))
	}
}
