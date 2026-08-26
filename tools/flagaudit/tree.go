// Purpose:     walk the real cobra command tree (the same registration the
//
//	shipped binary uses, via cmd/commands — no separate parser to
//	drift) and record, for every command path, the full set of
//	flags that would actually parse there.
//
// Inputs:      commands.RootCmd, after commands.ApplyCommandGroups().
// Outputs:     a map from command path ("nself ssl setup") to the flag names
//
//	valid on that exact invocation.
//
// Constraints: tools/cmdinventory's committed JSON is generated at -depth 2,
//
//	so it does not carry flags for deeper subcommands (e.g.
//	`nself ssl setup`). This walks the live tree to unlimited
//	depth instead of reusing that JSON. A flag registered with
//	PersistentFlags() on an ancestor is valid on every descendant
//	(cobra's own inheritance, exposed by InheritedFlags()); a flag
//	registered with Flags() only is local to that one node
//	(LocalFlags()) — both are folded into one set per path so a
//	flag is never wrongly flagged just because it lives on the
//	parent rather than the leaf, or vice versa.
package main

import (
	"github.com/nself-org/cli/cmd/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// commandNode is one resolvable point in the cobra tree: a full path plus
// every flag name that parses there, and the child names reachable from it.
type commandNode struct {
	Path     string          // e.g. "ssl setup" (without the leading "nself")
	Flags    map[string]bool // "--flag-name" -> true
	Children map[string]*commandNode
}

// buildTree walks commands.RootCmd and returns the root node plus a flat
// index by path, so callers can either walk step-by-step (to detect where a
// documented path stops matching the real tree) or look up a full path in
// one shot.
func buildTree() (*commandNode, map[string]*commandNode) {
	commands.ApplyCommandGroups()
	root := &commandNode{
		Path:     "",
		Flags:    flagSet(commands.RootCmd),
		Children: map[string]*commandNode{},
	}
	index := map[string]*commandNode{"": root}
	walk(commands.RootCmd, root, index)
	return root, index
}

// walk recursively populates children of node from the cobra command's
// subcommands, skipping the cobra-synthesised "help" command (not something
// anyone documents invoking directly, and it carries no flags of interest).
func walk(c *cobra.Command, node *commandNode, index map[string]*commandNode) {
	for _, child := range c.Commands() {
		if child.Name() == "help" {
			continue
		}
		childPath := child.Name()
		if node.Path != "" {
			childPath = node.Path + " " + child.Name()
		}
		childNode := &commandNode{
			Path:     childPath,
			Flags:    flagSet(child),
			Children: map[string]*commandNode{},
		}
		node.Children[child.Name()] = childNode
		// Cobra alias names resolve to the same command when a user types
		// them, so a documented invocation using an alias must also resolve.
		for _, alias := range child.Aliases {
			node.Children[alias] = childNode
		}
		index[childPath] = childNode
		walk(child, childNode, index)
	}
}

// flagSet returns every flag name that parses on this exact command
// invocation: its own local flags plus every ancestor's persistent flags,
// which cobra already resolves via InheritedFlags(). --help and --version
// are cobra/root built-ins, always accepted, and are added explicitly since
// InheritedFlags does not surface the auto-added help flag consistently
// across cobra versions.
func flagSet(c *cobra.Command) map[string]bool {
	out := map[string]bool{"help": true, "h": true}
	add := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			out[f.Name] = true
			if f.Shorthand != "" {
				out[f.Shorthand] = true
			}
		})
	}
	add(c.LocalFlags())
	add(c.InheritedFlags())
	return out
}
