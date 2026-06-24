package commands

// ci.go — `nself ci` top-level command group.
//
// Purpose: Register the `nself ci` command under RootCmd and attach the
// eval subcommand group from internal/cmd/ci/. All ci subcommand logic lives
// in internal/cmd/ci/ to keep this file thin (registration only).
// Inputs: cobra registration via init().
// Outputs: nself ci [subcommand] available in the CLI.
// Constraints: this file must stay ≤80 lines; no business logic here.
// SPORT: F02-COMMAND-INVENTORY.md — nself ci eval + nself ci eval gate entries.

import (
	"errors"

	internalci "github.com/nself-org/cli/internal/cmd/ci"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/spf13/cobra"
)

// ciCmd is the parent command for all nself CI operations.
var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "CI/CD integration commands",
	Long: `Commands for integrating nSelf eval gates into CI/CD pipelines.

Subcommands:
  eval      Run eval suites and write eval-results.json artifact
  eval gate Check autonomy-tier gate clearance`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// exitCoder is the interface used by internal/cmd/ci error types to expose
// a process exit code. main() checks plugin.ExitCodeError; we wrap ci errors
// into that type here so exit codes propagate correctly.
type exitCoder interface {
	ExitCode() int
}

func init() {
	// Attach the eval subcommand group (which in turn attaches eval gate).
	ciCmd.AddCommand(internalci.EvalCmd())

	// Wrap any exit-coded errors from ci subcommands into plugin.ExitCodeError
	// so main() can call os.Exit with the correct code (0/1/2/3 per spec §6).
	ciCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	// Apply the exit-code wrapping at the PostRun level by overriding the
	// error transformation on RunE completion. cobra propagates errors from
	// RunE upward; we intercept in a PersistentPreRunE wrapper on RootCmd —
	// instead, use the standard cobra error-wrapping contract: any error from
	// a ci subcommand that satisfies exitCoder is re-wrapped as ExitCodeError
	// before reaching main(). We do this via a persistent post-run hook that
	// re-checks for the interface and replaces the error in cobra's state.
	//
	// The simpler approach: ciCmd.RunE delegates wrap in each subcommand's RunE.
	// The error returned from RunE is intercepted by cobra and passed to main().
	// The *exitError in internal/cmd/ci already satisfies exitCoder; we only need
	// main() to check for it. We add a cobra wrapper here that converts exitCoder
	// errors to *plugin.ExitCodeError so the existing main() path handles them.
	origPREE := ciCmd.PersistentPreRunE
	ciCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if origPREE != nil {
			return origPREE(cmd, args)
		}
		return nil
	}

	// Register a command-level error transformer: cobra calls RunE and returns
	// the error to main(). If it satisfies exitCoder, wrap it.
	// We achieve this by overriding each subcommand's RunE via a middleware.
	// The cleanest pattern for this CLI: document that ci errors already implement
	// Error() string and are passed to main(). main() checks *plugin.ExitCodeError.
	// So we need internal/cmd/ci's exitError to convert to *plugin.ExitCodeError.
	//
	// Solution: add a PersistentPostRunE that is never called (cobra only calls it
	// on success), and instead wrap the subcommand RunE functions directly.
	wrapSubcmds(ciCmd)

	RootCmd.AddCommand(ciCmd)
}

// wrapSubcmds wraps each subcommand's RunE to convert exitCoder errors to
// *plugin.ExitCodeError so main() can extract the process exit code.
//
// Purpose: Bridge between internal/cmd/ci exit-code errors and the existing
// main() error-handling pattern that checks for *plugin.ExitCodeError.
// Inputs: parent *cobra.Command whose RunE and subcommand RunE to wrap.
// Outputs: mutates RunE in place; no-op if RunE already returns nil or error.
func wrapSubcmds(parent *cobra.Command) {
	wrapRunE(parent)
	for _, sub := range parent.Commands() {
		wrapSubcmds(sub) // recurse to handle eval gate
	}
}

// wrapRunE wraps cmd.RunE to convert exitCoder to *plugin.ExitCodeError.
func wrapRunE(cmd *cobra.Command) {
	if cmd.RunE == nil {
		return
	}
	orig := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		err := orig(c, args)
		if err == nil {
			return nil
		}
		var ec exitCoder
		if errors.As(err, &ec) {
			return &plugin.ExitCodeError{Code: ec.ExitCode()}
		}
		return err
	}
}
