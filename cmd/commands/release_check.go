package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// releaseCheckResult holds the result of a single pre-release gate.
type releaseCheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn", "skip"
	Message string `json:"message,omitempty"`
}

var releaseCheckCmd = &cobra.Command{
	Use:   "check <version>",
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
	releaseCmd.AddCommand(releaseCheckCmd)
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
