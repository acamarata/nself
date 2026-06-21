package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nself-org/cli/cmd/commands"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ux"
)

func main() {
	if err := commands.Execute(); err != nil {
		// Plugin-managed exit codes take highest priority.
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

		// Use the canonical exit-code contract: 0=ok, 1=user-error,
		// 2=infra-error, 3=auth-error, 4=destructive-blocked.
		// errs.ExitCodeFor never returns 0 for a non-nil error.
		os.Exit(errs.ExitCodeFor(err))
	}
}
