package commands

import "testing"

// TestBuildCIArgs_FilesystemFlagForcesNoGit closes the residual gap in
// P6-E2-W1-S2-T6: nself ci --filesystem must translate into a --filesystem
// argument on the nself-ci gate binary, which plugins/free/ci's
// gitleaksArgs() uses to force gitleaks into filesystem (--no-git) scan mode
// even inside a real git checkout. The e2e proof that gitleaks itself skips
// the gitignored tree in git-mode lives in plugins/free/ci (a separate repo,
// out of this ticket's cli-only scope); this test guards the cli-side flag
// wiring that feeds it.
func TestBuildCIArgs_FilesystemFlagForcesNoGit(t *testing.T) {
	args := buildCIArgs(ciArgsInput{
		filesystem: true,
		repoRoot:   "/tmp/example",
	})

	found := false
	for _, a := range args {
		if a == "--filesystem" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --filesystem in argv, got %v", args)
	}
}

// TestBuildCIArgs_FilesystemFlagAbsentByDefault proves the flag is opt-in:
// a plain `nself ci` run must NOT force filesystem mode (git-mode is the
// default per the fix), or every checkout would pay the gitignored-tree scan
// cost this ticket exists to avoid.
func TestBuildCIArgs_FilesystemFlagAbsentByDefault(t *testing.T) {
	args := buildCIArgs(ciArgsInput{repoRoot: "/tmp/example"})

	for _, a := range args {
		if a == "--filesystem" {
			t.Fatalf("expected no --filesystem in argv by default, got %v", args)
		}
	}
}

// TestBuildCIArgs_AllFlagsMapCorrectly exercises the full flag surface so a
// future flag added to nself ci without a matching buildCIArgs branch fails
// loudly here instead of silently not reaching the gate binary.
func TestBuildCIArgs_AllFlagsMapCorrectly(t *testing.T) {
	args := buildCIArgs(ciArgsInput{
		checkMode:  true,
		noGitleaks: true,
		filesystem: true,
		verbose:    true,
		sha:        "abc1234",
		owner:      "nself-org",
		repo:       "cli",
		repoRoot:   "/tmp/example",
	})

	want := []string{
		"--no-status", "--no-gitleaks", "--filesystem", "-v",
		"--sha", "abc1234", "--owner", "nself-org", "--repo", "cli",
		"/tmp/example",
	}
	if len(args) != len(want) {
		t.Fatalf("argv length mismatch: got %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("argv[%d] = %q, want %q (full: got %v, want %v)", i, args[i], want[i], args, want)
		}
	}
}
