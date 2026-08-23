package commands

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// releaseVerRe validates a semver string including optional pre-release suffix
// (e.g. 1.2.3 or 1.2.3-rc.1). Anchored at both ends to prevent shell injection.
// Note: semverRe (without pre-release) is declared in release_check.go.
var releaseVerRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$`)

// badgePat matches version-X.Y.Z- badge strings in README files.
var badgePat = regexp.MustCompile(`version-[0-9]+\.[0-9]+\.[0-9]+-`)

// releaseStepResult holds outcome of one cascade step.
type releaseStepResult struct {
	Step    int    `json:"step"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "done", "failed", "skipped", "dry-run"
	Message string `json:"message,omitempty"`
	Elapsed string `json:"elapsed,omitempty"`
}

// releaseResult is the full cascade result.
type releaseResult struct {
	Version string              `json:"version"`
	Tag     string              `json:"tag"`
	DryRun  bool                `json:"dry_run"`
	Steps   []releaseStepResult `json:"steps"`
	Status  string              `json:"status"` // "success", "failed", "dry-run"
	Elapsed string              `json:"elapsed,omitempty"`
	Error   string              `json:"error,omitempty"`
}

var releaseCmd = &cobra.Command{
	Use:   "release <version>",
	Short: "Orchestrate the full release cascade for a version",
	Long: `Orchestrate the 12-step nSelf release cascade.

Steps:
  1.  git tag + push + GitHub Release (cli)
  2.  git tag + push + GitHub Release (plugins-pro)
  3.  Docker build + push (admin)
  4.  Homebrew formula update PR
  5.  Deploy ping_api via canary → promote
  6.  Verify all artifacts (curl + docker manifest + brew)
  7.  Bump web/* package.json version strings
  8.  Vercel deploy 11 web subapps
  9.  README badges sync (10 repos)
  10. SPORT regeneration + changelog PR
  11. Start 48h soak protocol
  12. Send post-release PCIs

Runs nself release-check first. Aborts on any step failure.
Use --dry-run to log all steps without executing them.

Examples:
  nself release v1.0.12
  nself release 1.0.12 --dry-run
  nself release v1.0.12 --skip-check --json`,
	Args: cobra.ExactArgs(1),
	RunE: runRelease,
}

var releaseStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running version vs latest across all release artifacts",
	Long: `Shows per-artifact version drift summary.

Artifacts checked:
  - cli        (GitHub latest release vs version.go)
  - admin      (Docker Hub :latest tag)
  - homebrew   (Formula/nself.rb version)
  - ping_api   (ping.nself.org/version endpoint)
  - web/org    (nself.org version badge)
  - vercel     (deployment header X-Nself-Version)

Status:
  fresh   — artifact matches latest GitHub release
  stale   — artifact is behind latest release
  unknown — artifact unreachable or version undetectable

Examples:
  nself release-status
  nself release-status --json`,
	RunE: runReleaseStatus,
}

var releaseRollbackCmd = &cobra.Command{
	Use:   "rollback <version> <prior-version>",
	Short: "Roll back a release to prior version",
	Long: `Reverts a release to a prior version.

Rollback actions:
  1. Homebrew formula reverted to <prior-version>
  2. ping_api env NSELF_CLI_VERSION reset to <prior-version>
  3. Docker latest tag repointed to <prior-version>
  4. Changelog entry emitted: "Rolled back v<version> → v<prior-version>"

NOTE: Git tags are NOT deleted by default (destructive — requires --delete-tags flag).
      GitHub Release is NOT deleted automatically — mark as pre-release manually.

This command does NOT roll back production deployments. Use 'nself deploy rollback' for that.

Examples:
  nself release-rollback v1.0.12 v1.0.11
  nself release-rollback 1.0.12 1.0.11 --dry-run
  nself release-rollback v1.0.12 v1.0.11 --delete-tags`,
	Args: cobra.ExactArgs(2),
	RunE: runReleaseRollback,
}

func init() {
	releaseCmd.Flags().Bool("dry-run", false, "Log all steps without executing them")
	releaseCmd.Flags().Bool("json", false, "Output results as JSON")
	releaseCmd.Flags().Bool("skip-check", false, "Skip pre-release gate (release-check)")
	releaseCmd.Flags().Bool("skip-plugins-pro", false, "Skip plugins-pro tag (use when not releasing plugins-pro)")
	releaseCmd.Flags().String("release-notes", "", "Path to release notes file (markdown)")

	releaseStatusCmd.Flags().Bool("json", false, "Output as JSON")
	releaseStatusCmd.Flags().Duration("timeout", 30*time.Second, "HTTP timeout per artifact check")

	releaseRollbackCmd.Flags().Bool("dry-run", false, "Log rollback steps without executing")
	releaseRollbackCmd.Flags().Bool("json", false, "Output as JSON")
	releaseRollbackCmd.Flags().Bool("delete-tags", false, "Also delete git tags (DESTRUCTIVE — requires explicit flag)")

	// CLI-R09: four top-level release* commands collapse into one `release`
	// group. The old spellings still work via legacy_spellings.go.
	releaseCmd.AddCommand(releaseStatusCmd)
	releaseCmd.AddCommand(releaseRollbackCmd)
	RootCmd.AddCommand(releaseCmd)
}

// ── release ──────────────────────────────────────────────────────────────────

func runRelease(cmd *cobra.Command, args []string) error {
	rawVersion := args[0]
	ver := strings.TrimPrefix(rawVersion, "v")
	tag := "v" + ver

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	jsonOut, _ := cmd.Flags().GetBool("json")
	skipCheck, _ := cmd.Flags().GetBool("skip-check")
	skipPluginsPro, _ := cmd.Flags().GetBool("skip-plugins-pro")

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
	defer cancel()

	start := time.Now()
	result := releaseResult{
		Version: ver,
		Tag:     tag,
		DryRun:  dryRun,
		Status:  "success",
	}

	if !jsonOut {
		if dryRun {
			fmt.Printf("%s DRY RUN — nself release %s\n\n", ui.C(ui.Yellow, "[DRY-RUN]"), tag)
		} else {
			fmt.Printf("%s Executing release cascade for %s\n\n", ui.C(ui.Bold, "nSelf Release"), ui.C(ui.Cyan, tag))
		}
	}

	// Pre-flight: run release-check unless --skip-check
	if !skipCheck {
		if err := runStep(ctx, &result, 0, "Pre-release gate (release-check)", dryRun, func() error {
			return runReleaseCheckStep(ctx, ver)
		}); err != nil {
			return emitResult(result, jsonOut, err)
		}
	}

	// Step 1 — CLI tag + GitHub release
	if err := runStep(ctx, &result, 1, fmt.Sprintf("git tag %s + GitHub Release (cli)", tag), dryRun, func() error {
		return execReleaseCLITag(ctx, tag)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 2 — plugins-pro tag
	if skipPluginsPro {
		result.Steps = append(result.Steps, releaseStepResult{Step: 2, Name: "git tag (plugins-pro)", Status: "skipped", Message: "--skip-plugins-pro"})
	} else {
		if err := runStep(ctx, &result, 2, fmt.Sprintf("git tag %s + GitHub Release (plugins-pro)", tag), dryRun, func() error {
			return execReleasePluginsProTag(ctx, tag)
		}); err != nil {
			return emitResult(result, jsonOut, err)
		}
	}

	// Step 3 — admin Docker build + push
	if err := runStep(ctx, &result, 3, fmt.Sprintf("Docker build + push (admin:%s)", ver), dryRun, func() error {
		return execAdminDockerBuild(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 4 — Homebrew formula PR
	if err := runStep(ctx, &result, 4, fmt.Sprintf("Homebrew formula update PR → %s", ver), dryRun, func() error {
		return execHomebrewPR(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 5 — ping_api deploy (canary → promote)
	if err := runStep(ctx, &result, 5, "Deploy ping_api (canary → promote)", dryRun, func() error {
		return execPingAPIDeploy(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 6 — verification
	if err := runStep(ctx, &result, 6, "Verify artifacts (curl + docker manifest + brew)", dryRun, func() error {
		return execArtifactVerification(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 7 — web/* version bumps
	if err := runStep(ctx, &result, 7, "Bump web/* package.json version strings", dryRun, func() error {
		return execWebVersionBumps(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 8 — Vercel deploy
	if err := runStep(ctx, &result, 8, "Vercel deploy 11 web subapps", dryRun, func() error {
		return execVercelDeploy(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 9 — README badges
	if err := runStep(ctx, &result, 9, "README badge sync (10 repos)", dryRun, func() error {
		return execReadmeBadges(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 10 — SPORT regen + changelog PR
	if err := runStep(ctx, &result, 10, "SPORT regen + changelog PR", dryRun, func() error {
		return execSPORTRegen(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 11 — 48h soak start
	if err := runStep(ctx, &result, 11, "Start 48h soak protocol", dryRun, func() error {
		return execSoakStart(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	// Step 12 — post-release PCIs
	if err := runStep(ctx, &result, 12, "Send post-release PCIs (camclaw/ummat/acamarata)", dryRun, func() error {
		return execPostReleasePCIs(ctx, ver)
	}); err != nil {
		return emitResult(result, jsonOut, err)
	}

	result.Elapsed = time.Since(start).Round(time.Second).String()
	return emitResult(result, jsonOut, nil)
}
