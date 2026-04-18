package audit

import (
	"fmt"
	"sort"
	"strings"
)

// PRDescription returns a human-readable PR body describing the auto-fix run.
//
// The CI workflow uses this when opening a "quarterly doc audit" PR. It
// summarizes the finding counts, lists the modified files, and links to the
// full JSON report stored in .claude/qa/findings/.
func PRDescription(report *DocReport, modified []string, reportPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Quarterly doc audit — %s\n\n", report.Quarter)
	fmt.Fprintf(&b, "Generated %s. Scan covered %d file(s).\n\n", report.Timestamp, report.FilesScan)

	fmt.Fprintf(&b, "## Findings\n\n")
	fmt.Fprintf(&b, "| Category | Count |\n|---|---|\n")
	fmt.Fprintf(&b, "| banned_word | %d |\n", report.Summary.BannedWords)
	fmt.Fprintf(&b, "| dead_link | %d |\n", report.Summary.DeadLinks)
	fmt.Fprintf(&b, "| missing_file | %d |\n", report.Summary.MissingFiles)
	fmt.Fprintf(&b, "| stale_version | %d |\n", report.Summary.StaleVersion)
	fmt.Fprintf(&b, "| broken_section | %d |\n", report.Summary.Broken)
	fmt.Fprintf(&b, "| **total** | **%d** |\n\n", report.Summary.Total)

	if len(modified) > 0 {
		sorted := append([]string(nil), modified...)
		sort.Strings(sorted)
		fmt.Fprintf(&b, "## Files modified by auto-fix (%d)\n\n", len(sorted))
		for _, m := range sorted {
			fmt.Fprintf(&b, "- `%s`\n", m)
		}
		b.WriteString("\n")
	} else {
		b.WriteString("## Files modified by auto-fix\n\nNone.\n\n")
	}

	if report.Summary.DeadLinks > 0 || report.Summary.MissingFiles > 0 {
		b.WriteString("## Manual review required\n\n")
		if report.Summary.DeadLinks > 0 {
			fmt.Fprintf(&b, "- %d dead link(s) left for a reviewer to resolve.\n", report.Summary.DeadLinks)
		}
		if report.Summary.MissingFiles > 0 {
			fmt.Fprintf(&b, "- %d missing anchor file(s) — file a follow-up ticket.\n", report.Summary.MissingFiles)
		}
		b.WriteString("\n")
	}

	if reportPath != "" {
		fmt.Fprintf(&b, "Full JSON report: `%s`\n", reportPath)
	}
	return b.String()
}

// IsSafeAutoFix reports whether f is eligible for silent rewrite in a PR.
// Currently only banned-word fixes are treated as silent-safe.
func IsSafeAutoFix(f Finding) bool {
	return f.AutoFix && f.Category == "banned_word"
}
