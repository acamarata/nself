package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/ux"

	"github.com/nself-org/cli/cmd/commands"
)

// main is the only place in the tree allowed to call os.Exit. Commands signal a
// non-zero status by returning an error: errs.Exit / errs.ExitWith carry an
// explicit code, everything else exits 1. See internal/errs/exit_error.go.
//
// NOTE: internal/errs also documents a five-class contract (ExitCodeFor: 1 user,
// 2 infra, 3 auth, 4 destructive-blocked) that no caller has ever wired up, and
// which points at a .claude/docs/standards/exit-codes.md that does not exist.
// Switching main() to ExitCodeFor would silently change the status returned for
// license and docker failures, so it is deliberately left alone here and
// recorded as residue rather than changed as a side effect of CLI-R04.
func main() {
	err := commands.Execute()
	if err == nil {
		return
	}

	code := 1
	var coder errs.ExitCoder
	if errors.As(err, &coder) {
		code = coder.ExitCode()
	}

	// A silent error means the command (or a plugin subprocess sharing this
	// terminal) already wrote its output; printing again would duplicate it.
	var silencer errs.Silencer
	if errors.As(err, &silencer) && silencer.Silent() {
		os.Exit(code)
	}

	// Route structured UXErrors through the rich renderer.
	// Plain errors fall back to the simple "Error: ..." format.
	var uxErr *ux.UXError
	if errors.As(err, &uxErr) {
		uxErr.Print()
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	os.Exit(code)
}
