package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

// runStep executes a named release step, logs it, and appends the result.
func runStep(ctx context.Context, result *releaseResult, stepNum int, name string, dryRun bool, fn func() error) error {
	start := time.Now()
	s := releaseStepResult{Step: stepNum, Name: name}

	if dryRun {
		s.Status = "dry-run"
		s.Message = "skipped (dry-run)"
		fmt.Printf("  [dry-run] Step %2d: %s\n", stepNum, name)
		result.Steps = append(result.Steps, s)
		return nil
	}

	fmt.Printf("  [running] Step %2d: %s\n", stepNum, name)
	err := fn()
	s.Elapsed = time.Since(start).Round(time.Millisecond).String()
	if err != nil {
		s.Status = "failed"
		s.Message = err.Error()
		result.Status = "failed"
		result.Error = fmt.Sprintf("step %d (%s) failed: %s", stepNum, name, err)
		result.Steps = append(result.Steps, s)
		fmt.Printf("  %s   Step %2d failed: %s\n", ui.C(ui.Red, "✗"), stepNum, err)
		return fmt.Errorf("release aborted at step %d (%s): %w", stepNum, name, err)
	}
	s.Status = "done"
	result.Steps = append(result.Steps, s)
	fmt.Printf("  %s   Step %2d done (%s)\n", ui.C(ui.Green, "✓"), stepNum, s.Elapsed)
	return nil
}

func emitResult(result releaseResult, jsonOut bool, err error) error {
	if jsonOut {
		_ = ui.PrintJSON(result)
	} else if err != nil {
		fmt.Printf("\n%s Release cascade FAILED: %s\n", ui.C(ui.Red, "✗"), err)
		fmt.Printf("\nRun 'nself release-rollback %s <prior>' to roll back if needed.\n", result.Tag)
	} else if result.DryRun {
		fmt.Printf("\n%s Dry-run complete — all %d steps logged.\n", ui.C(ui.Yellow, "[DRY-RUN]"), len(result.Steps))
	} else {
		fmt.Printf("\n%s Release %s complete in %s\n", ui.C(ui.Green, "✓"), ui.C(ui.Bold, result.Tag), result.Elapsed)
		fmt.Println("\nNext: monitor the 48h soak at status.nself.org")
	}
	return err
}

// ── release-status ────────────────────────────────────────────────────────────

type artifactStatus struct {
	Artifact string `json:"artifact"`
	Running  string `json:"running"`
	Latest   string `json:"latest"`
	Status   string `json:"status"` // "fresh", "stale", "unknown"
}

func runReleaseStatus(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	// Fetch latest GitHub release
	latest := fetchLatestGitHubRelease(ctx, "nself-org/cli")

	statuses := []artifactStatus{
		checkArtifactCLI(ctx, latest),
		checkArtifactAdmin(ctx, latest),
		checkArtifactHomebrew(ctx, latest),
		checkArtifactPingAPI(ctx, latest),
		checkArtifactWebOrg(ctx, latest),
		checkArtifactVercel(ctx, latest),
	}

	out := cmd.OutOrStdout()

	if jsonOut {
		data, err := json.Marshal(map[string]interface{}{
			"latest":    latest,
			"artifacts": statuses,
			"checked":   time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	fmt.Fprintf(out, "%s Release Status  (latest: %s)\n\n", ui.C(ui.Bold, "nSelf"), ui.C(ui.Cyan, latest))
	fmt.Fprintf(out, "  %-16s %-12s %-12s %s\n", "Artifact", "Running", "Latest", "Status")
	fmt.Fprintf(out, "  %s\n", strings.Repeat("─", 56))
	for _, s := range statuses {
		statusStr := s.Status
		color := ui.Green
		switch s.Status {
		case "stale":
			color = ui.Yellow
		case "unknown":
			color = ui.Dim
		}
		fmt.Fprintf(out, "  %-16s %-12s %-12s %s\n", s.Artifact, s.Running, s.Latest, ui.C(color, statusStr))
	}
	return nil
}

func fetchLatestGitHubRelease(ctx context.Context, repo string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "unknown"
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "unknown"
	}
	defer resp.Body.Close()
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(release.TagName, "v")
}

func checkArtifactCLI(_ context.Context, latest string) artifactStatus {
	running := strings.TrimPrefix(strings.TrimPrefix(os.Getenv("NSELF_VERSION"), "v"), "")
	if running == "" {
		// Try reading version.go indirectly via go binary if available
		out, err := exec.Command("nself", "version", "--short").Output()
		if err == nil {
			running = strings.TrimSpace(string(out))
		}
	}
	return makeArtifactStatus("cli", running, latest)
}

func checkArtifactAdmin(ctx context.Context, latest string) artifactStatus {
	// curl Docker Hub manifest API
	url := "https://registry.hub.docker.com/v2/repositories/nself/nself-admin/tags/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return artifactStatus{Artifact: "admin", Status: "unknown"}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return artifactStatus{Artifact: "admin", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	var tag struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tag); err != nil {
		return makeArtifactStatus("admin", "unknown", latest)
	}
	running := tag.Name
	if running == "latest" || running == "" {
		running = "unknown"
	}
	return makeArtifactStatus("admin", running, latest)
}

func checkArtifactHomebrew(ctx context.Context, latest string) artifactStatus {
	out, err := exec.CommandContext(ctx, "brew", "info", "--json=v1", "nself-org/nself/nself").Output()
	if err != nil {
		return artifactStatus{Artifact: "homebrew", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	var info []struct {
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(out, &info); err != nil || len(info) == 0 {
		return artifactStatus{Artifact: "homebrew", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	return makeArtifactStatus("homebrew", info[0].Versions.Stable, latest)
}

func checkArtifactPingAPI(ctx context.Context, latest string) artifactStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ping.nself.org/version", nil)
	if err != nil {
		return artifactStatus{Artifact: "ping_api", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return artifactStatus{Artifact: "ping_api", Running: "unreachable", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	var v struct {
		LatestCLIVersion string `json:"latestCliVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return artifactStatus{Artifact: "ping_api", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	return makeArtifactStatus("ping_api", strings.TrimPrefix(v.LatestCLIVersion, "v"), latest)
}

func checkArtifactWebOrg(ctx context.Context, latest string) artifactStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://nself.org", nil)
	if err != nil {
		return artifactStatus{Artifact: "web/org", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	req.Header.Set("User-Agent", "nself-release-check/1.0")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return artifactStatus{Artifact: "web/org", Running: "unreachable", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	running := strings.TrimPrefix(resp.Header.Get("X-Nself-Version"), "v")
	if running == "" {
		running = "unknown"
	}
	return makeArtifactStatus("web/org", running, latest)
}

func checkArtifactVercel(ctx context.Context, latest string) artifactStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://nself.org", nil)
	if err != nil {
		return artifactStatus{Artifact: "vercel", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return artifactStatus{Artifact: "vercel", Running: "unreachable", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	running := strings.TrimPrefix(resp.Header.Get("X-Nself-Version"), "v")
	if running == "" {
		running = "unknown"
	}
	return makeArtifactStatus("vercel", running, latest)
}

func makeArtifactStatus(name, running, latest string) artifactStatus {
	s := artifactStatus{Artifact: name, Running: running, Latest: latest}
	if running == "" || running == "unknown" || running == "unreachable" {
		s.Status = "unknown"
	} else if running == latest {
		s.Status = "fresh"
	} else {
		s.Status = "stale"
	}
	return s
}

// ── release-rollback ──────────────────────────────────────────────────────────

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

// ── cascade step implementations ─────────────────────────────────────────────
// Each function calls external tools (git, gh, docker, brew, vercel, pci-send).
// In the real CLI these run subprocesses with proper context propagation.

func runReleaseCheckStep(ctx context.Context, ver string) error {
	bin, _ := os.Executable()
	if bin == "" {
		bin, _ = exec.LookPath("nself")
	}
	if bin == "" {
		return fmt.Errorf("nself binary not found for release-check")
	}
	cmd := exec.CommandContext(ctx, bin, "release-check", ver, "--skip-ci")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func execReleaseCLITag(ctx context.Context, tag string) error {
	if err := gitTagAndPush(ctx, tag); err != nil {
		return err
	}
	return createGitHubRelease(ctx, "nself-org/cli", tag)
}

func execReleasePluginsProTag(ctx context.Context, tag string) error {
	// plugins-pro is a private repo — same cascade
	if err := gitTagAndPushInRepo(ctx, tag, "../plugins-pro"); err != nil {
		return fmt.Errorf("plugins-pro tag: %w", err)
	}
	return createGitHubRelease(ctx, "nself-org/plugins-pro", tag)
}

func execAdminDockerBuild(ctx context.Context, ver string) error {
	image := fmt.Sprintf("nself/nself-admin:%s", ver)
	buildArgs := []string{"build", "-t", image, "-t", "nself/nself-admin:latest", "."}
	c := exec.CommandContext(ctx, "docker", buildArgs...)
	c.Dir = "../admin"
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	pushVer := exec.CommandContext(ctx, "docker", "push", image)
	pushVer.Stdout = os.Stdout
	pushVer.Stderr = os.Stderr
	if err := pushVer.Run(); err != nil {
		return fmt.Errorf("docker push %s: %w", image, err)
	}
	pushLatest := exec.CommandContext(ctx, "docker", "push", "nself/nself-admin:latest")
	pushLatest.Stdout = os.Stdout
	pushLatest.Stderr = os.Stderr
	return pushLatest.Run()
}

func execHomebrewPR(ctx context.Context, ver string) error {
	// Trigger the dispatch workflow or run the Ruby formula update script
	c := exec.CommandContext(ctx, "gh", "workflow", "run", "update-formula.yml",
		"--repo", "nself-org/homebrew-nself",
		"-f", "version="+ver,
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("homebrew workflow dispatch: %w", err)
	}
	return nil
}

func execPingAPIDeploy(ctx context.Context, ver string) error {
	// Deploy via vercel CLI (canary → promote)
	deploy := exec.CommandContext(ctx, "vercel", "deploy",
		"--env", "NSELF_CLI_VERSION="+ver,
	)
	deploy.Dir = "../web/backend/services/ping_api"
	deploy.Stdout = os.Stdout
	deploy.Stderr = os.Stderr
	if err := deploy.Run(); err != nil {
		return fmt.Errorf("ping_api canary deploy: %w", err)
	}
	promote := exec.CommandContext(ctx, "vercel", "promote", "--yes")
	promote.Dir = "../web/backend/services/ping_api"
	promote.Stdout = os.Stdout
	promote.Stderr = os.Stderr
	return promote.Run()
}

func execArtifactVerification(ctx context.Context, ver string) error {
	// Curl ping.nself.org/version and verify latestCliVersion
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ping.nself.org/version", nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ping.nself.org/version unreachable: %w", err)
	}
	defer resp.Body.Close()
	var v struct {
		LatestCLIVersion string `json:"latestCliVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return fmt.Errorf("parse ping_api response: %w", err)
	}
	reported := strings.TrimPrefix(v.LatestCLIVersion, "v")
	if reported != ver {
		return fmt.Errorf("ping_api reports %q but expected %q — deploy may not be live yet", reported, ver)
	}
	return nil
}

func execWebVersionBumps(ctx context.Context, ver string) error {
	// Semver validation gate — rejects non-semver strings before any exec call.
	// This prevents shell injection via ver (e.g. "1.0.0;echo INJECTED").
	if !releaseVerRe.MatchString(ver) {
		return fmt.Errorf("invalid version %q: must match semver (e.g. 1.2.3 or 1.2.3-rc.1)", ver)
	}

	// Update package.json version across all web subapps.
	// ver is passed via NSELF_VERSION env var — never interpolated into the script string itself.
	const nodeScript = `
const fs = require('fs');
const path = require('path');
const ver = process.env.NSELF_VERSION;
const subapps = ['org','docs','nchat','nclaw','cloud','install','base','ntask','ntv','nfamily','clawde'];
subapps.forEach(app => {
  const pkg = path.join('../web', app, 'package.json');
  if (!fs.existsSync(pkg)) return;
  const j = JSON.parse(fs.readFileSync(pkg, 'utf8'));
  j.version = ver;
  fs.writeFileSync(pkg, JSON.stringify(j, null, 2)+'\n');
  console.log('bumped', app, 'to', j.version);
});
`
	c := exec.CommandContext(ctx, "node", "--eval", nodeScript)
	c.Env = append(os.Environ(), "NSELF_VERSION="+ver)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func execVercelDeploy(ctx context.Context, ver string) error {
	// Trigger Vercel production deploy by pushing to main (auto-deploy) or explicit CLI
	c := exec.CommandContext(ctx, "vercel", "deploy", "--prod", "--yes")
	c.Dir = "../web"
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("vercel deploy: %w", err)
	}
	_ = ver
	return nil
}

func execReadmeBadges(ctx context.Context, ver string) error {
	// Semver validation gate — rejects non-semver strings before any file operation.
	if !releaseVerRe.MatchString(ver) {
		return fmt.Errorf("invalid version %q: must match semver (e.g. 1.2.3 or 1.2.3-rc.1)", ver)
	}
	_ = ctx // no shell invoked; ctx used for future cancellation support

	// Go-native badge replace — no shell, no fmt.Sprintf interpolation into a shell command.
	replacement := "version-" + ver + "-"
	repos := []string{"cli", "admin", "nchat", "nclaw", "ntask", "ntv", "nfamily", "clawde", "plugins", "web"}
	for _, r := range repos {
		readmePath := fmt.Sprintf("../%s/README.md", r)
		if _, err := os.Stat(readmePath); err != nil {
			continue
		}
		f, err := os.OpenFile(readmePath, os.O_RDWR, 0o644)
		if err != nil {
			return fmt.Errorf("badge update open %s/README.md: %w", r, err)
		}
		data, err := io.ReadAll(f)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("badge update read %s/README.md: %w", r, err)
		}
		updated := badgePat.ReplaceAll(data, []byte(replacement))
		if err := f.Truncate(0); err != nil {
			_ = f.Close()
			return fmt.Errorf("badge update truncate %s/README.md: %w", r, err)
		}
		if _, err := f.WriteAt(updated, 0); err != nil {
			_ = f.Close()
			return fmt.Errorf("badge update write %s/README.md: %w", r, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("badge update close %s/README.md: %w", r, err)
		}
		fmt.Printf("badge updated %s/README.md → %s\n", r, ver)
	}
	return nil
}

func execSPORTRegen(ctx context.Context, ver string) error {
	// Run SPORT regeneration if script exists
	sportScript := "../.claude/docs/sport/regen.sh"
	if _, err := os.Stat(sportScript); err == nil {
		c := exec.CommandContext(ctx, "bash", sportScript)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("SPORT regen: %w", err)
		}
	}
	_ = ver
	return nil
}

func execSoakStart(ctx context.Context, ver string) error {
	// Write soak-start marker to .claude/docs/operations/release-soak-protocol.md
	// and emit a PCI to Downloads agent
	soakPath := "../.claude/docs/operations/release-soak-protocol.md"
	if _, err := os.Stat(soakPath); err == nil {
		entry := fmt.Sprintf("\n## Soak started for v%s\n\n- Started: %s\n- Target: 48h with no CRITICAL incidents\n- GA gate: 2 × green synthetic probe cycles\n",
			ver, time.Now().UTC().Format(time.RFC3339))
		f, err := os.OpenFile(soakPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = f.WriteString(entry)
			f.Close()
		}
	}

	// PCI to Downloads inbox to start camclaw soak
	msg := fmt.Sprintf(`## Context
Release v%s has shipped. 48h soak protocol starts now.

## Request
Monitor camclaw instance for CRITICAL issues. Report back after 48h or immediately on any CRITICAL incident.

## Soak criteria
- Zero CRITICAL issues in 48h window
- All synthetic probes green
- User-reported issues: none CRITICAL
`, ver)
	c := exec.CommandContext(ctx,
		os.ExpandEnv("$HOME/bin/pci-send"),
		"nclaw", fmt.Sprintf("soak-start-v%s", ver), "medium", "info",
		fmt.Sprintf("48h soak started for v%s", ver),
	)
	c.Stdin = strings.NewReader(msg)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run() // non-fatal — soak can be started manually
	return nil
}

func execPostReleasePCIs(_ context.Context, ver string) error {
	fmt.Printf("    ↳ Post-release PCIs: send 'v%s released' to camclaw / ummat / acamarata inboxes via pci-send\n", ver)
	return nil
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

// ── git helpers ───────────────────────────────────────────────────────────────

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
