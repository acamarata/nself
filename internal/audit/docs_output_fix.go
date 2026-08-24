package audit

// docs_output_fix.go — report output and auto-fix for doc findings.
//
// Purpose: write a DocReport as JSON or a human summary, and apply safe whole-word auto-fixes for findings that support it, used after RunDocs in docs.go, split out for file size.
// Inputs: a DocReport and, for ApplyAutoFix, the original doc files.
// Outputs: written report files, or doc files with the flagged word safely replaced.
// Constraints: pure move from docs.go (CLI-R12 Batch E); no behaviour change.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

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
			replacement, safe := safeReplacement(f.Match)
			if !safe {
				continue
			}
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

// safeReplacement returns the neutral replacement for a banned word and
// whether an automatic rewrite is safe at all.
//
// Deletion is only safe for words that are grammatically optional in place:
// conjunctive filler ("moreover, X" -> "X"). It is NOT safe for the adjectives
// and set phrases in bannedWords, because they modify a following noun —
// deleting the word alone leaves broken prose and, in Markdown, broken markup:
//
//	"Comprehensive guide for migrations"  -> " guide for migrations"
//	"Wiki documentation is comprehensive" -> "Wiki documentation is "
//	"**Comprehensive backup strategies**" -> "** backup strategies**"  (no longer bold)
//	"The most powerful way to query"      -> "The most way to query"
//
// Those need a phrase-level rewrite with capitalization repair, which the
// scanner has no sentence model for. They stay advisory: still reported as
// findings so a human fixes them, never auto-rewritten. See the 2026-08-16
// nself-org/plugins#34 review, which caught 48 wiki files being mangled this
// way before merge.
func safeReplacement(word string) (string, bool) {
	switch strings.ToLower(word) {
	case "moreover", "furthermore", "additionally":
		return "", true // conjunctive filler: grammatically optional, safe to drop
	case "leverage":
		return "use", true // direct verb-for-verb substitution
	}
	return "", false // adjectives and set phrases: report only
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
