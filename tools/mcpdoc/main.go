// Command mcpdoc prints the current MCP tool/resource/prompt list as
// markdown, generated from source rather than hand-maintained.
//
// Purpose: CLI-R15 requires the wiki's tool list to be "generated from code
//
//	rather than hand-listed" so it can't silently drift the way the old
//	eleven-tool table did. This is a text scanner over the two files that
//	physically hold every mcp.NewTool/mcp.NewResource/mcp.NewPrompt call —
//	the same two files tools/parity/mcptools.go already hardcodes for its
//	own coverage check — plus mcp_resources.go and mcp_prompts.go for the
//	resource/prompt tables.
//
// Inputs:  cmd/commands/mcp.go, cmd/commands/mcp_sentry.go (tools),
//
//	cmd/commands/mcp_resources.go (resources), cmd/commands/mcp_prompts.go
//	(prompts).
//
// Outputs: markdown tables on stdout — paste into the relevant PROSE block
//
//	in .github/wiki/cmd-mcp.md by hand (wikigen owns the FLAGS block only;
//	PROSE blocks are for a human to fill in, generated or not).
//
// Constraints: read-only; writes nothing.
package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
)

var (
	resourceRe    = regexp.MustCompile(`mcp\.NewResource\(\s*"([a-zA-Z0-9_:/.]+)",\s*"([^"]*)"`)
	toolDescOnly  = regexp.MustCompile(`mcp\.NewTool\(\s*"([a-zA-Z0-9_]+)",\s*\n\s*mcp\.WithDescription\("((?:[^"\\]|\\.)*)"\)`)
	promptDescOnl = regexp.MustCompile(`mcp\.NewPrompt\(\s*"([a-zA-Z0-9_-]+)",\s*\n\s*mcp\.WithPromptDescription\("((?:[^"\\]|\\.)*)"\)`)
)

type entry struct{ name, desc string }

func extract(re *regexp.Regexp, paths ...string) []entry {
	var out []entry
	seen := map[string]bool{}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			if seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			out = append(out, entry{name: m[1], desc: m[2]})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func printTable(title string, entries []entry) {
	fmt.Printf("### %s\n\n| Name | Description |\n|------|-------------|\n", title)
	for _, e := range entries {
		fmt.Printf("| `%s` | %s |\n", e.name, e.desc)
	}
	fmt.Println()
}

func main() {
	tools := extract(toolDescOnly, "cmd/commands/mcp.go", "cmd/commands/mcp_sentry.go")
	resources := extract(resourceRe, "cmd/commands/mcp_resources.go")
	prompts := extract(promptDescOnl, "cmd/commands/mcp_prompts.go")

	printTable(fmt.Sprintf("Tools (%d)", len(tools)), tools)
	printTable(fmt.Sprintf("Resources (%d)", len(resources)), resources)
	printTable(fmt.Sprintf("Prompts (%d)", len(prompts)), prompts)
}
