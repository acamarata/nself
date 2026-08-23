package main

import (
	"os"
	"path/filepath"
	"strings"
)

// proseBlocks holds the human-written parts of a command page that
// regeneration must preserve.
type proseBlocks struct {
	summary     string
	description string
	examples    string
	seeAlso     string
}

// fullyRegenerated sections are rebuilt from cobra and never carried, so a
// regenerated page never shows the same content twice.
var fullyRegenerated = []string{
	"## Synopsis", "## Description", "## Examples", "## See Also", "## See also",
}

// tableRegenerated sections have their table rebuilt from cobra, but hand-written
// prose and ### subsections inside them are kept. Legacy pages put detailed
// per-subcommand reference under "## Subcommands" as ### blocks; dropping the
// whole section to avoid a duplicate table would have deleted that reference
// (47 lines on cmd-api alone).
var tableRegenerated = []string{"## Flags", "## Subcommands"}

// readExisting recovers the prose from a page that is already on disk.
//
// Two shapes are handled. Pages written by this generator carry
// BEGIN/END PROSE markers and are read back exactly. Pages that predate the
// generator have no markers, so their Description, Examples and See Also
// sections are lifted by heading, and any further hand-written sections are
// appended to the description — that is the whole point of this function:
// regenerating must not throw away prose someone wrote by hand.
//
// Legacy filenames (`db.md` rather than `cmd-db.md`) and the older commands/
// subdirectory are checked too, since GitHub Wiki flattens the namespace and
// those pages publish under the same names.
func readExisting(dir, cmd string) proseBlocks {
	var candidates []string
	for _, name := range append([]string{pageName(cmd)}, legacyPageNames(cmd)...) {
		candidates = append(candidates, filepath.Join(dir, name))
		candidates = append(candidates, filepath.Join(dir, "commands", name))
	}

	// Several commands have a page in BOTH .github/wiki/ and the older
	// .github/wiki/commands/. GitHub Wiki flattens the namespace, so those are
	// the same published page and one of them is dead weight — but which one is
	// richer varies (commands/cmd-account.md has 178 lines to the root page's
	// 97, while root cmd-generate.md has 155 to commands/'s 133). Taking the
	// first candidate found would silently drop the better page, so every
	// candidate is parsed and the one carrying the most prose wins.
	best := proseBlocks{}
	bestScore := -1

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		body := string(data)

		var blocks proseBlocks
		if strings.Contains(body, beginProse("description")) {
			blocks = proseBlocks{
				summary:     between(body, beginProse("summary"), endProse("summary")),
				description: between(body, beginProse("description"), endProse("description")),
				examples:    between(body, beginProse("examples"), endProse("examples")),
				seeAlso:     between(body, beginProse("see-also"), endProse("see-also")),
			}
		} else {
			desc := section(body, "## Description")
			examples := section(body, "## Examples")
			seeAlso := firstNonEmpty(section(body, "## See Also"), section(body, "## See also"))

			extras := legacyExtras(body)
			switch {
			case desc == "" && extras == "":
				desc = bodyAfterTitle(body)
			case desc == "":
				desc = extras
			case extras != "":
				// cmd-deploy.md is the case that proves this is needed: it has a
				// Description heading *and* 100+ further lines of hand-written
				// reference below it. Carrying only the Description section
				// would have deleted the rest.
				desc = strings.TrimSpace(desc + "\n\n" + extras)
			}
			blocks = proseBlocks{
				summary:     legacySummary(body),
				description: desc,
				examples:    examples,
				seeAlso:     seeAlso,
			}
		}

		if score := blocks.score(); score > bestScore {
			bestScore = score
			best = blocks
		}
	}

	return best
}

// score approximates how much human writing a page carries, so the richest of
// several candidate sources is the one kept. Placeholder text counts for
// nothing: a generated skeleton must never outrank a hand-written page.
func (p proseBlocks) score() int {
	n := len(p.summary) + len(p.description) + len(p.examples) + len(p.seeAlso)
	if strings.Contains(p.description, placeholderMarker) {
		n -= len(placeholderMarker)
	}
	if strings.Contains(p.examples, placeholderMarker) {
		n -= len(placeholderMarker)
	}
	return n
}

// between returns the text bounded by start and end, exclusive.
func between(body, start, end string) string {
	i := strings.Index(body, start)
	if i < 0 {
		return ""
	}
	i += len(start)
	j := strings.Index(body[i:], end)
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(body[i : i+j])
}

// isFence reports whether a trimmed line opens or closes a code fence.
func isFence(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

// section returns the body of a markdown section, from its heading to the next
// heading of the same or higher level.
//
// Fence-aware on purpose: a `# comment` line inside a ```bash block is not a
// heading. Treating it as one truncated every Examples section at its first
// comment, which would have silently deleted most examples on 90-odd pages.
func section(body, heading string) string {
	lines := strings.Split(body, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}

	var out []string
	inFence := false
	for _, l := range lines[start:] {
		t := strings.TrimSpace(l)
		if isFence(t) {
			inFence = !inFence
			out = append(out, l)
			continue
		}
		if !inFence {
			if strings.HasPrefix(t, "## ") || strings.HasPrefix(t, "# ") {
				break
			}
			// The bottom nav is regenerated, never carried.
			if strings.HasPrefix(t, "← [[Commands]]") {
				break
			}
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// firstNonEmpty returns the first argument that is not blank.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// bodyAfterTitle returns everything below the H1 title, stopping before a
// trailing See Also section (carried separately) and before the bottom nav.
func bodyAfterTitle(body string) string {
	lines := strings.Split(body, "\n")
	start := 0
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "# ") {
			start = i + 1
			break
		}
	}

	var out []string
	inFence := false
	for _, l := range lines[start:] {
		t := strings.TrimSpace(l)
		if isFence(t) {
			inFence = !inFence
			out = append(out, l)
			continue
		}
		if !inFence && (t == "## See Also" || t == "## See also" || strings.HasPrefix(t, "← [[Commands]]")) {
			break
		}
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// legacyExtras returns the hand-written sections of a legacy page that are
// neither the Description nor anything wikigen regenerates.
//
// Content before the first "## " heading is dropped: that is the summary
// blockquote, carried separately by legacySummary. Keeping it here too would
// duplicate the summary inside the description on every page.
func legacyExtras(body string) string {
	lines := strings.Split(bodyAfterTitle(body), "\n")

	const (
		modePreamble  = iota // before the first heading: the summary blockquote
		modeSkip             // inside a fully regenerated section
		modeTableOnly        // inside Flags/Subcommands: drop the table, keep prose
		modeKeep
	)

	var out []string
	mode := modePreamble
	inFence := false
	seenSummary := false

	for _, l := range lines {
		t := strings.TrimSpace(l)

		if isFence(t) {
			inFence = !inFence
			if mode == modeKeep || mode == modeTableOnly {
				out = append(out, l)
			}
			continue
		}

		if !inFence && strings.HasPrefix(t, "## ") {
			mode = modeKeep
			for _, h := range fullyRegenerated {
				if t == h {
					mode = modeSkip
					break
				}
			}
			if mode == modeKeep {
				for _, h := range tableRegenerated {
					if t == h {
						mode = modeTableOnly
						break
					}
				}
			}
			if mode != modeKeep {
				continue
			}
		}

		switch mode {
		case modePreamble:
			// The first blockquote line is the summary, carried separately.
			// Anything else above the first heading is real prose — several
			// pages put a tier warning or an intro paragraph there — so it is
			// kept rather than dropped.
			if !seenSummary && strings.HasPrefix(t, "> ") {
				seenSummary = true
				continue
			}
			if t == "" && len(out) == 0 {
				continue
			}
			out = append(out, l)
		case modeSkip:
			continue
		case modeTableOnly:
			// Drop the regenerated table itself; keep everything else.
			if !inFence && (strings.HasPrefix(t, "|") || t == "") {
				continue
			}
			out = append(out, l)
		default:
			out = append(out, l)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// legacySummary returns the one-line blockquote below the title, which is often
// richer than cobra's Short string and is worth keeping as editable prose.
func legacySummary(body string) string {
	for _, l := range strings.Split(bodyAfterTitle(body), "\n") {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "> ") {
			return strings.TrimSpace(strings.TrimPrefix(t, ">"))
		}
		return ""
	}
	return ""
}
