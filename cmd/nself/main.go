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
		var exitErr *plugin.ExitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
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
