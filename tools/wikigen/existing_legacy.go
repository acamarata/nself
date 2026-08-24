package main

// Purpose: extraction of hand-written prose (legacyExtras) and the summary blockquote (legacySummary) from a legacy wiki command page, backing readExisting in existing.go.
// Inputs: the raw Markdown body of an existing command page.
// Outputs: the preserved hand-written extras section and the summary text.
// Constraints: split out of existing.go as a pure move (CLI-R12); no behavior change.

import (
	"strings"
)

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
