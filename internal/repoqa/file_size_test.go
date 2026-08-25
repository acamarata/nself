package repoqa

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// maxFileLines is the engineering-standard cap: no non-test Go file over 300
// lines (ASI Policy 3, .claude/rules/go.md).
const maxFileLines = 300

// budgetFile holds the number of files currently allowed to exceed the cap.
//
// CLI-R12 has 120 files to split, which cannot land in one change. A hard
// "zero violations" test would have to be skipped until the very last split,
// which means it protects nothing in the meantime. A ratchet protects
// immediately: the count may fall, never rise. Each split lowers the number in
// .github/file-size-budget.txt, and a new oversized file fails the build the
// day it is written.
//
// The budget is a single integer in its own file on purpose — a list of
// filenames would conflict on every concurrent split.
const budgetFile = ".github/file-size-budget.txt"

// oversizedFiles returns every non-test Go file over the cap, with its length.
func oversizedFiles(t *testing.T, root string) map[string]int {
	t.Helper()
	out := map[string]int{}

	for _, dir := range []string{"cmd", "internal", "tools"} {
		base := filepath.Join(root, dir)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "testdata" || info.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			n := 0
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 1024*1024), 1024*1024)
			for sc.Scan() {
				n++
			}
			if err := sc.Err(); err != nil {
				return err
			}

			if n > maxFileLines {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				out[filepath.ToSlash(rel)] = n
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	return out
}

func readBudget(t *testing.T, root string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, budgetFile))
	if err != nil {
		t.Fatalf("read %s: %v", budgetFile, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			t.Fatalf("%s: %q is not an integer", budgetFile, line)
		}
		return n
	}
	t.Fatalf("%s contains no budget number", budgetFile)
	return 0
}

// TestFileSizeBudgetNotExceeded fails when a new oversized file appears.
func TestFileSizeBudgetNotExceeded(t *testing.T) {
	root := repoRoot(t)
	over := oversizedFiles(t, root)
	budget := readBudget(t, root)

	if len(over) > budget {
		names := make([]string, 0, len(over))
		for f, n := range over {
			names = append(names, fmt.Sprintf("%s (%d lines)", f, n))
		}
		sort.Strings(names)
		t.Fatalf("%d files exceed the %d-line cap but the budget is %d.\n"+
			"Split the file you just added, or if you are deliberately raising the\n"+
			"budget, say why in the commit message.\n  %s",
			len(over), maxFileLines, budget, strings.Join(names, "\n  "))
	}
}

// TestFileSizeBudgetIsTight fails when the budget is left higher than reality,
// which would silently re-open room for new oversized files. Whoever splits a
// file must lower the number in the same change.
func TestFileSizeBudgetIsTight(t *testing.T) {
	root := repoRoot(t)
	over := oversizedFiles(t, root)
	budget := readBudget(t, root)

	if len(over) < budget {
		t.Fatalf("budget is %d but only %d files exceed the %d-line cap — "+
			"lower the number in %s to %d so the ratchet keeps holding",
			budget, len(over), maxFileLines, budgetFile, len(over))
	}
}
