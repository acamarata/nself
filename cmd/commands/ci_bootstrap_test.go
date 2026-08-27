package commands

// ci_bootstrap_test.go — Guards the installed-CLI gate bootstrap.
//
// Purpose: an installed `nself` ships as a single binary, so the nself-ci gate
//          source is not adjacent to it. Before this, `nself ci` simply errored
//          with "nself-ci binary not found on PATH" and told the user to clone
//          a second repo and run `go build` by hand. That manual step is the
//          reason `nself ci` could not be made a required status check, which
//          is what the two-server rule depends on.
// Inputs:  a temporary HOME.
// Outputs: assertions on ciCacheBinary / ensureCIBinaryFromModule.
// Constraints: no network. The fetch path itself is deliberately NOT exercised
//          here — it shells out to `go install` and, on a cold machine, pulls a
//          whole Go toolchain (measured at 112s). Only the cache-hit and
//          path-resolution halves are unit-testable; the fetch is covered by
//          the bounded-timeout error path instead.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setHome points os.UserHomeDir at a temp dir on every platform. HOME alone is
// not enough: on Windows os.UserHomeDir reads %USERPROFILE%, so a test that
// sets only HOME silently uses the runner's real home directory.
func setHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

// TestCICacheBinary_UnderNselfDir pins where the fetched gate is cached.
// It lives under ~/.nself/ rather than GOBIN so it is not mixed into the user's
// own tools and is cleared along with the rest of nSelf's state.
func TestCICacheBinary_UnderNselfDir(t *testing.T) {
	home := setHome(t)

	got, err := ciCacheBinary()
	if err != nil {
		t.Fatalf("ciCacheBinary: %v", err)
	}

	want := filepath.Join(home, ".nself", "bin", "nself-ci")
	if got != want {
		t.Errorf("cache path = %q, want %q", got, want)
	}
}

// TestEnsureCIBinaryFromModule_CacheHit is the invariant that keeps the fetch
// a one-time cost. If this regresses, every `nself ci` invocation re-runs
// `go install` and the command becomes unusable as a pre-commit gate.
func TestEnsureCIBinaryFromModule_CacheHit(t *testing.T) {
	home := setHome(t)

	cached := filepath.Join(home, ".nself", "bin", "nself-ci")
	if err := os.MkdirAll(filepath.Dir(cached), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cached, []byte("#!/bin/sh\nexit 0\n"), 0o750); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Must return immediately from cache. A network fetch here would hang the
	// test rather than fail it, which is itself the signal.
	got, err := ensureCIBinaryFromModule(false)
	if err != nil {
		t.Fatalf("ensureCIBinaryFromModule: %v", err)
	}
	if got != cached {
		t.Errorf("returned %q, want the cached binary %q", got, cached)
	}
}

// TestCIGateModule_IsPinnedToPublicPath guards the module path. It must stay a
// constant: it is interpolated into an exec.Command, so anything user-derived
// here would be a command-injection surface.
func TestCIGateModule_IsPinnedToPublicPath(t *testing.T) {
	t.Parallel()

	const wantPrefix = "github.com/nself-org/plugins/free/ci"
	if !strings.HasPrefix(ciGateModule, wantPrefix) {
		t.Errorf("ciGateModule = %q, must be under %q", ciGateModule, wantPrefix)
	}
	// `go install` requires a main package; the gate's is ./cmd/.
	if !strings.Contains(ciGateModule, "/cmd@") {
		t.Errorf("ciGateModule = %q, must target the ./cmd/ main package", ciGateModule)
	}
}

// TestCIGateFetchTimeout_IsGenerousButFinite pins the bound. A cold fetch was
// measured at 112s because Go transparently downloads a newer toolchain
// ("requires go >= 1.26.4; switching to go1.26.7"), so a tight timeout would
// break first runs on slow links. Unbounded is worse: that is exactly how this
// presented while it was being built — a silent, indefinite stall.
func TestCIGateFetchTimeout_IsGenerousButFinite(t *testing.T) {
	t.Parallel()

	if ciGateFetchTimeout <= 0 {
		t.Fatal("fetch must be bounded; an unbounded go install stalls with no output")
	}
	if ciGateFetchTimeout.Minutes() < 5 {
		t.Errorf("timeout %s is too tight — a cold fetch includes a Go toolchain download (~112s measured, slower on poor links)", ciGateFetchTimeout)
	}
	if ciGateFetchTimeout.Minutes() > 30 {
		t.Errorf("timeout %s is effectively unbounded for an interactive command", ciGateFetchTimeout)
	}
}
