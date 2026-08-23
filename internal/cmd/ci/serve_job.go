// Package ci — serve_job.go
//
// Purpose: Run a single CI gate job in an ephemeral Docker container.
//
//	Clones the target repo ref into a temp workdir, mounts it into a container
//	with CPU/memory limits and a per-job timeout, runs the nself-ci binary, then
//	posts a GitHub commit status and emits a completion event to NSELF_CI_EVENT_SINK.
//	Container is always cleaned up — leak-free even on timeout or panic.
//
// Inputs:  ciJob, binaryPath string, ServeConfig
// Outputs: GitHub commit status (pending → success/failure); optional event POST
// Constraints: Requires Docker daemon + gh CLI on PATH; uses exec (not Docker SDK)
//
//	to keep vendor tree clean; SPORT CLI-CMD-CI-SERVE-001
package ci

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"
)

// runJob clones the ref, runs the gate in Docker, posts status, and emits a
// completion event. Errors are logged but never propagate (best-effort async).
func runJob(job ciJob, binaryPath string, cfg ServeConfig) {
	start := time.Now()
	shortSHA := job.SHA[:min(7, len(job.SHA))]
	label := fmt.Sprintf("%s @ %s", job.RepoFullName, shortSHA)

	ui.Info(fmt.Sprintf("[ci-serve] start  %s  (%s)", label, job.EventType))

	// Post "pending" status immediately so PR shows activity.
	owner, repoName := splitFullName(job.RepoFullName)
	_ = postCommitStatus(owner, repoName, job.SHA, "pending", "nself-ci: running…")

	// Clone the specific SHA into an ephemeral workdir.
	workdir, err := cloneRef(job, cfg)
	if err != nil {
		ui.Warn(fmt.Sprintf("[ci-serve] clone failed %s: %v", label, err))
		_ = postCommitStatus(owner, repoName, job.SHA, "error",
			fmt.Sprintf("clone failed: %v", truncateStr(err.Error(), 100)))
		emitEvent(completionEvent{
			Repo: job.RepoFullName, Ref: job.Ref, SHA: job.SHA,
			Status: "error", Duration: time.Since(start).String(),
		})
		return
	}
	defer os.RemoveAll(workdir) // always clean up

	// Run gate inside Docker container with resource limits.
	passed, summary, runErr := runGateInDocker(binaryPath, workdir, cfg)
	elapsed := time.Since(start)

	state := "success"
	if !passed || runErr != nil {
		state = "failure"
		if runErr != nil {
			summary = fmt.Sprintf("docker run error: %v", truncateStr(runErr.Error(), 100))
		}
	}

	_ = postCommitStatus(owner, repoName, job.SHA, state, truncateStr(summary, 140))

	ui.Info(fmt.Sprintf("[ci-serve] done   %s  %s  %s", label, state, elapsed.Round(time.Second)))

	emitEvent(completionEvent{
		Repo:     job.RepoFullName,
		Ref:      job.Ref,
		SHA:      job.SHA,
		Status:   state,
		Duration: elapsed.String(),
		Summary:  summary,
	})
}

// cloneRef performs a shallow clone of job.CloneURL at the exact SHA into a
// temp dir under cfg.WorkDir. Returns the path to the cloned directory.
func cloneRef(job ciJob, cfg ServeConfig) (string, error) {
	dir, err := os.MkdirTemp(cfg.WorkDir, "nself-ci-*")
	if err != nil {
		return "", fmt.Errorf("mktemp: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Shallow clone + fetch the specific commit.
	cloneArgs := []string{
		"clone", "--depth=50", "--no-tags",
		job.CloneURL, dir,
	}
	if out, err := runCmd(ctx, "", "git", cloneArgs...); err != nil {
		return dir, fmt.Errorf("git clone: %w\n%s", err, out)
	}

	// Reset to the exact SHA (handles PRs from forks or stale shallow clones).
	if out, err := runCmd(ctx, dir, "git", "fetch", "--depth=1", "origin", job.SHA); err != nil {
		// Non-fatal: SHA may already be in the shallow history.
		_ = out
	}
	if out, err := runCmd(ctx, dir, "git", "checkout", "--detach", job.SHA); err != nil {
		return dir, fmt.Errorf("git checkout %s: %w\n%s", job.SHA[:min(7, len(job.SHA))], err, out)
	}
	return dir, nil
}

// runGateInDocker runs the nself-ci binary inside a container.
// The workdir is bind-mounted read-write at /repo. The nself-ci binary is
// bind-mounted read-only from the host so Docker doesn't need to install Go.
// CPU/memory limits prevent a runaway build from starving the ops box.
func runGateInDocker(binaryPath, workdir string, cfg ServeConfig) (passed bool, summary string, err error) {
	containerName := fmt.Sprintf("nself-ci-%d", time.Now().UnixNano())

	// Check Docker availability.
	if _, lookErr := exec.LookPath("docker"); lookErr != nil {
		// Docker not available — fall back to running the binary directly on host.
		return runGateDirect(binaryPath, workdir, cfg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.JobTimeout)*time.Second)
	defer cancel()

	args := []string{
		"run",
		"--rm",
		"--name", containerName,
		"--network=none",         // no outbound network (gate is local)
		"--cpus=1.5",             // cap CPU: 1.5 cores
		"--memory=1g",            // cap RAM: 1 GiB
		"--memory-swap=1g",       // disable swap
		"--read-only",            // immutable container FS
		"--tmpfs=/tmp:size=256m", // writable /tmp in RAM
		"-v", fmt.Sprintf("%s:/repo:rw", workdir),
		"-v", fmt.Sprintf("%s:/usr/local/bin/nself-ci:ro", binaryPath),
		"--workdir=/repo",
		"ubuntu:24.04",
		"/usr/local/bin/nself-ci", "--check", "--no-gitleaks", "/repo",
	}

	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()
	output := buf.String()

	if ctx.Err() != nil {
		// Timeout — clean up container (best-effort).
		_ = exec.Command("docker", "rm", "-f", containerName).Run()
		return false, "job timed out", fmt.Errorf("timeout after %ds", cfg.JobTimeout)
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Gate failure (not infrastructure error).
			return false, extractSummary(output), nil
		}
		return false, "docker run failed", fmt.Errorf("docker: %w — %s", runErr, truncateStr(output, 200))
	}
	return true, extractSummary(output), nil
}

// runGateDirect runs the nself-ci binary directly on the host when Docker is unavailable.
// This is the fallback path — full isolation requires Docker.
func runGateDirect(binaryPath, workdir string, cfg ServeConfig) (passed bool, summary string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.JobTimeout)*time.Second)
	defer cancel()

	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, binaryPath, "--check", "--no-gitleaks", workdir)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()
	output := buf.String()

	if ctx.Err() != nil {
		return false, "job timed out (direct mode)", fmt.Errorf("timeout after %ds", cfg.JobTimeout)
	}
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, extractSummary(output), nil
		}
		return false, "gate binary failed", fmt.Errorf("%w", runErr)
	}
	return true, extractSummary(output), nil
}

// postCommitStatus posts a nself-ci GitHub commit status via `gh api` (OAuth).
// Mirrors the logic in plugins/free/ci/internal/status.go without importing it
// (different Go module — invoked as a binary, not a library).
func postCommitStatus(owner, repo, sha, state, description string) error {
	if owner == "" || repo == "" || sha == "" {
		return fmt.Errorf("owner/repo/sha required")
	}
	endpoint := fmt.Sprintf("repos/%s/%s/statuses/%s", owner, repo, sha)
	args := []string{
		"api", "--method", "POST", endpoint,
		"-f", "state=" + state,
		"-f", "context=nself-ci",
		"-f", "description=" + truncateStr(description, 140),
	}
	var stderr bytes.Buffer
	cmd := exec.Command("gh", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh api: %w — %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// completionEvent is posted to NSELF_CI_EVENT_SINK when a job finishes.
// This is the seam for nself-cron-monitor (port 3839).
type completionEvent struct {
	Repo     string `json:"repo"`
	Ref      string `json:"ref"`
	SHA      string `json:"sha"`
	Status   string `json:"status"`   // "success" | "failure" | "error"
	Duration string `json:"duration"` // e.g. "1m23s"
	Summary  string `json:"summary,omitempty"`
}

// emitEvent POSTs the completion event to NSELF_CI_EVENT_SINK if configured.
// Best-effort: errors are logged but never propagate.
func emitEvent(ev completionEvent) {
	sink := os.Getenv("NSELF_CI_EVENT_SINK")
	if sink == "" {
		return
	}

	body, err := json.Marshal(ev)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sink, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ui.Warn(fmt.Sprintf("[ci-serve] event sink %s: %v", sink, err))
		return
	}
	_ = resp.Body.Close()
}

// runCmd executes a command and returns combined stdout+stderr.
func runCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// splitFullName splits "owner/repo" into (owner, repo).
func splitFullName(full string) (string, string) {
	parts := strings.SplitN(full, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return full, full
}

// extractSummary pulls the last non-empty line of nself-ci output as the summary.
func extractSummary(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return truncateStr(l, 140)
		}
	}
	return "gate complete"
}

// truncateStr truncates s to max runes, appending "…" if truncated.
func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// ensureCIBinary resolves the nself-ci gate binary path (also used from ci.go CLI side).
// Duplicating the lookup here keeps this package self-contained from the cmd layer.
func ensureCIBinary(verbose bool) (string, error) {
	if p, err := exec.LookPath("nself-ci"); err == nil {
		return p, nil
	}

	// Locate plugin source relative to the cli/ repo.
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	candidates := []string{
		filepath.Join(filepath.Dir(exe), "..", "plugins", "free", "ci"),
		filepath.Join(filepath.Dir(exe), "..", "..", "plugins", "free", "ci"),
	}

	var pluginDir string
	for _, c := range candidates {
		abs, _ := filepath.Abs(c)
		if info, err := os.Stat(filepath.Join(abs, "cmd", "main.go")); err == nil && !info.IsDir() {
			pluginDir = abs
			break
		}
	}
	if pluginDir == "" {
		return "", fmt.Errorf("nself-ci not on PATH and plugin source not found near CLI binary")
	}

	binary := filepath.Join(pluginDir, "nself-ci")
	if _, err := os.Stat(binary); err != nil {
		// Build it.
		if verbose {
			fmt.Fprintf(os.Stderr, "[nself-ci] building from %s\n", pluginDir)
		}
		var stderr bytes.Buffer
		build := exec.Command("go", "build", "-o", binary, "./cmd/")
		build.Dir = pluginDir
		build.Stderr = &stderr
		if err := build.Run(); err != nil {
			return "", fmt.Errorf("go build failed: %w — %s", err, strings.TrimSpace(stderr.String()))
		}
	}
	return binary, nil
}
