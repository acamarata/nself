package audit

// docs_scan.go — per-file doc scanning for banned words and broken links.
//
// Purpose: scan one doc file for banned words, broken internal links and required anchors, and summarize the results, used by RunDocs in docs.go, split out for file size.
// Inputs: a doc file path and the compiled banned-word/link regexes.
// Outputs: Finding values appended to the running DocReport.
// Constraints: pure move from docs.go (CLI-R12 Batch E); no behaviour change.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
				// Only claim auto-fixability when the rewrite is actually safe;
				// adjectives need a human. Keeping these in sync stops the
				// summary's auto-fixable count from overstating what --fix does.
				_, fixable := safeReplacement(word)
				msg := fmt.Sprintf("banned word %q (F15 brand spec)", word)
				if !fixable {
					msg += " — rewrite the phrase by hand (no safe automatic fix)"
				}
				findings = append(findings, Finding{
					Category: "banned_word",
					Severity: "medium",
					File:     relPath,
					Line:     lineNum,
					Match:    word,
					Message:  msg,
					AutoFix:  fixable,
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
