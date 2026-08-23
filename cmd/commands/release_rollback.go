package commands

// Purpose: Implements `nself release rollback`: reverts the Homebrew
// formula, resets the ping API's reported version, retags the admin Docker
// image, rewrites the changelog entry, and deletes the git tag/GitHub
// release for a bad release. Split out of release.go (CLI-R12) to separate
// the rollback cascade from the forward-release cascade
// (release_deploy_steps.go) and cobra wiring (release.go).
// Inputs: the releaseRollbackCmd cobra.Command (registered onto releaseCmd
// in release.go's init — untouched by this split), plus the from/to
// version strings being rolled back between.
// Outputs: errors surfaced back through runStep to the rollback's
// releaseResult; each step shells out to git, gh, docker, or brew.
// Constraints: pure move — no behavior changes. runReleaseRollback keeps
// the exact step order wired in release.go's releaseRollbackCmd.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

func runReleaseRollback(cmd *cobra.Command, args []string) error {
	fromVer := strings.TrimPrefix(args[0], "v")
	toVer := strings.TrimPrefix(args[1], "v")
	fromTag := "v" + fromVer

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	jsonOut, _ := cmd.Flags().GetBool("json")
	deleteTags, _ := cmd.Flags().GetBool("delete-tags")

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
	defer cancel()

	result := releaseResult{
		Version: fromVer,
		Tag:     fromTag,
		DryRun:  dryRun,
		Status:  "success",
	}

	if !jsonOut {
		if dryRun {
			fmt.Printf("%s DRY RUN — rollback %s → %s\n\n", ui.C(ui.Yellow, "[DRY-RUN]"), fromTag, "v"+toVer)
		} else {
			fmt.Printf("%s Rolling back %s → %s\n\n", ui.C(ui.Bold, "nSelf Rollback"), ui.C(ui.Red, fromTag), ui.C(ui.Green, "v"+toVer))
		}
	}

	// Step 1 — Homebrew revert
	if err := runStep(ctx, &result, 1, fmt.Sprintf("Revert Homebrew formula to %s", toVer), dryRun, func() error {
		return execHomebrewRevert(ctx, fromVer, toVer)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 2 — ping_api env reset
	if err := runStep(ctx, &result, 2, fmt.Sprintf("Reset ping_api NSELF_CLI_VERSION to %s", toVer), dryRun, func() error {
		return execPingAPIVersionReset(ctx, toVer)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 3 — Docker latest retag
	if err := runStep(ctx, &result, 3, fmt.Sprintf("Retag Docker :latest → admin:%s", toVer), dryRun, func() error {
		return execDockerRetag(ctx, toVer)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 4 — Changelog entry
	if err := runStep(ctx, &result, 4, "Emit rollback changelog entry", dryRun, func() error {
		return execRollbackChangelog(ctx, fromVer, toVer)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 5 — Delete tags (only with --delete-tags)
	if deleteTags {
		if err := runStep(ctx, &result, 5, fmt.Sprintf("Delete git tag %s (--delete-tags)", fromTag), dryRun, func() error {
			return execDeleteTag(ctx, fromTag)
		}); err != nil {
			return emitResult(result, jsonOut, err)
		}
	} else {
		result.Steps = append(result.Steps, releaseStepResult{
			Step: 5, Name: fmt.Sprintf("Delete git tag %s", fromTag),
			Status: "skipped", Message: "use --delete-tags to delete (destructive)",
		})
	}

	return emitResult(result, jsonOut, nil)
}

// ── rollback implementations ─────────────────────────────────────────────────

func execHomebrewRevert(ctx context.Context, fromVer, toVer string) error {
	c := exec.CommandContext(ctx, "gh", "workflow", "run", "update-formula.yml",
		"--repo", "nself-org/homebrew-nself",
		"-f", "version="+toVer,
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("homebrew revert %s → %s: %w", fromVer, toVer, err)
	}
	return nil
}

func execPingAPIVersionReset(ctx context.Context, toVer string) error {
	c := exec.CommandContext(ctx, "vercel", "env", "add", "NSELF_CLI_VERSION", "production")
	c.Dir = "../web/backend/services/ping_api"
	c.Stdin = strings.NewReader(toVer)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func execDockerRetag(ctx context.Context, toVer string) error {
	pull := exec.CommandContext(ctx, "docker", "pull", fmt.Sprintf("nself/nself-admin:%s", toVer))
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("docker pull admin:%s: %w", toVer, err)
	}
	tag := exec.CommandContext(ctx, "docker", "tag",
		fmt.Sprintf("nself/nself-admin:%s", toVer),
		"nself/nself-admin:latest",
	)
	tag.Stdout = os.Stdout
	tag.Stderr = os.Stderr
	if err := tag.Run(); err != nil {
		return err
	}
	push := exec.CommandContext(ctx, "docker", "push", "nself/nself-admin:latest")
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	return push.Run()
}

func execRollbackChangelog(_ context.Context, fromVer, toVer string) error {
	entry := fmt.Sprintf("\n## [%s] — ROLLBACK\n\nRolled back v%s → v%s on %s\n",
		toVer+"-rollback", fromVer, toVer, time.Now().UTC().Format("2006-01-02"))
	f, err := os.OpenFile("CHANGELOG.md", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("append to CHANGELOG.md: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

func execDeleteTag(ctx context.Context, tag string) error {
	local := exec.CommandContext(ctx, "git", "tag", "--delete", tag)
	local.Stdout = os.Stdout
	local.Stderr = os.Stderr
	if err := local.Run(); err != nil {
		return fmt.Errorf("delete local tag %s: %w", tag, err)
	}
	remote := exec.CommandContext(ctx, "git", "push", "--delete", "origin", tag)
	remote.Stdout = os.Stdout
	remote.Stderr = os.Stderr
	if err := remote.Run(); err != nil {
		return fmt.Errorf("delete remote tag %s: %w", tag, err)
	}
	return nil
}
