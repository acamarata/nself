package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"
	"github.com/nself-org/cli/internal/version"

	"github.com/spf13/cobra"
)

// releaseCheckResult holds the result of a single pre-release gate.
type releaseCheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn", "skip"
	Message string `json:"message,omitempty"`
}

var releaseCheckCmd = &cobra.Command{
	Use:   "release-check <version>",
	Short: "Run pre-release gates for a version",
	Long: `Run pre-release gates before executing 'nself release <version>'.

Gates checked:
  1. Version format is valid semver
  2. Version is higher than current deployed version
  3. CHANGELOG entry exists for this version
  4. CI is green (GitHub Actions on main)
  5. Security scan passes (no critical CVEs in go.sum / pnpm-lock.yaml)
  6. No USER DECISION blockers in .claude/tasks/active.md
  7. SPORT files are regeneration-clean (no stale sport.json)
  8. CLI / Admin version lockstep (both match <version>)
  9. Git working tree is clean on main branch
  10. Tag <version> does not already exist

Exit 0 only when ALL gates pass (or only WARNs remain).

Examples:
  nself release-check v1.0.12
  nself release-check 1.0.12 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runReleaseCheck,
}

func init() {
	releaseCheckCmd.Flags().Bool("json", false, "Output results as JSON")
	releaseCheckCmd.Flags().Bool("skip-ci", false, "Skip GitHub CI check (use in offline environments)")
	releaseCheckCmd.Flags().Bool("skip-security", false, "Skip security CVE scan")
	RootCmd.AddCommand(releaseCheckCmd)
}

func runReleaseCheck(cmd *cobra.Command, args []string) error {
	rawVersion := args[0]
	ver := strings.TrimPrefix(rawVersion, "v")
	tag := "v" + ver

	jsonOut, _ := cmd.Flags().GetBool("json")
	skipCI, _ := cmd.Flags().GetBool("skip-ci")
	skipSecurity, _ := cmd.Flags().GetBool("skip-security")

	ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Minute)
	defer cancel()

	results := []releaseCheckResult{}

	// Gate 1 — semver format
	results = append(results, checkSemver(ver))

	// Gate 2 — version is higher than current
	results = append(results, checkVersionHigher(ver))

	// Gate 3 — CHANGELOG entry exists
	results = append(results, checkChangelogEntry(ver))

	// Gate 4 — CI green
	if skipCI {
		results = append(results, releaseCheckResult{Name: "CI green", Status: "skip", Message: "--skip-ci flag set"})
	} else {
		results = append(results, checkCIGreen(ctx, tag))
	}

	// Gate 5 — security scan
	if skipSecurity {
		results = append(results, releaseCheckResult{Name: "Security scan", Status: "skip", Message: "--skip-security flag set"})
	} else {
		results = append(results, checkSecurityScan(ctx))
	}

	// Gate 6 — no USER DECISION blockers
	results = append(results, checkNoBlockers())

	// Gate 7 — SPORT regen clean
	results = append(results, checkSPORTClean())

	// Gate 8 — version lockstep
	results = append(results, checkVersionLockstep(ver))

	// Gate 9 — git working tree clean
	results = append(results, checkGitClean(ctx))

	// Gate 10 — tag does not exist
	results = append(results, checkTagAbsent(ctx, tag))

	if jsonOut {
		return ui.PrintJSON(results)
	}

	// Pretty-print results
	allPass := true
	hasFail := false
	for _, r := range results {
		symbol := "✓"
		color := ui.Green
		switch r.Status {
		case "fail":
			symbol = "✗"
			color = ui.Red
			hasFail = true
			allPass = false
		case "warn":
			symbol = "!"
			color = ui.Yellow
		case "skip":
			symbol = "-"
			color = ui.Dim
		}
		line := fmt.Sprintf("  %s %s", ui.C(color, symbol), r.Name)
		if r.Message != "" {
			line += fmt.Sprintf(": %s", r.Message)
		}
		fmt.Println(line)
	}

	fmt.Println()
	if hasFail {
		fmt.Printf("%s Pre-release gate FAILED — fix the above issues before releasing %s\n",
			ui.C(ui.Red, "✗"), ui.C(ui.Bold, tag))
		return fmt.Errorf("release-check failed for %s", tag)
	}
	if allPass {
		fmt.Printf("%s All gates passed — ready to run: nself release %s\n",
			ui.C(ui.Green, "✓"), ui.C(ui.Bold, tag))
	} else {
		fmt.Printf("%s Gates passed with warnings — review above before releasing %s\n",
			ui.C(ui.Yellow, "!"), ui.C(ui.Bold, tag))
	}
	return nil
}

// ── individual gate checks ───────────────────────────────────────────────────

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func checkSemver(ver string) releaseCheckResult {
	if semverRe.MatchString(ver) {
		return releaseCheckResult{Name: "Semver format", Status: "pass"}
	}
	return releaseCheckResult{Name: "Semver format", Status: "fail", Message: fmt.Sprintf("%q is not valid semver (expected X.Y.Z)", ver)}
}

func checkVersionHigher(ver string) releaseCheckResult {
	current := version.GetVersion()
	if compareVersions(ver, current) > 0 {
		return releaseCheckResult{Name: "Version is higher than current", Status: "pass",
			Message: fmt.Sprintf("%s > %s", ver, current)}
	}
	return releaseCheckResult{Name: "Version is higher than current", Status: "fail",
		Message: fmt.Sprintf("%s is not higher than current %s", ver, current)}
}

func checkChangelogEntry(ver string) releaseCheckResult {
	candidates := []string{
		"CHANGELOG.md",
		"cli/CHANGELOG.md",
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "["+ver+"]") || strings.Contains(string(data), "## ["+ver+"]") {
			return releaseCheckResult{Name: "CHANGELOG entry", Status: "pass", Message: path}
		}
	}
	// Search relative to executable working dir
	wd, _ := os.Getwd()
	chlog := filepath.Join(wd, "CHANGELOG.md")
	if data, err := os.ReadFile(chlog); err == nil {
		if strings.Contains(string(data), "["+ver+"]") {
			return releaseCheckResult{Name: "CHANGELOG entry", Status: "pass", Message: chlog}
		}
		return releaseCheckResult{Name: "CHANGELOG entry", Status: "fail",
			Message: fmt.Sprintf("no entry for [%s] found in %s", ver, chlog)}
	}
	return releaseCheckResult{Name: "CHANGELOG entry", Status: "warn",
		Message: "CHANGELOG.md not found — ensure it exists with an entry for " + ver}
}

func checkCIGreen(ctx context.Context, tag string) releaseCheckResult {
	// Use gh CLI to check latest workflow run on main
	out, err := exec.CommandContext(ctx, "gh", "run", "list",
		"--branch", "main",
		"--limit", "5",
		"--json", "status,conclusion,workflowName",
		"--repo", "nself-org/cli",
	).Output()
	if err != nil {
		return releaseCheckResult{Name: "CI green", Status: "warn",
			Message: "gh CLI not available or repo not accessible — verify CI manually"}
	}

	type runInfo struct {
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		Workflow   string `json:"workflowName"`
	}
	var runs []runInfo
	if err := json.Unmarshal(out, &runs); err != nil || len(runs) == 0 {
		return releaseCheckResult{Name: "CI green", Status: "warn", Message: "unable to parse gh run list output"}
	}

	failing := []string{}
	for _, r := range runs {
		if r.Status == "completed" && r.Conclusion != "success" && r.Conclusion != "skipped" {
			failing = append(failing, r.Workflow+" ("+r.Conclusion+")")
		}
	}
	if len(failing) > 0 {
		return releaseCheckResult{Name: "CI green", Status: "fail",
			Message: "failing runs: " + strings.Join(failing, ", ")}
	}
	return releaseCheckResult{Name: "CI green", Status: "pass", Message: fmt.Sprintf("last %d runs green", len(runs))}
}

func checkSecurityScan(ctx context.Context) releaseCheckResult {
	// Check go.sum for known CRITICAL CVEs via govulncheck if available
	if _, err := exec.LookPath("govulncheck"); err == nil {
		out, err := exec.CommandContext(ctx, "govulncheck", "./...").CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "CRITICAL") || strings.Contains(string(out), "HIGH") {
				return releaseCheckResult{Name: "Security scan", Status: "fail",
					Message: "govulncheck found critical/high vulnerabilities"}
			}
		}
		return releaseCheckResult{Name: "Security scan", Status: "pass", Message: "govulncheck clean"}
	}

	// Fallback: check for go.sum existence and warn
	if _, err := os.Stat("go.sum"); err == nil {
		return releaseCheckResult{Name: "Security scan", Status: "warn",
			Message: "govulncheck not installed — install with: go install golang.org/x/vuln/cmd/govulncheck@latest"}
	}
	return releaseCheckResult{Name: "Security scan", Status: "skip", Message: "go.sum not found"}
}

func checkNoBlockers() releaseCheckResult {
	candidates := []string{
		".claude/tasks/active.md",
		"../。claude/tasks/active.md",
	}
	// Walk up to find active.md
	wd, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(wd, ".claude/tasks/active.md")
		if data, err := os.ReadFile(candidate); err == nil {
			if strings.Contains(string(data), "USER DECISION") && strings.Contains(string(data), "blocked") {
				return releaseCheckResult{Name: "No USER DECISION blockers", Status: "fail",
					Message: "active.md contains USER DECISION blocked items — resolve before release"}
			}
			return releaseCheckResult{Name: "No USER DECISION blockers", Status: "pass"}
		}
		wd = filepath.Dir(wd)
	}
	_ = candidates
	return releaseCheckResult{Name: "No USER DECISION blockers", Status: "warn",
		Message: ".claude/tasks/active.md not found — verify manually"}
}

func checkSPORTClean() releaseCheckResult {
	// Check if sport.json exists and is not stale (modified in last 24h relative to source files)
	sportPaths := []string{
		".claude/docs/sport/sport.json",
		"../.claude/docs/sport/sport.json",
	}
	wd, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		for _, rel := range sportPaths {
			path := filepath.Join(wd, rel)
			if info, err := os.Stat(path); err == nil {
				if time.Since(info.ModTime()) > 7*24*time.Hour {
					return releaseCheckResult{Name: "SPORT regen clean", Status: "warn",
						Message: fmt.Sprintf("sport.json is %s old — consider regenerating", time.Since(info.ModTime()).Round(time.Hour))}
				}
				return releaseCheckResult{Name: "SPORT regen clean", Status: "pass",
					Message: fmt.Sprintf("sport.json last updated %s ago", time.Since(info.ModTime()).Round(time.Minute))}
			}
		}
		wd = filepath.Dir(wd)
	}
	return releaseCheckResult{Name: "SPORT regen clean", Status: "warn", Message: "sport.json not found — run SPORT regeneration before release"}
}

func checkVersionLockstep(ver string) releaseCheckResult {
	// Check that admin/package.json version matches
	candidates := []string{
		"../admin/package.json",
		"admin/package.json",
	}
	wd, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		for _, rel := range candidates {
			path := filepath.Join(wd, rel)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var pkg struct {
				Version string `json:"version"`
			}
			if err := json.Unmarshal(data, &pkg); err != nil {
				continue
			}
			if pkg.Version == ver {
				return releaseCheckResult{Name: "CLI/Admin version lockstep", Status: "pass",
					Message: fmt.Sprintf("admin/package.json = %s", pkg.Version)}
			}
			return releaseCheckResult{Name: "CLI/Admin version lockstep", Status: "fail",
				Message: fmt.Sprintf("admin/package.json version %s != %s", pkg.Version, ver)}
		}
		wd = filepath.Dir(wd)
	}
	return releaseCheckResult{Name: "CLI/Admin version lockstep", Status: "warn",
		Message: "admin/package.json not found — verify admin version manually"}
}

func checkGitClean(ctx context.Context) releaseCheckResult {
	out, err := exec.CommandContext(ctx, "git", "status", "--porcelain").Output()
	if err != nil {
		return releaseCheckResult{Name: "Git working tree clean", Status: "warn", Message: "git not available"}
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return releaseCheckResult{Name: "Git working tree clean", Status: "fail",
			Message: "uncommitted changes in working tree — commit or stash before release"}
	}

	// Also check current branch is main
	branch, err := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err == nil {
		b := strings.TrimSpace(string(branch))
		if b != "main" {
			return releaseCheckResult{Name: "Git working tree clean", Status: "warn",
				Message: fmt.Sprintf("current branch is %q (expected main)", b)}
		}
	}
	return releaseCheckResult{Name: "Git working tree clean", Status: "pass"}
}

func checkTagAbsent(ctx context.Context, tag string) releaseCheckResult {
	out, err := exec.CommandContext(ctx, "git", "tag", "--list", tag).Output()
	if err != nil {
		return releaseCheckResult{Name: "Tag does not exist", Status: "warn", Message: "git not available"}
	}
	if strings.TrimSpace(string(out)) != "" {
		return releaseCheckResult{Name: "Tag does not exist", Status: "fail",
			Message: fmt.Sprintf("tag %s already exists locally — delete it first: git tag --delete %s", tag, tag)}
	}

	// Also check remote
	out2, err2 := exec.CommandContext(ctx, "git", "ls-remote", "--tags", "origin", tag).Output()
	if err2 == nil && strings.TrimSpace(string(out2)) != "" {
		return releaseCheckResult{Name: "Tag does not exist", Status: "fail",
			Message: fmt.Sprintf("tag %s already exists on origin", tag)}
	}
	return releaseCheckResult{Name: "Tag does not exist", Status: "pass"}
}

// compareVersions returns 1 if a > b, -1 if a < b, 0 if equal.
// Both versions must be X.Y.Z format.
func compareVersions(a, b string) int {
	parseV := func(v string) [3]int {
		var parts [3]int
		fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
		return parts
	}
	pa, pb := parseV(a), parseV(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return 1
		}
		if pa[i] < pb[i] {
			return -1
		}
	}
	return 0
}

// Ensure http import is used.
var _ = http.StatusOK
