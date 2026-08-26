// Command flagaudit catches documented or scripted `nself ... --flag`
// invocations whose flag was never registered on the cobra command it names.
//
// Purpose:     PR #258 fixed `nself doctor --quick`: the flag was documented
//
//	in help_topics.go and relied on by scripts/golden-path.sh, but
//	was never added to the doctor command's flag set, so every
//	invocation died at the flag parser before RunE ran. That went
//	unnoticed for over a month because nothing checked docs and
//	scripts against the binary's actual flags. This tool is that
//	check, run as a CI gate (see .github/workflows/doc-sync.yml).
//
// Inputs:      .github/wiki/*.md and scripts/**/*.sh (override with -wiki
//
//	and -scripts), scanned for `nself <path> --flag` text; the
//	real cobra tree from cmd/commands.RootCmd for what those
//	flags actually resolve to.
//
// Outputs:     exit 1 and one line per undocumented-vs-unregistered flag,
//
//	naming the flag, the offending file:line, and the resolved
//	command, when any are found. Also prints, without failing,
//	the set of top-level command words used in the wiki/scripts
//	that the core binary does not register at all — these are
//	plugin-provided (region, alerts, dr, ... per CLI-R11) and
//	cannot be checked without the plugin installed.
//
// Constraints: read-only. Never execs the built binary — flags are resolved
//
//	by importing cmd/commands and walking the same cobra
//	registration `nself --help` would report, which also means
//	this works in CI without a build step.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// historicalWikiRe matches wiki pages that document a *past* CLI version by
// design: the changelog, and the "upgrading/migrating from vN" guides that
// exist specifically to show the old syntax a reader is moving away from.
// Their examples legitimately reference flags and commands the current
// binary no longer has — that isn't drift, it's the point of the page — so
// scanning them for current-flag drift would be checking the wrong thing.
var historicalWikiRe = regexp.MustCompile(`(?i)^(Changelog|Upgrad(e|ing)-.*v[0-9][0-9.]*|Guide-Migration-from-v[0-9][0-9.]*)\.md$`)

func main() {
	wikiDir := flag.String("wiki", ".github/wiki", "wiki directory to scan for nself invocations")
	scriptsDir := flag.String("scripts", "scripts", "scripts directory to scan for nself invocations")
	flag.Parse()

	_, index := buildTree()

	files, err := collectFiles(*wikiDir, ".md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "flagaudit: %v\n", err)
		os.Exit(1)
	}
	files = filterHistorical(files)
	scriptFiles, err := collectFiles(*scriptsDir, ".sh")
	if err != nil {
		fmt.Fprintf(os.Stderr, "flagaudit: %v\n", err)
		os.Exit(1)
	}
	files = append(files, scriptFiles...)
	sort.Strings(files)

	var invs []invocation
	for _, f := range files {
		found, err := scanFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "flagaudit: read %s: %v\n", f, err)
			os.Exit(1)
		}
		invs = append(invs, found...)
	}

	root := index[""]
	result := audit(root, invs)

	if len(result.Skips) > 0 {
		fmt.Println("Skipped (plugin-provided, not in the core binary):")
		seen := map[string]bool{}
		for _, s := range result.Skips {
			key := s.Command
			if seen[key] {
				continue
			}
			seen[key] = true
			fmt.Printf("  nself %s — first seen %s:%d\n", s.Command, s.File, s.Line)
		}
		fmt.Println()
	}

	if len(result.Findings) == 0 {
		fmt.Printf("flag-drift audit passed: %d invocation(s) checked across %d file(s), 0 unregistered flags.\n", len(invs), len(files))
		return
	}

	fmt.Fprintf(os.Stderr, "::error::%d unregistered flag(s) found in docs/scripts:\n", len(result.Findings))
	for _, f := range result.Findings {
		fmt.Fprintf(os.Stderr, "  %s:%d: --%s is not registered on `%s` (from: %s)\n", f.File, f.Line, f.Flag, f.Command, f.Raw)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Either register the flag on the command it documents, or fix the doc/script to match what the command actually accepts.")
	os.Exit(1)
}

// filterHistorical drops wiki pages matched by historicalWikiRe.
func filterHistorical(files []string) []string {
	out := files[:0]
	for _, f := range files {
		if historicalWikiRe.MatchString(filepath.Base(f)) {
			continue
		}
		out = append(out, f)
	}
	return out
}

// collectFiles returns every file under dir with the given extension. A
// missing dir is not an error — scripts/ and .github/wiki/ both exist in
// this repo, but the flag defaults should stay harmless if pointed
// elsewhere.
func collectFiles(dir, ext string) ([]string, error) {
	var out []string
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return out, nil
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ext {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
