package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// sidebarBegin and sidebarEnd delimit the generated command list inside
// _Sidebar.md. Everything outside the markers is hand-curated and untouched.
const (
	sidebarBegin = "<!-- BEGIN GENERATED:command-list -->"
	sidebarEnd   = "<!-- END GENERATED:command-list -->"
)

// writeSidebar replaces the generated block in _Sidebar.md with a complete,
// alphabetised index of every command page.
//
// The hand-curated grouped lists above it stay: they are the useful reading
// order. This block is the completeness guarantee — before it existed, 69 of
// 92 commands had no route from the sidebar to their page at all.
func writeSidebar(path string, cmds []*cobra.Command, check bool) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read sidebar: %w", err)
	}
	body := string(data)

	block := renderSidebarBlock(cmds)

	var updated string
	switch {
	case strings.Contains(body, sidebarBegin) && strings.Contains(body, sidebarEnd):
		start := strings.Index(body, sidebarBegin)
		end := strings.Index(body, sidebarEnd) + len(sidebarEnd)
		updated = body[:start] + block + body[end:]
	default:
		updated = strings.TrimRight(body, "\n") + "\n\n---\n\n" + block + "\n"
	}

	if updated == body {
		return false, nil
	}
	if check {
		return true, nil
	}
	return true, os.WriteFile(path, []byte(updated), 0o644)
}

func renderSidebarBlock(cmds []*cobra.Command) string {
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.Name())
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString(sidebarBegin + "\n")
	b.WriteString("\n**All commands (" + fmt.Sprint(len(names)) + ")**\n\n")

	// Grouped by first letter so the list stays scannable at 90-odd entries.
	var current byte
	var line []string
	flush := func() {
		if len(line) > 0 {
			b.WriteString("- _" + strings.ToUpper(string(current)) + ":_ " + strings.Join(line, " · ") + "\n")
			line = nil
		}
	}
	for _, n := range names {
		if n[0] != current {
			flush()
			current = n[0]
		}
		line = append(line, "[[cmd-"+n+"]]")
	}
	flush()

	b.WriteString("\n" + sidebarEnd)
	return b.String()
}
