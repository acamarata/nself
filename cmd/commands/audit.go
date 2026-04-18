package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/audit"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Run ecosystem audits (docs, origin, etc.)",
	Long: `Run ecosystem-level audits.

Subcommands:
  nself audit docs   Quarterly documentation audit (banned words, dead links,
                     missing anchors across README, wiki, docs, SPORT, PPI, PRI)
`,
}

var auditDocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Run the quarterly documentation audit",
	Long: `Scan READMEs, wiki pages, .claude/docs/, SPORT (F01-F15), MASTER-LISTS,
FEATURES.md, PPI, and every PRI for banned words (F15), dead local links,
and missing required anchors.

Emits a JSON report plus an optional human summary. The auto-fix flag will
rewrite safe banned-word occurrences in place; callers typically run this
under git and commit the result as a PR.

Exit codes:
  0  No findings
  1  Findings present (non-fatal, report emitted)
  2  Scan failed (I/O, unreadable tree)`,
	RunE: runAuditDocs,
}

func init() {
	auditDocsCmd.Flags().String("root", "", "project root (default: current directory)")
	auditDocsCmd.Flags().String("output", "", "write JSON report to this file (default: stdout)")
	auditDocsCmd.Flags().Bool("summary", false, "print human summary to stderr")
	auditDocsCmd.Flags().String("quarter", "", "override quarter label (default: current quarter, e.g. 2026-Q3)")
	auditDocsCmd.Flags().Bool("fix", false, "apply safe auto-fixes in place (banned-word rewrites)")
	auditDocsCmd.Flags().StringSlice("scan", nil, "override scan roots (relative to root)")

	auditCmd.AddCommand(auditDocsCmd)
	RootCmd.AddCommand(auditCmd)
}

func runAuditDocs(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("root")
	outPath, _ := cmd.Flags().GetString("output")
	summary, _ := cmd.Flags().GetBool("summary")
	quarter, _ := cmd.Flags().GetString("quarter")
	fix, _ := cmd.Flags().GetBool("fix")
	scanRoots, _ := cmd.Flags().GetStringSlice("scan")

	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("audit: getwd: %w", err)
		}
		root = wd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("audit: resolve root: %w", err)
	}

	report, err := audit.RunDocs(audit.Options{
		Root:      absRoot,
		ScanRoots: scanRoots,
		Quarter:   quarter,
	})
	if err != nil {
		ui.Error(fmt.Sprintf("audit failed: %v", err))
		os.Exit(2)
	}

	var modified []string
	if fix {
		modified, err = audit.ApplyAutoFix(absRoot, report)
		if err != nil {
			ui.Error(fmt.Sprintf("auto-fix failed: %v", err))
			os.Exit(2)
		}
	}

	// JSON output — default stdout, or --output file.
	if outPath == "" {
		if err := report.WriteJSON(os.Stdout); err != nil {
			return fmt.Errorf("audit: write json: %w", err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("audit: mkdir output: %w", err)
		}
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("audit: create output: %w", err)
		}
		defer f.Close()
		if err := report.WriteJSON(f); err != nil {
			return fmt.Errorf("audit: write output: %w", err)
		}
		ui.Success(fmt.Sprintf("wrote report: %s", outPath))
	}

	if summary {
		report.WriteSummary(os.Stderr)
	}

	if fix && len(modified) > 0 {
		ui.Info(fmt.Sprintf("auto-fix applied to %d file(s)", len(modified)))
		for _, m := range modified {
			fmt.Fprintf(os.Stderr, "  %s\n", m)
		}
	}

	if report.Summary.Total > 0 {
		// Non-fatal but signal findings with exit code 1.
		os.Exit(1)
	}
	return nil
}
