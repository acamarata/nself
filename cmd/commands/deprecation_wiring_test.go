package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/deprecation"
	"github.com/spf13/cobra"
)

// captureStderr runs fn with os.Stderr replaced by a pipe and returns what was
// written. Used because the deprecation warning goes to the real os.Stderr, not
// to a cobra-managed writer.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		done <- buf.String()
	}()

	fn()

	os.Stderr = orig
	_ = w.Close()
	out := <-done
	_ = r.Close()
	return out
}

// TestDeprecatedAliasWarns is the regression guard for CLI-R03's second defect.
// `up` is a cobra alias of startCmd, so cmd.CommandPath() reports "nself start".
// Keying the registry lookup on CommandPath meant the warning could never fire
// for an aliased spelling — the exact mechanism every CLI-R09/R19 rename relies
// on. invokedCommandPath keys on what the user actually typed instead.
func TestDeprecatedAliasWarns(t *testing.T) {
	t.Setenv("NSELF_ALLOW_SOURCE_DIR", "1")

	out := captureStderr(t, func() {
		RootCmd.SetArgs([]string{"up"})
		_ = RootCmd.Execute()
	})

	if !strings.Contains(out, "[DEPRECATED] 'nself up'") {
		t.Fatalf("expected a deprecation warning for the `up` alias, got: %q", out)
	}
	if !strings.Contains(out, "nself start") {
		t.Fatalf("warning did not name the replacement: %q", out)
	}
}

// TestCanonicalSpellingDoesNotWarn is the other half: warning on the canonical
// name would make every correct `nself start` noisy.
func TestCanonicalSpellingDoesNotWarn(t *testing.T) {
	t.Setenv("NSELF_ALLOW_SOURCE_DIR", "1")

	out := captureStderr(t, func() {
		RootCmd.SetArgs([]string{"start"})
		_ = RootCmd.Execute()
	})

	if strings.Contains(out, "[DEPRECATED]") {
		t.Fatalf("canonical `nself start` emitted a deprecation warning: %q", out)
	}
}

// TestNoDeprecationWarningsFlagSuppresses verifies the scripted-use escape hatch.
func TestNoDeprecationWarningsFlagSuppresses(t *testing.T) {
	t.Setenv("NSELF_ALLOW_SOURCE_DIR", "1")

	out := captureStderr(t, func() {
		RootCmd.SetArgs([]string{"up", "--no-deprecation-warnings"})
		_ = RootCmd.Execute()
	})

	if strings.Contains(out, "[DEPRECATED]") {
		t.Fatalf("--no-deprecation-warnings did not suppress the warning: %q", out)
	}
}

// TestEveryDeprecatedCommandEntryIsReachable stops the registry filling up with
// entries that can never match. Each `type: command` entry must name a real
// command path or a real alias of one; otherwise the warning is dead weight and
// the rename it documents is silently unannounced.
func TestEveryDeprecatedCommandEntryIsReachable(t *testing.T) {
	reg, err := deprecation.LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("LoadEmbeddedRegistry: %v", err)
	}

	reachable := map[string]bool{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		reachable[c.CommandPath()] = true
		prefix := "nself"
		if p := c.Parent(); p != nil {
			prefix = p.CommandPath()
		}
		for _, a := range c.Aliases {
			reachable[prefix+" "+a] = true
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(RootCmd)

	var dead []string
	for _, name := range reg.Names() {
		item, _ := reg.Lookup(name)
		if item.Type != deprecation.TypeCommand {
			continue
		}
		if !reachable[name] {
			dead = append(dead, name)
		}
	}

	if len(dead) > 0 {
		t.Fatalf("deprecation registry has %d command entr(ies) no invocation can reach:\n  %s",
			len(dead), strings.Join(dead, "\n  "))
	}
}

// TestInstalledBinaryEmitsDeprecationWarning is the CLI-R03 end-to-end proof:
// build the CLI, put it in an otherwise-empty directory (no source tree, no
// registry.yaml on disk anywhere near it) and confirm the warning still fires.
// This is exactly the situation — `make install`, Homebrew, a release tarball —
// in which the old path-based loader silently produced no warnings at all.
func TestInstalledBinaryEmitsDeprecationWarning(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary; skipped under -short")
	}
	if runtime.GOOS == "windows" {
		t.Skip("binary staging in this test is POSIX-only")
	}

	root := findRepoRootForTest(t)
	installDir := t.TempDir()
	binPath := filepath.Join(installDir, "nself")

	build := exec.Command("go", "build", "-mod=vendor", "-o", binPath, "./cmd/nself/")
	build.Dir = root
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Run from a *different* empty directory so neither the binary's own
	// directory nor the working directory contains a source tree.
	runDir := t.TempDir()
	run := exec.Command(binPath, "up")
	run.Dir = runDir
	// Explicitly clear the override so only the embedded copy can be the source.
	run.Env = append(os.Environ(), deprecation.RegistryPathEnv+"=")

	var stderr bytes.Buffer
	run.Stderr = &stderr
	_ = run.Run() // a non-zero exit is expected: there is no project here.

	if !strings.Contains(stderr.String(), "[DEPRECATED] 'nself up'") {
		t.Fatalf("installed binary in an empty dir emitted no deprecation warning.\nstderr:\n%s", stderr.String())
	}
}

func findRepoRootForTest(t *testing.T) string {
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
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
