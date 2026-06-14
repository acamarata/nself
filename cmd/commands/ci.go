package commands

// Purpose: nself ci — local CI gate runner.
//   Runs the nself-ci gate suite for a repo (lint/test/build + gitleaks) then
//   posts a GitHub nself-ci commit status via gh OAuth. Replaces billing-blocked
//   GitHub Actions as the merge gate so branch protection can require nself-ci.
// Inputs:  [flags] [repo-root]
// Outputs: gate table to stdout, GitHub commit status posted, exit 0/1
// Constraints: requires gh CLI with repo scope; no hardcoded tokens.
// SPORT: CLI-CMD-CI-001

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci [repo-root]",
	Short: "Run the nself-ci gate suite and post a GitHub commit status",
	Long: `Run the local CI gate suite for a repository.

Detects the repo stack (Go, Node/TS, Flutter) and runs the appropriate
lint, test, and build checks plus a gitleaks secret scan. Posts a
"nself-ci" GitHub commit status via gh OAuth so branch protection can
require nself-ci instead of billing-blocked GitHub Actions checks.

Examples:
  nself ci                        # gate current directory, post status
  nself ci --check                # gate only, no status posted
  nself ci --no-gitleaks .        # skip secret scan
  nself ci --sha abc1234 .        # override commit SHA
  nself ci /path/to/repo          # gate a specific repo`,
	RunE: runCI,
}

func init() {
	ciCmd.Flags().Bool("check", false, "Run gates only; do not post a GitHub commit status")
	ciCmd.Flags().Bool("no-gitleaks", false, "Skip the gitleaks secret scan")
	ciCmd.Flags().Bool("no-status", false, "Alias for --check")
	ciCmd.Flags().String("sha", "", "Commit SHA to report on (default: HEAD)")
	ciCmd.Flags().String("owner", "", "GitHub owner (default: from git remote)")
	ciCmd.Flags().String("repo", "", "GitHub repo name (default: from git remote)")
	ciCmd.Flags().BoolP("verbose", "v", false, "Print each gate command before running")
	RootCmd.AddCommand(ciCmd)
}

func runCI(cmd *cobra.Command, args []string) error {
	// Resolve repo root.
	repoRoot := "."
	if len(args) > 0 {
		repoRoot = args[0]
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}
	repoRoot = abs

	// Flags.
	checkMode, _ := cmd.Flags().GetBool("check")
	noStatus, _ := cmd.Flags().GetBool("no-status")
	noGitleaks, _ := cmd.Flags().GetBool("no-gitleaks")
	sha, _ := cmd.Flags().GetString("sha")
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	verbose, _ := cmd.Flags().GetBool("verbose")

	postStatus := !checkMode && !noStatus

	// Build the nself-ci binary path (from plugins/free/ci).
	binaryPath, buildErr := ensureCIBinary(verbose)
	if buildErr != nil {
		return fmt.Errorf("cannot build nself-ci gate binary: %w", buildErr)
	}

	// Build argument list for the nself-ci binary.
	ciArgs := []string{}
	if checkMode || noStatus {
		ciArgs = append(ciArgs, "--no-status")
	}
	if noGitleaks {
		ciArgs = append(ciArgs, "--no-gitleaks")
	}
	if verbose {
		ciArgs = append(ciArgs, "-v")
	}
	if sha != "" {
		ciArgs = append(ciArgs, "--sha", sha)
	}
	if owner != "" {
		ciArgs = append(ciArgs, "--owner", owner)
	}
	if repo != "" {
		ciArgs = append(ciArgs, "--repo", repo)
	}
	ciArgs = append(ciArgs, repoRoot)

	ui.Section("nself-ci gate")
	if postStatus {
		ui.Info(fmt.Sprintf("repo: %s", repoRoot))
	} else {
		ui.Info(fmt.Sprintf("repo: %s (check mode — no status posted)", repoRoot))
	}

	// Run the gate binary with inherited stdin/stdout/stderr for live output.
	c := exec.Command(binaryPath, ciArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Gate failures are reported by the binary; propagate exit code.
			return fmt.Errorf("gate failed")
		}
		return fmt.Errorf("nself-ci: %w", err)
	}
	return nil
}

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

	if pluginDir == "" {
		return "", fmt.Errorf("nself-ci binary not found on PATH and plugin source not found near CLI binary.\n" +
			"Install gitleaks and build the plugin:\n" +
			"  cd plugins/free/ci && go build -o nself-ci ./cmd/")
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
