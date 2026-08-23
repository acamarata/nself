// Command cmdinventory prints the CLI's real command tree.
//
// Purpose:     make SPORT F02, the wiki, and the parity matrix generated from
//
//	the binary rather than hand-maintained. Every previous count in
//	the docs was typed by a human and had drifted (F02 and the PRI
//	both claimed 84 when the binary registered more).
//
// Inputs:      -format json|markdown|names, -depth N (0 = top level only).
// Outputs:     the command tree on stdout.
// Constraints: imports cmd/commands so the source of truth is the same cobra
//
//	registration the shipped binary uses; no separate parser to
//	drift. Read-only — never mutates the tree.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/nself-org/cli/cmd/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Command is one node of the inventory.
type Command struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Short       string    `json:"short"`
	Hidden      bool      `json:"hidden"`
	Deprecated  string    `json:"deprecated,omitempty"`
	GroupID     string    `json:"group_id,omitempty"`
	Aliases     []string  `json:"aliases,omitempty"`
	Flags       []string  `json:"flags,omitempty"`
	Subcommands []Command `json:"subcommands,omitempty"`
}

func main() {
	format := flag.String("format", "json", "output format: json, markdown, names")
	depth := flag.Int("depth", 1, "how many levels below root to include (1 = top level only)")
	includeHidden := flag.Bool("hidden", false, "include hidden commands")
	flag.Parse()

	commands.ApplyCommandGroups()
	root := commands.RootCmd
	tree := collect(root, *depth, *includeHidden)

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(tree); err != nil {
			fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			os.Exit(1)
		}
	case "names":
		for _, c := range tree {
			fmt.Println(c.Name)
		}
	case "markdown":
		printMarkdown(tree)
	default:
		fmt.Fprintf(os.Stderr, "unknown -format %q\n", *format)
		os.Exit(2)
	}
}

// collect walks the cobra tree to the requested depth.
func collect(parent *cobra.Command, depth int, includeHidden bool) []Command {
	if depth <= 0 {
		return nil
	}
	var out []Command
	for _, c := range parent.Commands() {
		// `help` is synthesised by cobra, not registered by us.
		if c.Name() == "help" {
			continue
		}
		if c.Hidden && !includeHidden {
			continue
		}
		out = append(out, Command{
			Name:        c.Name(),
			Path:        c.CommandPath(),
			Short:       c.Short,
			Hidden:      c.Hidden,
			Deprecated:  c.Deprecated,
			GroupID:     c.GroupID,
			Aliases:     c.Aliases,
			Flags:       localFlagNames(c),
			Subcommands: collect(c, depth-1, includeHidden),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// localFlagNames lists the flags declared on c itself, excluding inherited ones.
func localFlagNames(c *cobra.Command) []string {
	var names []string
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		names = append(names, "--"+f.Name)
	})
	sort.Strings(names)
	return names
}

func printMarkdown(tree []Command) {
	fmt.Println("| Command | Short Description | Group | Subcommands |")
	fmt.Println("|---|---|---|---|")
	for _, c := range tree {
		subs := make([]string, 0, len(c.Subcommands))
		for _, s := range c.Subcommands {
			subs = append(subs, s.Name)
		}
		group := c.GroupID
		if group == "" {
			group = "—"
		}
		sub := strings.Join(subs, ", ")
		if sub == "" {
			sub = "—"
		}
		fmt.Printf("| `%s` | %s | %s | %s |\n", c.Path, escapePipes(c.Short), group, sub)
	}
	fmt.Printf("\nTotal top-level commands: %d\n", len(tree))
}

func escapePipes(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
