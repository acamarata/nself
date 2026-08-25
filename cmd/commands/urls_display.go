package commands

// Purpose: Route-conflict detection and terminal rendering for `nself urls`
// — finds routes that would resolve to the same subdomain, and prints the
// grouped URL listing. Split out of urls.go (CLI-R12) to separate display
// concerns from the cobra entry point (urls.go), the output builder
// (urls_build.go), and the diff subcommand (urls_diff.go).
// Inputs: a populated urlsOutput and a showAll flag.
// Outputs: a []conflict slice, and printed grouped URL output.
// Constraints: pure move — no behavior changes.

import (
	"fmt"

	"github.com/nself-org/cli/internal/ui"
)

// detectConflicts finds routes that resolve to the same subdomain.
func detectConflicts(out urlsOutput) []conflict {
	seen := make(map[string]string) // route -> first service name
	var conflicts []conflict

	allEntries := make([]serviceURL, 0, 32)
	allEntries = append(allEntries, out.RequiredServices...)
	allEntries = append(allEntries, out.OptionalServices...)
	allEntries = append(allEntries, out.CustomServices...)
	allEntries = append(allEntries, out.FrontendApps...)
	allEntries = append(allEntries, out.InternalRoutes...)

	for _, entry := range allEntries {
		if entry.Internal {
			continue
		}
		route := entry.URL
		if prev, ok := seen[route]; ok {
			conflicts = append(conflicts, conflict{
				Route:    route,
				Service1: prev,
				Service2: entry.Name,
			})
		} else {
			seen[route] = entry.Name
		}
	}

	return conflicts
}

// printURLGroups renders the grouped URL listing to the terminal.
func printURLGroups(out urlsOutput, showAll bool) {
	printGroup("Required Services", out.RequiredServices)
	printGroup("Optional Services", out.OptionalServices)
	printGroup("Custom Services", out.CustomServices)
	printGroup("Frontend Apps", out.FrontendApps)
	if showAll {
		printGroup("Internal Routes", out.InternalRoutes)
	}

	fmt.Printf("\n%d routes on %s\n", out.TotalRoutes, ui.C(ui.Cyan, out.BaseDomain))
}

// printGroup renders a single URL group section.
func printGroup(title string, entries []serviceURL) {
	if len(entries) == 0 {
		return
	}
	ui.Section(title)

	// Calculate max name width for alignment.
	maxName := 0
	for _, e := range entries {
		if len(e.Name) > maxName {
			maxName = len(e.Name)
		}
	}

	for _, e := range entries {
		name := fmt.Sprintf("%-*s", maxName, e.Name)
		if e.Internal {
			fmt.Printf("  %s   %s   %s\n",
				ui.C(ui.Dim, name),
				ui.C(ui.Dim, e.URL),
				ui.C(ui.Dim, "(internal only)"),
			)
		} else {
			fmt.Printf("  %s   %s\n",
				ui.C(ui.Cyan, name),
				ui.C(ui.White, e.URL),
			)
		}
	}
}
