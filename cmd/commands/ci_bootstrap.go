package commands

// ci_bootstrap.go — Resolving the nself-ci gate binary.
//
// Purpose: `nself ci` runs its gates from a separate binary built from the free
//   `ci` plugin. Split out of ci.go (which crossed the 300-line cap) because
//   locating/fetching the gate is a distinct concern from running it.
// Inputs:  the CLI's own executable path, PATH, HOME, a Go toolchain.
// Outputs: an absolute path to a runnable nself-ci binary.
// Constraints: the fetch is bounded and announced; it must be a one-time cost,
//   so the cache-hit path is the invariant guarded by ci_bootstrap_test.go.
// SPORT: CLI-CMD-CI-001

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ensureCIBinary returns the path to the nself-ci binary, building it first
// if necessary. Looks for the plugin source relative to the CLI source tree,
// or falls back to PATH if a pre-built nself-ci is already there.
func ensureCIBinary(verbose bool) (string, error) {
	// Fast path: nself-ci already on PATH (installed or previously built).
	if p, err := exec.LookPath("nself-ci"); err == nil {
		return p, nil
	}

	// Locate the plugin source relative to this binary.
	// The CLI binary lives at, e.g., ~/Sites/nself/cli/nself (dev) or
	// /usr/local/bin/nself (installed). In dev the plugin source is adjacent.
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	candidates := []string{
		// Dev: cli/ → parent nself/ → plugins/free/ci
		filepath.Join(filepath.Dir(exe), "..", "plugins", "free", "ci"),
		filepath.Join(filepath.Dir(exe), "..", "..", "plugins", "free", "ci"),
	}

	var pluginDir string
	for _, c := range candidates {
		abs, _ := filepath.Abs(c)
		if fileExists(filepath.Join(abs, "cmd", "main.go")) {
			pluginDir = abs
			break
		}
	}

	// Installed CLI: the plugin source is not adjacent, because only the single
	// `nself` binary was distributed. Fetch and cache the gate instead of
	// telling the user to clone a second repo and build Go by hand — that
	// requirement is why `nself ci` could not be a required check.
	if pluginDir == "" {
		return ensureCIBinaryFromModule(verbose)
	}

	binary := filepath.Join(pluginDir, "nself-ci")

	// Build if binary is missing or source is newer.
	if needsBuild(binary, pluginDir) {
		if verbose {
			fmt.Fprintf(os.Stderr, "[nself-ci] building gate binary from %s\n", pluginDir)
		}
		var stderr bytes.Buffer
		build := exec.Command("go", "build", "-o", binary, "./cmd/")
		build.Dir = pluginDir
		build.Stderr = &stderr
		if err := build.Run(); err != nil {
			return "", fmt.Errorf("go build failed: %w\n%s", err, strings.TrimSpace(stderr.String()))
		}
	}
	return binary, nil
}

// ciGateModule is the public Go module providing the nself-ci gate binary.
const ciGateModule = "github.com/nself-org/plugins/free/ci/cmd@latest"

// ciGateFetchTimeout bounds the one-off gate fetch. Generous because it may
// include a Go toolchain download, but finite so a first run cannot hang
// silently forever.
const ciGateFetchTimeout = 10 * time.Minute

// ciCacheBinary returns the path the fetched gate binary is cached at.
// Kept under the nSelf config dir rather than GOBIN so it is not mixed in with
// the user's own tools and can be cleared with the rest of nSelf's state.
func ciCacheBinary() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".nself", "bin", "nself-ci"), nil
}

// ensureCIBinaryFromModule resolves the gate for an INSTALLED CLI, where the
// plugin source is not on disk beside the binary.
//
// Order: cached build, then `go install` of the public module into that cache.
// Only reached when the binary is absent from PATH and no adjacent source
// exists, so it costs nothing on a dev checkout or a machine that already has
// the gate.
//
// Requires a Go toolchain. That is a real constraint and the error says so
// plainly instead of failing with a bare "not found on PATH", which is what
// this replaces.
func ensureCIBinaryFromModule(verbose bool) (string, error) {
	cached, err := ciCacheBinary()
	if err != nil {
		return "", err
	}

	// Already fetched by a previous run.
	if fileExists(cached) {
		return cached, nil
	}

	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf(
			"nself-ci gate not installed, and no Go toolchain is available to fetch it.\n"+
				"Either install Go and re-run, or build the gate manually:\n"+
				"  git clone https://github.com/nself-org/plugins\n"+
				"  cd plugins/free/ci && go build -o %s ./cmd/", cached)
	}

	if err := os.MkdirAll(filepath.Dir(cached), 0o750); err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(cached), err)
	}

	// Announced unconditionally, not only under -v. This runs once per machine
	// and can take minutes; a silent stall is indistinguishable from a hang, and
	// that is exactly how it presented while this was being built.
	fmt.Fprintf(os.Stderr, "[nself-ci] gate not installed; fetching %s (one-time, may take a few minutes)\n", ciGateModule)

	// Bounded: the gate module declares a newer Go than many machines have, so
	// `go install` may transparently download a whole toolchain first
	//   go: ... requires go >= 1.26.4; switching to go1.26.7
	// which is seconds on a warm cache and minutes on a cold one. Without a
	// deadline a first run just appears to hang with no output.
	ctx, cancel := context.WithTimeout(context.Background(), ciGateFetchTimeout)
	defer cancel()

	// GOBIN targets the cache directly, so the binary lands where we look for
	// it next time rather than in the user's GOPATH/bin.
	var stderr bytes.Buffer
	install := exec.CommandContext(ctx, "go", "install", ciGateModule) //nolint:gosec // fixed module path
	install.Env = append(os.Environ(), "GOBIN="+filepath.Dir(cached))
	// Under -v mirror Go's own progress (module and toolchain downloads) to the
	// terminal as it happens, while still keeping a copy for the error message.
	if verbose {
		install.Stderr = io.MultiWriter(&stderr, os.Stderr)
	} else {
		install.Stderr = &stderr
	}
	if err := install.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf(
				"fetching the nself-ci gate timed out after %s.\n"+
					"Go may be downloading a newer toolchain; re-run to resume, or build it directly:\n"+
					"  git clone https://github.com/nself-org/plugins\n"+
					"  cd plugins/free/ci && go build -o %s ./cmd/", ciGateFetchTimeout, cached)
		}
		return "", fmt.Errorf("fetching nself-ci gate failed: %w\n%s\n"+
			"Build it manually if this persists:\n"+
			"  git clone https://github.com/nself-org/plugins\n"+
			"  cd plugins/free/ci && go build -o %s ./cmd/",
			err, strings.TrimSpace(stderr.String()), cached)
	}

	// `go install` names the binary after the package directory ("cmd"), not
	// the module, so it has to be moved into place.
	installed := filepath.Join(filepath.Dir(cached), "cmd")
	if fileExists(installed) && installed != cached {
		if err := os.Rename(installed, cached); err != nil {
			return "", fmt.Errorf("placing gate binary at %s: %w", cached, err)
		}
	}
	if !fileExists(cached) {
		return "", fmt.Errorf("gate install reported success but %s is missing", cached)
	}
	return cached, nil
}

// needsBuild returns true if the binary is missing or any source file is newer.
func needsBuild(binary, pluginDir string) bool {
	info, err := os.Stat(binary)
	if err != nil {
		return true // binary missing
	}
	sources := []string{
		filepath.Join(pluginDir, "cmd", "main.go"),
		filepath.Join(pluginDir, "internal", "gate.go"),
		filepath.Join(pluginDir, "internal", "status.go"),
	}
	for _, src := range sources {
		si, err := os.Stat(src)
		if err != nil {
			continue
		}
		if si.ModTime().After(info.ModTime()) {
			return true
		}
	}
	return false
}

// fileExists returns true if the path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
