package commands

// Purpose: Shared git/GitHub helpers used by both the forward-release and
// rollback cascades — creates and pushes an annotated tag (optionally in a
// sibling repo directory) and creates a GitHub Release for a tag via `gh`.
// Split out of release.go (CLI-R12); grouped separately because both
// release_deploy_steps.go and release_rollback.go call these.
// Inputs: a context.Context, a repo slug (owner/name), a tag string, and
// (for gitTagAndPushInRepo) the directory of a sibling repo checkout.
// Outputs: errors from the underlying git/gh subprocess invocations.
// Constraints: pure move — no behavior changes.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func gitTagAndPush(ctx context.Context, tag string) error {
	tagCmd := exec.CommandContext(ctx, "git", "tag", "-a", tag, "-m", "Release "+tag)
	tagCmd.Stdout = os.Stdout
	tagCmd.Stderr = os.Stderr
	if err := tagCmd.Run(); err != nil {
		return fmt.Errorf("git tag %s: %w", tag, err)
	}
	push := exec.CommandContext(ctx, "git", "push", "origin", tag)
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	return push.Run()
}

func gitTagAndPushInRepo(ctx context.Context, tag, dir string) error {
	tagCmd := exec.CommandContext(ctx, "git", "tag", "-a", tag, "-m", "Release "+tag)
	tagCmd.Dir = dir
	tagCmd.Stdout = os.Stdout
	tagCmd.Stderr = os.Stderr
	if err := tagCmd.Run(); err != nil {
		return fmt.Errorf("git tag %s in %s: %w", tag, dir, err)
	}
	push := exec.CommandContext(ctx, "git", "push", "origin", tag)
	push.Dir = dir
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	return push.Run()
}

func createGitHubRelease(ctx context.Context, repo, tag string) error {
	args := []string{
		"release", "create", tag,
		"--repo", repo,
		"--title", "Release " + tag,
		"--generate-notes",
	}
	c := exec.CommandContext(ctx, "gh", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
