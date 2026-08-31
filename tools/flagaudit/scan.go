// Purpose:     find every `nself <command path> ... --flag` invocation
//
//	written in the wiki or in shell scripts, so audit.go can
//	check each flag against what the binary actually registers.
//
// Inputs:      .github/wiki/*.md and scripts/**/*.sh.
// Outputs:     one invocation per matched command line.
// Constraints: only scans code, never prose. Wiki prose talks *about* the
//
//	CLI in plain English ("nself was designed to...", "the build
//	nself downloads..."), and a bare word-boundary match on
//	"nself" over raw text drowns real invocations in hundreds of
//	false positives. So for .md files this only looks inside
//	fenced ```/~~~ code blocks and inline `single-backtick`
//	spans — the two places a wiki page ever shows a real command
//	— and skips prose between them entirely. For .sh files the
//	whole file is already code; only full-line `#` comments are
//	skipped. Within a code segment, "nself" still must be
//	followed by a space to match (so "nself.org" inside a code
//	block, e.g. a curl URL, is not mistaken for the command). A
//	real invocation nested inside another tool's example — e.g.
//	the `ssh host 'nself start --wait-healthy'` pattern used by
//	the soak scripts — is deliberately still caught: the
//	wrapping ssh/docker/curl command is irrelevant, the nested
//	nself command is exactly the kind of drift this gate exists
//	to catch. Flags are resolved from the deepest command-path
//	segment recognised before the first flag or unrecognised
//	token, so `nself ssl setup --wildcard` is checked against
//	`ssl setup`, not `ssl`.
package main

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// invocation is one `nself ...` command line found in a doc or script.
type invocation struct {
	File       string
	Line       int
	Raw        string   // the matched text, for error messages
	PathTokens []string // command-path words, in order, before the first flag
	Flags      []string // flag tokens (without leading dashes), in order
}

// invocationRe finds "nself" bounded on the left by start-of-string or one of
// the listed punctuation/whitespace characters, and on the right by
// whitespace — so it lands on the command word, not a domain like
// "nself.org" or a hyphenated name like "homebrew-nself".
var invocationRe = regexp.MustCompile("(?:^|[\\s`'\"({])nself\\s+(.*)$")

// inlineCodeRe extracts the contents of every `single-backtick` span on a
// markdown line — the only place outside a fenced block a wiki page shows a
// real command.
var inlineCodeRe = regexp.MustCompile("`([^`]+)`")

// fenceRe matches a fenced-code-block delimiter line (``` or ~~~, with an
// optional language tag), used to toggle in/out of code context.
var fenceRe = regexp.MustCompile("^(```|~~~)")

// cutAtRe truncates a matched remainder at the first point where what
// follows is no longer part of the same nself invocation:
//
//   - shell control operators (| ; &&) — a pipeline or chained command,
//     e.g. `nself status --json | grep healthy`.
//   - a command substitution start ($() — whatever it invokes is a
//     different program, e.g. `nself completion bash > "$(brew --prefix)..."`.
//   - output redirection (>) — same reasoning.
//   - a stray backtick — closes an inline code span from the markdown
//     source that leaked into a .sh file's own echoed markdown output
//     (scripts/sport/generate-core-services.sh echoes wiki prose verbatim).
var cutAtRe = regexp.MustCompile("[|;`>]|&&|\\$\\(")

// trimPunct strips punctuation that prose or embedded markdown commonly
// wraps a code span in, and that would otherwise stick to the last token
// (closing quote, paren, backtick, an escaping backslash left behind when
// cutAtRe truncates right after a `\“ a shell script used to escape a
// literal backtick, sentence punctuation).
const trimPunct = "`'\")(,.;:\\"

// pathTokenRe restricts what a bare word may look like to count as a
// command-path segment. A real nself subcommand or plugin command name is
// short and alphanumeric-with-hyphens; a token containing shell metacharacters,
// slashes, or angle brackets is virtually always scanner noise — a
// redirection operator (`2>&1`), a filesystem path (`/usr/local/bin/nself`),
// or a doc placeholder (`<slug>`) that happened to land after the word
// "nself" — not an attempted command word. Rejecting those up front keeps
// the informational skip list (for genuinely plugin-provided commands like
// `region`/`alerts`/`dr`) readable instead of drowned in garbage.
var pathTokenRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*$`)

// scanFile extracts every nself invocation found in code context: fenced
// code blocks and inline spans for markdown, non-comment lines for shell.
func scanFile(path string) ([]invocation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var out []invocation
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	isShell := strings.HasSuffix(path, ".sh")
	isMD := strings.HasSuffix(path, ".md")
	inFence := false

	lineNo := 0

	for scanner.Scan() {
		lineNo++
		text := scanner.Text()

		if isMD && fenceRe.MatchString(strings.TrimSpace(text)) {
			inFence = !inFence
			continue
		}

		if isShell && strings.HasPrefix(strings.TrimSpace(text), "#") {
			continue
		}

		for _, seg := range codeSegments(text, isMD, inFence) {
			if inv, ok := parseInvocation(path, lineNo, seg); ok {
				out = append(out, inv)
			}
		}
	}
	return out, scanner.Err()
}

// codeSegments returns the pieces of a line that are actual code, given the
// file type and current fence state. A shell file (or a markdown line
// inside a fence) is code end to end; a markdown line outside a fence only
// contributes its inline `code` spans, so surrounding prose is never
// scanned for a false-positive "nself" mention.
func codeSegments(line string, isMD, inFence bool) []string {
	if !isMD || inFence {
		return []string{line}
	}
	matches := inlineCodeRe.FindAllStringSubmatch(line, -1)
	if matches == nil {
		return nil
	}
	segs := make([]string, 0, len(matches))
	for _, m := range matches {
		segs = append(segs, m[1])
	}
	return segs
}

// parseInvocation finds and decomposes one nself invocation on a single
// (already-joined) logical line. It returns ok=false when the line mentions
// "nself" but not as an invocable command (e.g. a bare domain).
func parseInvocation(file string, line int, text string) (invocation, bool) {
	m := invocationRe.FindStringSubmatch(text)
	if m == nil {
		return invocation{}, false
	}
	rest := m[1]
	if loc := cutAtRe.FindStringIndex(rest); loc != nil {
		rest = rest[:loc[0]]
	}

	fields := strings.Fields(rest)
	inv := invocation{File: file, Line: line, Raw: "nself " + rest}

	seenFlag := false
	for _, tok := range fields {
		tok = strings.Trim(tok, trimPunct)
		if tok == "" {
			continue
		}
		if strings.HasPrefix(tok, "--") || (strings.HasPrefix(tok, "-") && len(tok) > 1) {
			seenFlag = true
			name := strings.TrimLeft(tok, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			}
			// A bare "--" (POSIX end-of-options marker, e.g.
			// `nself exec postgres -- pg_dump mydb`) or a run of dashes used
			// as a prose/log separator (`echo "--- nself version ---"`)
			// leaves nothing alphanumeric after stripping leading dashes —
			// neither is a flag.
			if name == "" || strings.Trim(name, "-") == "" {
				continue
			}
			inv.Flags = append(inv.Flags, name)
			continue
		}
		if !seenFlag && pathTokenRe.MatchString(tok) {
			inv.PathTokens = append(inv.PathTokens, tok)
			continue
		}
		// A positional argument after the path but before/between flags
		// (e.g. the table name in `migrate apply --rls-force np_chat_messages`,
		// or the path in `license verify --offline ~/.nself/license/cache.json`),
		// or a bare word that doesn't look like a command segment at all
		// (a redirect target, a filesystem path, a doc placeholder) — is
		// neither a path segment nor a flag. Skip it silently, and stop
		// treating anything further as path prefix even if it does look
		// like a word, since we've already broken out of the command path.
		seenFlag = true
	}

	if len(inv.PathTokens) == 0 && len(inv.Flags) == 0 {
		return invocation{}, false
	}
	return inv, true
}
