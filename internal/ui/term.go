package ui

import (
	"os"

	"golang.org/x/term"
)

// stdoutIsTerminalFunc is the function used to detect whether os.Stdout is
// connected to a terminal. It is a package-level variable so tests can
// override it to drive both the TTY and non-TTY code paths in the same
// process. Production code never reassigns this variable.
var stdoutIsTerminalFunc = func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// stdoutIsTerminal reports whether stdout is connected to a terminal.
// All UI helpers go through this seam so tests can reliably exercise both
// branches without allocating a real pseudo-terminal.
func stdoutIsTerminal() bool {
	return stdoutIsTerminalFunc()
}
