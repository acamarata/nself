package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/cmdlog"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/deprecation"
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
	// Load deprecation registry at startup (≤5ms; cached in memory).
	// Resolve path relative to the executable so it works after `make install`.
	// Missing or malformed registry is logged in debug mode only — never crashes.
	var regErr error
	regPath := resolveRegistryPath()
	deprecationRegistry, regErr = deprecation.LoadRegistry(regPath)
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
				cmdPath := cmd.CommandPath() // e.g. "nself old-cmd"
				if item, ok := deprecationRegistry.Lookup(cmdPath); ok {
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

		v, _ := cmd.Flags().GetBool("version")
		if v {
			fmt.Println(version.GetVersion())
			os.Exit(0)
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() error {
	// Product-alias shim: 'nsentry <args>' (symlinked binary) ≡ 'nself sentry <args>'.
	normalizeInvokedBinary()

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
				// Proxy to plugin
				pluginArgs := []string{}
				if len(os.Args) > 2 {
					pluginArgs = os.Args[2:]
				}
				if err := plugin.ProxyCommand(cmdName, pluginArgs); err != nil {
					fmt.Fprintf(os.Stderr, "Plugin error: %v\n", err)
					os.Exit(1)
				}
				return nil
			}
		}
	}

	if err := RootCmd.Execute(); err != nil {
		return err
	}
	// Read custom exit code set by commands (e.g. status).
	if ctx := RootCmd.Context(); ctx != nil {
		if code, ok := ctx.Value(exitCodeKey).(int); ok && code != 0 {
			os.Exit(code)
		}
	}
	return nil
}

// checkNotInSourceRepo detects if the user is running nself from within the
// nself CLI source repository (or any Go CLI source tree with cmd/ + internal/).
// This prevents generating docker-compose.yml, nginx/, ssl/, .env, and other
// runtime artifacts inside the source directory — a common mistake.
func checkNotInSourceRepo() error {
	cwd, err := os.Getwd()
	if err != nil {
		return nil // can't determine — allow
	}

	// Allow override for testing
	if os.Getenv("NSELF_ALLOW_SOURCE_DIR") == "1" {
		return nil
	}

	// Detect Go CLI source repo: cmd/commands/ + internal/ + go.mod
	markers := []string{
		filepath.Join(cwd, "cmd", "commands"),
		filepath.Join(cwd, "internal"),
		filepath.Join(cwd, "go.mod"),
	}
	allExist := true
	for _, m := range markers {
		if _, err := os.Stat(m); err != nil {
			allExist = false
			break
		}
	}
	if !allExist {
		return nil
	}

	// Additional check: go.mod must contain "module nself"
	data, err := os.ReadFile(filepath.Join(cwd, "go.mod"))
	if err != nil {
		return nil
	}
	if len(data) < 20 {
		return nil
	}
	// Check first line for "module nself"
	firstLine := string(data[:min(len(data), 100)])
	if firstLine == "" {
		return nil
	}
	for _, sig := range []string{"module nself", "github.com/nself-org/cli/cmd", "github.com/nself-org/cli/internal"} {
		if contains(firstLine, sig) {
			return fmt.Errorf(`cannot run nself commands inside the CLI source repository

You are in: %s

This directory contains the nself CLI source code (cmd/, internal/, go.mod).
Running nself here would generate docker-compose.yml, nginx configs, SSL certs,
and other runtime artifacts inside the source tree.

To fix:
  1. cd to your actual project directory
  2. Run: nself init
  3. Then: nself start

To override (testing only): export NSELF_ALLOW_SOURCE_DIR=1`, cwd)
		}
	}

	return nil
}

// isSourceSafeCommand returns true for commands that are safe to run from
// anywhere, including the source repository.
func isSourceSafeCommand(name string) bool {
	safe := map[string]bool{
		"help": true, "version": true, "completion": true,
		"update": true, "upgrade": true, "doctor": true, "nself": true,
	}
	return safe[name]
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstr(s, substr))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// resolveRegistryPath returns the path to the deprecation registry YAML.
// It looks next to the executable first (installed binary), then falls back
// to a development-tree path relative to the working directory.
func resolveRegistryPath() string {
	// Option 1: next to the installed binary (production)
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "internal", "deprecation", "registry.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Option 2: relative to cwd (development / `make build` in repo root)
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "internal", "deprecation", "registry.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Fallback: let LoadRegistry return a "not found" error gracefully
	return filepath.Join("internal", "deprecation", "registry.yaml")
}
