package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nself-org/cli/cmd/commands"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ux"
)

func main() {
	if err := commands.Execute(); err != nil {
		// Check for plugin ExitCodeError (from plugin subprocess)
		var pluginExitErr *plugin.ExitCodeError
		if errors.As(err, &pluginExitErr) {
			os.Exit(pluginExitErr.Code)
		}

		// Check for commands ExitError (from CLI gates and eval)
		var cmdExitErr *commands.ExitError
		if errors.As(err, &cmdExitErr) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", cmdExitErr)
			os.Exit(cmdExitErr.Code)
		}

		// Route structured UXErrors through the rich renderer.
		// Plain errors fall back to the simple "Error: ..." format.
		var uxErr *ux.UXError
		if errors.As(err, &uxErr) {
			uxErr.Print()
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
