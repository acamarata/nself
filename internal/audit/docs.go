package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Finding represents a single doc-audit finding.
type Finding struct {
	Category string `json:"category"`           // "banned_word" | "dead_link" | "missing_file" | "stale_version" | "broken_section"
	Severity string `json:"severity"`           // "low" | "medium" | "high"
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Match    string `json:"match,omitempty"`
	Message  string `json:"message"`
	AutoFix  bool   `json:"auto_fix"`           // true if a safe auto-fix is available
}

// DocReport aggregates the findings from one audit run.
type DocReport struct {
	Timestamp   string    `json:"timestamp"`
	Quarter     string    `json:"quarter"`     // e.g. "2026-Q3"
	ScanRoots   []string  `json:"scan_roots"`
	FilesScan   int       `json:"files_scanned"`
	Findings    []Finding `json:"findings"`
	Summary     Summary   `json:"summary"`
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

// hasExt reports whether path has one of the given extensions.
func hasExt(path string, exts []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

// linkPattern matches standard markdown links: [text](target).
var linkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)

// wordBoundary helper — build a case-insensitive regex for each banned phrase.
// Phrases can contain hyphens or spaces, so \b isn't always safe; use lookaround-ish
// approximation with (^|[^\p{L}])phrase([^\p{L}]|$).
var bannedRegex = buildBannedRegex(bannedWords)

func buildBannedRegex(words []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(words))
	for _, w := range words {
		esc := regexp.QuoteMeta(w)
		pat := fmt.Sprintf(`(?i)(^|[^A-Za-z0-9_-])(%s)([^A-Za-z0-9_-]|$)`, esc)
		out = append(out, regexp.MustCompile(pat))
	}
	return out
}

// scanFile inspects a single markdown file for findings.
func scanFile(root, relPath string) ([]Finding, error) {
	abs := filepath.Join(root, relPath)
	f, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("audit: open %s: %w", relPath, err)
	}
	defer f.Close()

	var findings []Finding
	scanner := bufio.NewScanner(f)
	// Allow long lines (wiki pages can have dense tables).
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	lineNum := 0
	inCodeBlock := false
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Track triple-backtick code fences so we don't flag examples.
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Banned-word scan.
		for i, re := range bannedRegex {
			if m := re.FindStringSubmatchIndex(line); m != nil {
				word := bannedWords[i]
				findings = append(findings, Finding{
					Category: "banned_word",
					Severity: "medium",
					File:     relPath,
					Line:     lineNum,
					Match:    word,
					Message:  fmt.Sprintf("banned word %q (F15 brand spec)", word),
					AutoFix:  true,
				})
			}
		}

		// Dead-link scan (local files only; external URLs left to a later pass).
		for _, m := range linkPattern.FindAllStringSubmatch(line, -1) {
			if len(m) < 2 {
				continue
			}
			target := strings.TrimSpace(m[1])
			if target == "" {
				continue
			}
			if isExternalLink(target) {
				continue
			}
			// Strip anchor + query.
			clean := target
			if i := strings.IndexAny(clean, "?#"); i >= 0 {
				clean = clean[:i]
			}
			if clean == "" {
				continue
			}
			var candidate string
			if filepath.IsAbs(clean) {
				candidate = clean
			} else {
				candidate = filepath.Join(filepath.Dir(abs), clean)
			}
			if _, err := os.Stat(candidate); err != nil {
				findings = append(findings, Finding{
					Category: "dead_link",
					Severity: "medium",
					File:     relPath,
					Line:     lineNum,
					Match:    target,
					Message:  fmt.Sprintf("link target %q does not exist on disk", target),
					AutoFix:  true, // a later pass can remove or flag in a PR
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("audit: read %s: %w", relPath, err)
	}
	return findings, nil
}

// isExternalLink reports whether target is a network link or anchor-only link
// that the scanner should skip.
func isExternalLink(target string) bool {
	if target == "" {
		return true
	}
	if strings.HasPrefix(target, "#") {
		return true
	}
	if strings.HasPrefix(target, "mailto:") {
		return true
	}
	lower := strings.ToLower(target)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	// Wikilinks [[Page]] — detected upstream, but guard here anyway.
	if strings.HasPrefix(target, "[[") && strings.HasSuffix(target, "]]") {
		return true
	}
	return false
}

// summarize tallies findings into the Summary struct.
func summarize(fs []Finding) Summary {
	s := Summary{Total: len(fs)}
	for _, f := range fs {
		switch f.Category {
		case "banned_word":
			s.BannedWords++
		case "dead_link":
			s.DeadLinks++
		case "missing_file":
			s.MissingFiles++
		case "stale_version":
			s.StaleVersion++
		case "broken_section":
			s.Broken++
		}
		if f.AutoFix {
			s.AutoFixable++
		}
	}
	return s
}

// WriteJSON marshals the report as pretty-printed JSON to w.
func (r *DocReport) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteSummary prints a human-readable summary to w.
func (r *DocReport) WriteSummary(w io.Writer) {
	fmt.Fprintf(w, "Quarterly doc audit — %s\n", r.Quarter)
	fmt.Fprintf(w, "Timestamp: %s\n", r.Timestamp)
	fmt.Fprintf(w, "Files scanned: %d\n", r.FilesScan)
	fmt.Fprintf(w, "Findings: %d (banned_words=%d, dead_links=%d, missing=%d, stale_version=%d, broken=%d, auto-fixable=%d)\n",
		r.Summary.Total,
		r.Summary.BannedWords,
		r.Summary.DeadLinks,
		r.Summary.MissingFiles,
		r.Summary.StaleVersion,
		r.Summary.Broken,
		r.Summary.AutoFixable,
	)
}

// ApplyAutoFix rewrites banned words in-place to neutral alternatives.
// Only banned_word and dead_link findings marked AutoFix are applied.
//
// Returns the set of files that were modified. On any write error, the
// partial changes that succeeded remain in place (caller should pair
// this with git to revert on failure).
func ApplyAutoFix(root string, report *DocReport) ([]string, error) {
	byFile := make(map[string][]Finding)
	for _, f := range report.Findings {
		if !f.AutoFix {
			continue
		}
		if f.Category != "banned_word" {
			// Dead-link removal is risky: leave it for the PR reviewer.
			continue
		}
		byFile[f.File] = append(byFile[f.File], f)
	}

	modified := make([]string, 0, len(byFile))
	for rel, fs := range byFile {
		abs := filepath.Join(root, rel)
		data, err := os.ReadFile(abs)
		if err != nil {
			return modified, fmt.Errorf("audit: read %s: %w", rel, err)
		}
		orig := string(data)
		updated := orig
		for _, f := range fs {
			replacement := safeReplacement(f.Match)
			updated = replaceWholeWord(updated, f.Match, replacement)
		}
		if updated == orig {
			continue
		}
		if err := os.WriteFile(abs, []byte(updated), 0o644); err != nil {
			return modified, fmt.Errorf("audit: write %s: %w", rel, err)
		}
		modified = append(modified, rel)
	}
	sort.Strings(modified)
	return modified, nil
}

// safeReplacement returns the neutral replacement for a banned word.
// The CLI treats deletion as safer than substitution to avoid tone drift;
// the result is a single space so sentence structure stays intact.
func safeReplacement(word string) string {
	switch strings.ToLower(word) {
	case "moreover", "furthermore", "additionally":
		return "" // drop conjunctive filler
	case "leverage":
		return "use"
	}
	return ""
}

// replaceWholeWord replaces occurrences of word in src, honoring
// non-alphanumeric boundaries, with replacement. Case-insensitive.
func replaceWholeWord(src, word, replacement string) string {
	esc := regexp.QuoteMeta(word)
	pat := regexp.MustCompile(fmt.Sprintf(`(?i)(^|[^A-Za-z0-9_-])(%s)([^A-Za-z0-9_-]|$)`, esc))
	return pat.ReplaceAllStringFunc(src, func(match string) string {
		m := pat.FindStringSubmatch(match)
		if len(m) < 4 {
			return match
		}
		left := m[1]
		right := m[3]
		if replacement == "" {
			// Collapse whitespace so dropping "moreover, " leaves a clean line.
			if left == " " && right == " " {
				return " "
			}
			return left + right
		}
		return left + replacement + right
	})
}
