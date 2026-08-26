// Purpose:     resolve each scanned invocation's command path against the
//
//	real cobra tree and check its flags against what that exact
//	node accepts.
//
// Inputs:      the tree built by buildTree, plus invocations from scanFile.
// Outputs:     one failure per flag the binary would reject, one skip per
//
//	command path that isn't part of the core binary at all
//	(plugin-provided — must never fail the gate).
//
// Constraints: an unresolved path token is treated as plugin-provided (skip,
//
//	never fail) whenever it occurs under a node that itself has
//	subcommands — a "router". `region`/`alerts`/`dr` fail to
//	resolve directly under root, which is exactly this case, but
//	so does `nself service templates ...` or `nself plugin cdn
//	...`: `service` and `plugin` both have known children, and
//	`templates`/`cdn` are not among them, so this may be a typo'd
//	command path or a subcommand a plugin registers at runtime
//	under that group — either way it cannot be resolved from the
//	core binary alone, and checking flags against the router
//	itself produced exactly the false positive this package's
//	docs warned about (`--wildcard` living on `ssl setup`, not
//	`ssl`, in reverse: a word that isn't a real child must not
//	silently fall back to its parent's flag set). An unresolved
//	token under a *leaf* node (no children at all, e.g. `init`)
//	is instead treated as a positional argument — `nself init
//	myproject --env prod` — and flags are checked against that
//	leaf, since a leaf can never have a plugin-registered
//	subcommand hiding under it.
package main

// alwaysIgnoredFlags are accepted everywhere regardless of what the target
// node registers. --help/-h is cobra's automatic flag; --version is only
// declared on the root command's local flags but is meaningful written
// after any subcommand in casual docs, so both are exempted rather than
// resolved per-node.
var alwaysIgnoredFlags = map[string]bool{
	"help":    true,
	"h":       true,
	"version": true,
	"v":       true,
}

// finding is one flag that the resolved command node does not register.
type finding struct {
	File    string
	Line    int
	Command string // resolved command path, e.g. "ssl setup"
	Flag    string
	Raw     string
}

// skip is one invocation whose top-level command isn't in the core binary.
type skip struct {
	File    string
	Line    int
	Command string
	Raw     string
}

// auditResult is the outcome of checking every invocation.
type auditResult struct {
	Findings []finding
	Skips    []skip
}

// audit resolves every invocation against the tree and classifies it.
func audit(root *commandNode, invs []invocation) auditResult {
	var res auditResult
	for _, inv := range invs {
		res.add(root, inv)
	}
	return res
}

func (r *auditResult) add(root *commandNode, inv invocation) {
	if len(inv.PathTokens) == 0 {
		// A bare `nself --flag` with no recognisable subcommand word at all.
		// Real invocations of this binary essentially always name a
		// subcommand first; a flag-only match here is far more often the
		// scanner catching "nself" used as someone else's argument value
		// (e.g. `psql -d nself -tAc "..."` in a pentest script) than a
		// genuine root-level flag example. Drop it rather than risk a false
		// failure on noise scan.go could not fully rule out up front.
		return
	}

	cur := root
	for i, tok := range inv.PathTokens {
		child, ok := cur.Children[tok]
		if ok {
			cur = child
			continue
		}
		if len(cur.Children) > 0 {
			// tok doesn't resolve under a router (a node with its own known
			// subcommands) — plugin-provided or a mis-typed path either
			// way, cannot be checked from the core binary. Skip, never fail.
			label := tok
			if i > 0 {
				label = cur.Path + " " + tok
			}
			r.Skips = append(r.Skips, skip{
				File:    inv.File,
				Line:    inv.Line,
				Command: label,
				Raw:     inv.Raw,
			})
			return
		}
		// cur is a leaf (no subcommands registered at all): tok is a
		// positional argument, e.g. the project name in `nself init
		// myproject --env prod`. Stop descending; check flags against cur.
		break
	}
	r.checkFlags(cur, inv)
}

func (r *auditResult) checkFlags(node *commandNode, inv invocation) {
	for _, flag := range inv.Flags {
		if alwaysIgnoredFlags[flag] {
			continue
		}
		if node.Flags[flag] {
			continue
		}
		cmdLabel := "nself " + node.Path
		if node.Path == "" {
			cmdLabel = "nself"
		}
		r.Findings = append(r.Findings, finding{
			File:    inv.File,
			Line:    inv.Line,
			Command: cmdLabel,
			Flag:    flag,
			Raw:     inv.Raw,
		})
	}
}
