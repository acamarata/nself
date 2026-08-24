package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Finding represents a single doc-audit finding.
type Finding struct {
	Category string `json:"category"` // "banned_word" | "dead_link" | "missing_file" | "stale_version" | "broken_section"
	Severity string `json:"severity"` // "low" | "medium" | "high"
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Match    string `json:"match,omitempty"`
	Message  string `json:"message"`
	AutoFix  bool   `json:"auto_fix"` // true if a safe auto-fix is available
}

// DocReport aggregates the findings from one audit run.
type DocReport struct {
	Timestamp string    `json:"timestamp"`
	Quarter   string    `json:"quarter"` // e.g. "2026-Q3"
	ScanRoots []string  `json:"scan_roots"`
	FilesScan int       `json:"files_scanned"`
	Findings  []Finding `json:"findings"`
	Summary   Summary   `json:"summary"`
}

// Summary groups findings by category.
type Summary struct {
	Total        int `json:"total"`
	BannedWords  int `json:"banned_words"`
	DeadLinks    int `json:"dead_links"`
	MissingFiles int `json:"missing_files"`
	StaleVersion int `json:"stale_version"`
	Broken       int `json:"broken_section"`
	AutoFixable  int `json:"auto_fixable"`
}

// bannedWords are words that F15 (brand spec) forbids in user-facing docs.
// Kept lowercase; matching is case-insensitive on whole words.
var bannedWords = []string{
	"powerful",
	"robust",
	"comprehensive",
	"seamlessly",
	"leverage",
	"dive into",
	"delve into",
	"state-of-the-art",
	"cutting-edge",
	"revolutionary",
	"game-changing",
	"world-class",
	"best-in-class",
	"moreover",
	"furthermore",
	"additionally",
}

// RequiredAnchors lists files that every audit expects to exist (relative to the
// project root). Absence is reported as a `missing_file` finding.
var RequiredAnchors = []string{
	".claude/CLAUDE.md",
	".claude/docs/VISION.md",
	".claude/docs/FEATURES.md",
	".claude/docs/MASTER-VERSIONS.md",
	".claude/docs/sport/F00-INDEX.md",
	".claude/docs/sport/F01-MASTER-VERSIONS.md",
	".claude/docs/sport/F02-COMMAND-INVENTORY.md",
	".claude/docs/sport/F06-BUNDLE-INVENTORY.md",
	".claude/docs/sport/F07-PRICING-TIERS.md",
	".claude/docs/sport/F15-BRAND-SPEC.md",
}

// Options controls an audit run.
type Options struct {
	Root       string   // project root (defaults to CWD)
	ScanRoots  []string // subdirs to walk; empty = default set
	Quarter    string   // "YYYY-QN"; empty = computed from time.Now
	IncludeExt []string // file extensions to scan; default {.md,.mdx}
}

// defaultScanRoots returns the subdirectories that are walked by default.
// These cover everything the sprint spec names: READMEs, wiki, docs,
// SPORT, MASTER-LISTS, FEATURES, PPI, PRI.
func defaultScanRoots() []string {
	return []string{
		".claude/docs",
		".claude/CLAUDE.md",
		"README.md",
		".github/wiki",
		".github/docs",
	}
}

// defaultExts lists the file extensions considered during a scan.
func defaultExts() []string {
	return []string{".md", ".mdx"}
}

// quarterOf returns "YYYY-QN" for t.
func quarterOf(t time.Time) string {
	q := ((int(t.Month()) - 1) / 3) + 1
	return fmt.Sprintf("%d-Q%d", t.Year(), q)
}

// RunDocs runs a full docs audit and returns the DocReport.
//
// It walks every scan root under opts.Root, inspects markdown files, and emits
// Finding records for banned words, dead links to local files, and missing
// required anchors.
func RunDocs(opts Options) (*DocReport, error) {
	root := opts.Root
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("audit: getwd: %w", err)
		}
		root = wd
	}
	scanRoots := opts.ScanRoots
	if len(scanRoots) == 0 {
		scanRoots = defaultScanRoots()
	}
	exts := opts.IncludeExt
	if len(exts) == 0 {
		exts = defaultExts()
	}
	quarter := opts.Quarter
	if quarter == "" {
		quarter = quarterOf(time.Now())
	}

	report := &DocReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Quarter:   quarter,
		ScanRoots: scanRoots,
		Findings:  []Finding{},
	}

	filesSeen := make(map[string]bool)
	for _, rel := range scanRoots {
		abs := filepath.Join(root, rel)
		info, err := os.Stat(abs)
		if err != nil {
			// Non-existent scan roots are not fatal (different repos have
			// different structures) but are noted in the report as a
			// low-severity finding.
			report.Findings = append(report.Findings, Finding{
				Category: "missing_file",
				Severity: "low",
				File:     rel,
				Message:  "scan root not present in this repo",
			})
			continue
		}

		if info.IsDir() {
			err = filepath.Walk(abs, func(path string, fi os.FileInfo, werr error) error {
				if werr != nil {
					return werr
				}
				if fi.IsDir() {
					return nil
				}
				if !hasExt(path, exts) {
					return nil
				}
				relPath, _ := filepath.Rel(root, path)
				if filesSeen[relPath] {
					return nil
				}
				filesSeen[relPath] = true
				fs, err := scanFile(root, relPath)
				if err != nil {
					return err
				}
				report.Findings = append(report.Findings, fs...)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("audit: walk %s: %w", rel, err)
			}
		} else {
			relPath, _ := filepath.Rel(root, abs)
			if filesSeen[relPath] {
				continue
			}
			filesSeen[relPath] = true
			fs, err := scanFile(root, relPath)
			if err != nil {
				return nil, err
			}
			report.Findings = append(report.Findings, fs...)
		}
	}

	// Required anchor check.
	for _, anchor := range RequiredAnchors {
		abs := filepath.Join(root, anchor)
		if _, err := os.Stat(abs); err != nil {
			report.Findings = append(report.Findings, Finding{
				Category: "missing_file",
				Severity: "high",
				File:     anchor,
				Message:  "required anchor file is missing",
			})
		}
	}

	report.FilesScan = len(filesSeen)
	report.Summary = summarize(report.Findings)

	// Stable finding order for deterministic output.
	sort.SliceStable(report.Findings, func(i, j int) bool {
		a, b := report.Findings[i], report.Findings[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Category < b.Category
	})

	return report, nil
}
