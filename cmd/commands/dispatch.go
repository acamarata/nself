package commands

// Command dispatch: how an invocation is resolved before and after cobra.
//
// Purpose: keep root.go to defining the root command and its lifecycle hooks,
// and put the three things that decide WHICH command runs here — the spelling
// the user actually typed, the notice for a command that moved to a plugin,
// and Execute's routing of unknown commands to the plugin proxy.
//
// Inputs: os.Args and the cobra tree.
//
// Outputs: the executed command's error, or a routed plugin invocation.
//
// Constraints: the legacy-spelling rewrite must run before the plugin proxy —
// a retired name is no longer registered, so the proxy would otherwise try to
// resolve it as a plugin.

import (
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/plugin"

	"github.com/spf13/cobra"
)

// invokedCommandPath returns the command path as the user actually spelled it,
// which is what the deprecation registry is keyed on.
//
// cmd.CommandPath() alone is wrong for an aliased invocation: `nself up` resolves
// to startCmd, whose CommandPath is "nself start". Keying on that would either
// never warn about the old spelling (registry entry "nself up" unreachable) or
// warn on every correct `nself start` (entry "nself start" always matching).
// cobra records the spelling that selected the command in CalledAs(), so an
// alias gets its own registry key while the canonical name stays silent.
func invokedCommandPath(cmd *cobra.Command) string {
	called := cmd.CalledAs()
	if called == "" || called == cmd.Name() {
		return cmd.CommandPath()
	}
	prefix := "nself"
	if parent := cmd.Parent(); parent != nil {
		prefix = parent.CommandPath()
	}
	return prefix + " " + called
}

// warnRelocatedCommand emits the deprecation notice for a command that has
// moved out of core into a plugin.
//
// The registry is keyed on the spelling the user typed, same as
// invokedCommandPath, but this path runs before cobra: an extracted command is
// no longer registered, so RootCmd.Execute never reaches PersistentPreRunE for
// it. Without this the entry exists only as documentation and the user is told
// nothing about where the command went.
func warnRelocatedCommand(cmdName string) bool {
	if deprecationRegistry == nil {
		return false
	}
	for _, a := range os.Args {
		if a == "--no-deprecation-warnings" || a == "--quiet" {
			return false
		}
	}
	item, ok := deprecationRegistry.Lookup("nself " + cmdName)
	if !ok {
		return false
	}
	deprecationRegistry.Warn(os.Stderr, item)
	return true
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() error {
	// Group the command tree for help output. Done here, not in init(): every
	// command's own init() must have registered it on RootCmd first.
	ApplyCommandGroups()

	// Product-alias shim: 'nsentry <args>' (symlinked binary) ≡ 'nself sentry <args>'.
	normalizeInvokedBinary()

	// CLI-R09: rewrite retired top-level spellings onto their new home before
	// anything else looks at os.Args. This has to precede the plugin proxy
	// below: a retired name is no longer a registered command, so the proxy
	// would otherwise try to resolve it as a plugin.
	if legacy := rewriteLegacyInvocation(); legacy != "" {
		warnLegacySpelling(legacy)
	}

	// Route cobra error/usage output to stderr so structured output stays clean.
	RootCmd.SetErr(os.Stderr)

	// Intercept unknown commands for the plugin router
	if len(os.Args) > 1 {
		cmdName := os.Args[1]

		// Ignore global flags or root help
		if cmdName != "" && cmdName != "help" && cmdName[0] != '-' {
			// Check if the command is known to Cobra
			isKnown := false
			for _, c := range RootCmd.Commands() {
				if c.Name() == cmdName || c.HasAlias(cmdName) {
					isKnown = true
					break
				}
			}

			if !isKnown {
				// A command that left core for a plugin (CLI-R11) still has a
				// deprecation registry entry naming where it went. cobra never
				// sees the invocation — it is not a registered command — so
				// PersistentPreRunE's warning never runs and the entry would be
				// decorative. Emit it here, on the one path that does see it.
				// ...but ONLY when the plugin is absent: once the user has run
				// `nself install soak`, `nself soak` IS the supported spelling,
				// and telling them to install what they just installed is noise.
				// When the registry names the plugin to install, it is
				// authoritative — the plugin is not always named after the
				// command (`claw` lives in `claw-cli`). Suppress the proxy's
				// generic hint in that case so the user is not given two
				// different install commands one line apart.
				warned := false
				if !plugin.IsCommandInstalled(cmdName) {
					warned = warnRelocatedCommand(cmdName)
				}
				installHint := "nself install " + cmdName
				if warned {
					installHint = ""
				}

				// Proxy to plugin
				pluginArgs := []string{}
				if len(os.Args) > 2 {
					pluginArgs = os.Args[2:]
				}
				if err := plugin.ProxyCommandWithHint(cmdName, pluginArgs, installHint); err != nil {
					// The message is printed here to preserve the exact
					// "Plugin error: ..." wording; main() must not print it
					// again, hence the silent exit error.
					fmt.Fprintf(os.Stderr, "Plugin error: %v\n", err)
					return errs.Exit(1)
				}
				return nil
			}
		}
	}

	if err := RootCmd.Execute(); err != nil {
		return err
	}
	// Some commands (status, health) succeed but request a non-zero status to
	// report the state they found. Surface it as a silent exit error so main()
	// stays the only os.Exit caller.
	if ctx := RootCmd.Context(); ctx != nil {
		if code, ok := ctx.Value(exitCodeKey).(int); ok && code != 0 {
			return errs.Exit(code)
		}
	}
	return nil
}
