package commands

// clean_test.go — Unit tests for the clean command's --all / --yes flags.
//
// Purpose: --all was documented as system-wide `docker system prune` but never
// implemented (see PR #262, which correctly removed the claim rather than
// invent behavior). This verifies the real flags exist on the real cleanCmd
// (not a hand-listed stub — that exact mistake let doctor's --quick flag ship
// unregistered for over a month, see doctor_test.go) and that --all refuses
// to prune without confirmation.
// Inputs:  cobra command execution against runClean, a spy in place of
//          dockerSystemPrune.
// Outputs: pass/fail per assertion below.
// Constraints: no Docker daemon required — the docker system prune call is
//              stubbed via the package-level dockerSystemPrune variable.

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCleanCmd_AllFlagRegistered is a regression guard: checks the flag
// directly on the real cleanCmd, not a hand-listed copy.
func TestCleanCmd_AllFlagRegistered(t *testing.T) {
	f := cleanCmd.Flags().Lookup("all")
	if f == nil {
		t.Fatal("clean command has no --all flag")
	}
	if f.Value.Type() != "bool" {
		t.Fatalf("--all should be a bool flag, got %q", f.Value.Type())
	}
}

// TestCleanCmd_YesFlagRegistered checks --yes directly on the real cleanCmd.
func TestCleanCmd_YesFlagRegistered(t *testing.T) {
	f := cleanCmd.Flags().Lookup("yes")
	if f == nil {
		t.Fatal("clean command has no --yes flag")
	}
	if f.Value.Type() != "bool" {
		t.Fatalf("--yes should be a bool flag, got %q", f.Value.Type())
	}
}

// newCleanTestCmd builds a cobra.Command wired to the real runClean function
// (not a reimplementation) with the same flags production registers, so
// behavior tests exercise the actual code path without mutating the shared
// global cleanCmd/RootCmd state.
func newCleanTestCmd() *cobra.Command {
	c := &cobra.Command{Use: "clean", RunE: runClean}
	c.Flags().Bool("all", false, "")
	c.Flags().Bool("yes", false, "")
	c.SetContext(context.Background())
	return c
}

// withTempCwd runs fn with the process working directory set to a fresh
// temp dir, restoring the original afterward.
func withTempCwd(t *testing.T, fn func()) {
	t.Helper()
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("restore os.Chdir: %v", err)
		}
	}()
	fn()
}

// TestCleanCmd_AllWithoutConfirmation_DoesNotPrune verifies that --all
// without --yes, and a non-affirmative response at the prompt, never invokes
// the host-wide prune.
func TestCleanCmd_AllWithoutConfirmation_DoesNotPrune(t *testing.T) {
	origPrune := dockerSystemPrune
	called := false
	dockerSystemPrune = func(ctx context.Context, out, errOut io.Writer) error {
		called = true
		return nil
	}
	defer func() { dockerSystemPrune = origPrune }()

	withTempCwd(t, func() {
		cmd := newCleanTestCmd()
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
		cmd.SetIn(strings.NewReader("no\n"))
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)

		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("runClean returned error: %v", err)
		}

		if called {
			t.Fatal("expected dockerSystemPrune NOT to be called without confirmation")
		}
		if !strings.Contains(out.String(), "canceled") {
			t.Fatalf("expected cancellation message in output, got: %q", out.String())
		}
	})
}

// TestCleanCmd_AllWithoutConfirmation_EOF verifies that an EOF on stdin
// (e.g. a non-interactive pipe) is treated as a decline, not a hang or a
// silent prune.
func TestCleanCmd_AllWithoutConfirmation_EOF(t *testing.T) {
	origPrune := dockerSystemPrune
	called := false
	dockerSystemPrune = func(ctx context.Context, out, errOut io.Writer) error {
		called = true
		return nil
	}
	defer func() { dockerSystemPrune = origPrune }()

	withTempCwd(t, func() {
		cmd := newCleanTestCmd()
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
		cmd.SetIn(strings.NewReader(""))
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)

		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("runClean returned error: %v", err)
		}
		if called {
			t.Fatal("expected dockerSystemPrune NOT to be called on EOF")
		}
	})
}

// TestCleanCmd_AllWithYes_Prunes verifies that --yes skips the prompt and
// invokes the host-wide prune.
func TestCleanCmd_AllWithYes_Prunes(t *testing.T) {
	origPrune := dockerSystemPrune
	called := false
	dockerSystemPrune = func(ctx context.Context, out, errOut io.Writer) error {
		called = true
		return nil
	}
	defer func() { dockerSystemPrune = origPrune }()

	withTempCwd(t, func() {
		cmd := newCleanTestCmd()
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
		if err := cmd.Flags().Set("yes", "true"); err != nil {
			t.Fatalf("set --yes: %v", err)
		}
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)

		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("runClean returned error: %v", err)
		}
		if !called {
			t.Fatal("expected dockerSystemPrune to be called with --yes")
		}
	})
}

// TestCleanCmd_AllWithConfirmation_Prunes verifies that typing "yes" at the
// prompt invokes the host-wide prune.
func TestCleanCmd_AllWithConfirmation_Prunes(t *testing.T) {
	origPrune := dockerSystemPrune
	called := false
	dockerSystemPrune = func(ctx context.Context, out, errOut io.Writer) error {
		called = true
		return nil
	}
	defer func() { dockerSystemPrune = origPrune }()

	withTempCwd(t, func() {
		cmd := newCleanTestCmd()
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all: %v", err)
		}
		cmd.SetIn(strings.NewReader("yes\n"))
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)

		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("runClean returned error: %v", err)
		}
		if !called {
			t.Fatal("expected dockerSystemPrune to be called after typing yes")
		}
	})
}

// TestCleanCmd_WithoutAll_NeverPrunes verifies default (no --all) behavior
// never touches the host-wide prune, regardless of stdin content.
func TestCleanCmd_WithoutAll_NeverPrunes(t *testing.T) {
	origPrune := dockerSystemPrune
	called := false
	dockerSystemPrune = func(ctx context.Context, out, errOut io.Writer) error {
		called = true
		return nil
	}
	defer func() { dockerSystemPrune = origPrune }()

	withTempCwd(t, func() {
		cmd := newCleanTestCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)

		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("runClean returned error: %v", err)
		}
		if called {
			t.Fatal("expected dockerSystemPrune NOT to be called without --all")
		}
	})
}
