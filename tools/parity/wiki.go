// Purpose:     check whether a top-level command has a published wiki page.
// Inputs:      the wiki directory and a command name.
// Outputs:     bool — true when .github/wiki/cmd-<name>.md exists.
// Constraints: matches tools/wikigen's pageName convention (cmd-<name>.md,
//
//	pages live at the wiki root per its CLI-R08 flattening note) so
//	this column can never disagree with what wikigen itself expects.
package main

import (
	"os"
	"path/filepath"
)

// hasWikiPage reports whether the generated wiki page for cmd exists.
func hasWikiPage(dir, cmd string) bool {
	_, err := os.Stat(filepath.Join(dir, pageName(cmd)))
	return err == nil
}

// pageName mirrors tools/wikigen.pageName — duplicated rather than imported
// since wikigen is package main and cannot be imported; keep the two in sync
// if the naming convention ever changes.
func pageName(cmd string) string { return "cmd-" + cmd + ".md" }
