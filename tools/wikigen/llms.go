package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// writeLLMsTxt emits an AI-consumable single-file reference of the whole CLI
// surface, following the llms.txt convention.
//
// The wiki is 92 separate pages plus prose; an agent answering "how do I do X
// with nself" needs the whole surface in one fetch. This is that file: every
// command, its subcommands, and its flags, generated from cobra so it is never
// out of date with the binary.
func writeLLMsTxt(path string, cmds []*cobra.Command, check bool) (bool, error) {
	var b strings.Builder

	b.WriteString("# nSelf CLI\n\n")
	b.WriteString("> Self-hosted backend platform. One Go binary orchestrates Postgres, Hasura,\n")
	b.WriteString("> auth and nginx; everything beyond the core is a plugin.\n\n")
	b.WriteString("Generated from the cobra command tree — do not hand edit.\n")
	b.WriteString("Regenerate with `make wiki-commands`.\n\n")
	b.WriteString("The golden path is three commands:\n\n")
	b.WriteString("```bash\nnself init    # write .env\nnself build   # generate docker-compose + nginx\nnself start   # boot the stack\n```\n\n")
	fmt.Fprintf(&b, "## Commands (%d)\n\n", len(cmds))

	for _, c := range cmds {
		fmt.Fprintf(&b, "### nself %s\n\n", c.Name())
		fmt.Fprintf(&b, "%s\n\n", strings.TrimRight(oneLine(c), "."))
		fmt.Fprintf(&b, "```\n%s\n```\n\n", synopsis(c))

		if subs := visibleSubcommands(c); len(subs) > 0 {
			b.WriteString("Subcommands:\n\n")
			for _, s := range subs {
				fmt.Fprintf(&b, "- `%s` — %s\n", s.Name(), strings.TrimRight(oneLine(s), "."))
			}
			b.WriteString("\n")
		}

		var flags []string
		c.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden || f.Name == "help" {
				return
			}
			def := f.DefValue
			if def == "" {
				def = `""`
			}
			flags = append(flags, fmt.Sprintf("- `--%s` (default %s) — %s", f.Name, def, f.Usage))
		})
		if len(flags) > 0 {
			b.WriteString("Flags:\n\n")
			b.WriteString(strings.Join(flags, "\n"))
			b.WriteString("\n\n")
		}

		fmt.Fprintf(&b, "Full page: [[cmd-%s]]\n\n", c.Name())
	}

	out := b.String()
	current, err := os.ReadFile(path)
	if err == nil && string(current) == out {
		return false, nil
	}
	if check {
		return true, nil
	}
	return true, os.WriteFile(path, []byte(out), 0o644)
}
