package repoqa

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exitAllowList names the only non-main locations permitted to reference
// os.Exit, with the reason each is unavoidable. Anything else must signal a
// non-zero status by returning errs.Exit / errs.ExitWith, so deferred cleanup
// and the OTel span flush in PersistentPostRunE still run.
//
// Keys are repo-relative paths; the value is why the exception holds.
var exitAllowList = map[string]string{
	// Signal handlers cannot return an error up the cobra stack: the goroutine
	// that receives SIGINT has no caller to return to. lifecycle.TrapSignals
	// takes the exit function as a parameter precisely so it is injectable and
	// testable, and os.Exit is what production passes in.
	"cmd/commands/start.go": "passes os.Exit as the injected exit func to lifecycle.TrapSignals",
	"cmd/commands/stop.go":  "passes os.Exit as the injected exit func to lifecycle.TrapSignals",
}

// mainPackageDirs are package main entrypoints, where os.Exit is the correct
// and only way to set process status.
var mainPackageDirs = []string{
	"cmd/nself",
	"cmd/nself-backup-archive",
	"cmd/aistudio",
}

// TestNoOsExitOutsideMain enforces the repo rule from .claude/rules/go.md:
// "os.Exit() outside cmd/nself/main.go" is forbidden.
func TestNoOsExitOutsideMain(t *testing.T) {
	root := repoRoot(t)

	var violations []string

	for _, dir := range []string{"cmd", "internal"} {
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

			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)

			for _, m := range mainPackageDirs {
				if strings.HasPrefix(rel, m+"/") {
					return nil
				}
			}
			if _, ok := exitAllowList[rel]; ok {
				return nil
			}

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				code := line
				// Ignore mentions inside line comments; the rule is about calls.
				if idx := strings.Index(code, "//"); idx >= 0 {
					code = code[:idx]
				}
				if strings.Contains(code, "os.Exit(") {
					violations = append(violations,
						fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("os.Exit is only allowed in package main (see .claude/rules/go.md). "+
			"Return errs.Exit(code) instead.\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestExitAllowListEntriesStillExist keeps the allow-list from rotting into a
// set of stale excuses after the files it names are split or renamed.
func TestExitAllowListEntriesStillExist(t *testing.T) {
	root := repoRoot(t)
	for rel, reason := range exitAllowList {
		path := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("allow-listed file %s no longer exists (%q) — drop the entry", rel, reason)
			continue
		}
		if !strings.Contains(string(data), "os.Exit") {
			t.Errorf("allow-listed file %s no longer references os.Exit (%q) — drop the entry", rel, reason)
		}
	}
}
