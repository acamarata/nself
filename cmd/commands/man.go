package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var manOutDir string

var manCmd = &cobra.Command{
	Use:   "man",
	Short: "Generate man pages for all nself commands",
	Long: `Generate man pages for all nself commands and write them to a directory.

By default, pages are written to ./man/. Pass --output to override.

Examples:
  nself man                        # write to ./man/
  nself man --output /usr/share/man/man1
  sudo cp man/*.1 /usr/local/share/man/man1/ && mandb`,
	RunE: runManGen,
}

func init() {
	manCmd.Flags().StringVarP(&manOutDir, "output", "o", "man", "Directory to write man pages into")
	RootCmd.AddCommand(manCmd)
}

func runManGen(cmd *cobra.Command, _ []string) error {
	if err := os.MkdirAll(manOutDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %q: %w", manOutDir, err)
	}

	count := 0
	for _, c := range collectLeafCommands(RootCmd) {
		if err := writeManPage(manOutDir, c); err != nil {
			return err
		}
		count++
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Generated %d man page(s) in %s\n", count, manOutDir)
	return nil
}

// collectLeafCommands returns every command (parent or leaf) in the tree,
// excluding the hidden root itself, so each gets its own man page.
func collectLeafCommands(root *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Hidden {
			return
		}
		if c != root {
			out = append(out, c)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return out
}

// commandPath returns the dot-joined command path (e.g. "nself-plugin-install").
func commandPath(c *cobra.Command) string {
	parts := []string{}
	cur := c
	for cur != nil {
		if cur.Use != "" {
			// Use contains "<args>" sometimes — take only the first word.
			parts = append([]string{strings.Fields(cur.Use)[0]}, parts...)
		}
		cur = cur.Parent()
	}
	return strings.Join(parts, "-")
}

// writeManPage writes a minimal roff man page for the given command.
func writeManPage(dir string, c *cobra.Command) error {
	name := commandPath(c)
	path := filepath.Join(dir, name+".1")

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	now := time.Now()
	short := c.Short
	if short == "" {
		short = c.Use
	}
	longDesc := c.Long
	if longDesc == "" {
		longDesc = short
	}

	// Write a minimal groff/troff man page (section 1 — user commands).
	fmt.Fprintf(f, ".TH %s 1 \"%s\" \"nself\" \"nself Manual\"\n", strings.ToUpper(name), now.Format("January 2006"))
	fmt.Fprintf(f, ".SH NAME\n%s \\- %s\n", name, roffEscape(short))
	fmt.Fprintf(f, ".SH SYNOPSIS\n.B %s\n", name)

	// Usage line (strip angle-bracket args for brevity).
	usageLine := c.UseLine()
	if usageLine != "" {
		fmt.Fprintf(f, "[%s]\n", roffEscape(usageLine))
	}

	fmt.Fprintf(f, ".SH DESCRIPTION\n%s\n", roffEscape(longDesc))

	// Flags section.
	flagUsages := c.Flags().FlagUsages()
	if flagUsages != "" {
		fmt.Fprintf(f, ".SH OPTIONS\n.nf\n%s\n.fi\n", roffEscape(flagUsages))
	}

	// See also: parent command.
	if p := c.Parent(); p != nil && p != RootCmd {
		parentName := commandPath(p)
		fmt.Fprintf(f, ".SH SEE ALSO\n.BR %s (1)\n", parentName)
	} else if c != RootCmd {
		fmt.Fprintf(f, ".SH SEE ALSO\n.BR nself (1)\n")
	}

	return nil
}

// roffEscape escapes characters that have special meaning in roff.
func roffEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}
