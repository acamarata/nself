package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/cmdlog"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/deprecation"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/observability"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/version"

	"github.com/spf13/cobra"
)

// deprecationRegistry is loaded once at startup from registry.yaml.
// A nil registry (load failure) is silently skipped — never crashes.
var deprecationRegistry *deprecation.Registry

// tracerShutdown holds the OTel tracer shutdown function set by PersistentPreRunE
// and called by PersistentPostRunE. It is nil when tracing is disabled (no
// OTEL_EXPORTER_OTLP_ENDPOINT). The package-level var survives across the
// PreRunE → RunE → PostRunE cobra lifecycle so spans are not flushed early.
var tracerShutdown func(context.Context) error

// contextKey is an unexported type for context keys in this package.
type contextKey int

// exitCodeKey stores a custom process exit code set by commands.
const exitCodeKey contextKey = 1

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "nself",
	Short: "nSelf CLI - Serverless hosting, anywhere",
	Long: `nSelf CLI empowers you to deploy a fully-featured, production-ready
backend to any hosting provider with absolute simplicity.

The Golden Path:
  nself init    # Generate your pristine .env configuration
  nself build   # Compose your infrastructure
  nself start   # Boot your stack`,
	// Enforcing strict bounds: RunE is used for graceful error bubbling
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	// Suppress cobra's automatic usage print and error print on errors.
	// Errors are printed exactly once by main() to stderr.
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Load the deprecation registry at startup (≤5ms; cached in memory).
	// The registry is compiled into the binary via go:embed, so it travels with
	// a single-file install. NSELF_DEPRECATION_REGISTRY overrides it with a file
	// for tests. A malformed registry is logged in debug mode only, never fatal.
	var regErr error
	deprecationRegistry, regErr = deprecation.LoadEmbeddedRegistry()
	if regErr != nil && os.Getenv("NSELF_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "debug: deprecation registry: %v\n", regErr)
	}

	// Add --version / -v flag to root command (legacy CLI compatibility)
	RootCmd.Flags().BoolP("version", "v", false, "Print version and exit")
	// --no-monorepo disables automatic monorepo backend detection globally.
	RootCmd.PersistentFlags().Bool("no-monorepo", false, "Disable automatic monorepo backend detection")
	// --no-deprecation-warnings suppresses deprecation output (for scripted use).
	RootCmd.PersistentFlags().Bool("no-deprecation-warnings", false, "Suppress deprecation warnings (for scripted use)")
	RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// ── OTel tracing ──────────────────────────────────────────────────────
		// InitTracer is only called when OTEL_EXPORTER_OTLP_ENDPOINT is set.
		// Leaving it unset is the zero-config path: no tracer, no side-effects.
		//
		// IMPORTANT: do NOT defer shutdown here. PersistentPreRunE returns before
		// RunE executes, so a defer inside this closure fires immediately after
		// PreRunE returns — before the command runs — dropping every span.
		// Instead store the shutdown func and call it in PersistentPostRunE, which
		// runs after RunE, guaranteeing spans are exported after the command body.
		if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
			tracerCfg := observability.TracerConfig{
				ServiceName: "nself-cli",
				Version:     version.GetVersion(),
			}
			if shutdown, err := observability.InitTracer(tracerCfg); err == nil {
				tracerShutdown = shutdown
			}
		}

		// ── Deprecation warning ───────────────────────────────────────────────
		// Emit before any other output so the warning is the first thing seen.
		// Written to stderr so piped stdout output is never polluted.
		if deprecationRegistry != nil {
			noWarn, _ := cmd.Flags().GetBool("no-deprecation-warnings")
			quiet, _ := cmd.Flags().GetBool("quiet")
			if !noWarn && !quiet {
				if item, ok := deprecationRegistry.Lookup(invokedCommandPath(cmd)); ok {
					deprecationRegistry.Warn(os.Stderr, item)
				}
			}
		}
		// ── Command execution log ─────────────────────────────────────────────
		// Write to ~/.nself/logs/ — a fixed, user-scoped path independent of
		// the working directory. Using cwd caused stray logs/ directories to
		// appear in arbitrary project folders (cross-project pollution).
		// Fallback to os.TempDir() if the home dir is unavailable.
		var logDir string
		if home, err := os.UserHomeDir(); err == nil {
			logDir = filepath.Join(home, ".nself", "logs")
		} else {
			logDir = filepath.Join(os.TempDir(), "nself", "logs")
		}
		finishLog := cmdlog.New(logDir).Begin(os.Args)
		defer func() { finishLog(0, nil) }()

		// `nself --version` prints and stops. Returning a silent exit-0 error
		// rather than calling os.Exit keeps the single-exit-point rule and lets
		// PersistentPostRunE flush any OTel spans first.
		v, _ := cmd.Flags().GetBool("version")
		if v {
			fmt.Println(version.GetVersion())
			return errs.Exit(0)
		}

		// Source directory guard — prevent running nself commands inside
		// the nself source repository. This avoids generating docker-compose,
		// nginx, ssl, .env, and other runtime artifacts inside the source tree.
		// Safe commands (help, version, completion) are whitelisted.
		if !isSourceSafeCommand(cmd.Name()) {
			if err := checkNotInSourceRepo(); err != nil {
				return err
			}
		}

		// ── Monorepo detection ────────────────────────────────────────────────
		// Run for all lifecycle commands so that stop, restart, logs, status,
		// build, and exec work correctly from a monorepo root — not just start.
		noMonorepo, _ := cmd.Flags().GetBool("no-monorepo")
		if !noMonorepo && !isSourceSafeCommand(cmd.Name()) {
			if cwd, err := os.Getwd(); err == nil {
				if backendRoot := config.DetectMonorepoRoot(cwd); backendRoot != "" {
					fmt.Printf("→ Detected monorepo layout. Using %s as project root.\n", filepath.Base(backendRoot))
					_ = os.Chdir(backendRoot)
				}
			}
		}

		// Migrate v1 license (~/.nself/license.json) to v2 location
		// (~/.config/nself/license.json) on first run after upgrade.
		if home, err := os.UserHomeDir(); err == nil {
			_ = license.MigrateLicenseFromV1(home)
		}

		return nil
	}

	// PersistentPostRunE runs AFTER RunE on every command, which is the correct
	// place to flush OTel spans. Cobra's defer-in-PreRunE anti-pattern fires the
	// shutdown before RunE executes, dropping all spans. Placing it here ensures
	// the tracer is shut down only after the command body has completed and had
	// a chance to record and export its spans.
	RootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if tracerShutdown != nil {
			// Use a background context so the flush is not cancelled if the
			// command's context has already expired.
			_ = tracerShutdown(context.Background())
			tracerShutdown = nil
		}
		return nil
	}
}

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
func warnRelocatedCommand(cmdName string) {
	if deprecationRegistry == nil {
		return
	}
	for _, a := range os.Args {
		if a == "--no-deprecation-warnings" || a == "--quiet" {
			return
		}
	}
	if item, ok := deprecationRegistry.Lookup("nself " + cmdName); ok {
		deprecationRegistry.Warn(os.Stderr, item)
	}
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
				warnRelocatedCommand(cmdName)

				// Proxy to plugin
				pluginArgs := []string{}
				if len(os.Args) > 2 {
					pluginArgs = os.Args[2:]
				}
				if err := plugin.ProxyCommand(cmdName, pluginArgs); err != nil {
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
