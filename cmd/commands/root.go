package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"nself/internal/license"
	"nself/internal/plugin"
	"nself/internal/version"

	"github.com/spf13/cobra"
)

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
	// Add --version / -v flag to root command (legacy CLI compatibility)
	RootCmd.Flags().BoolP("version", "v", false, "Print version and exit")
	RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
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

		// Migrate v1 license (~/.nself/license.json) to v2 location
		// (~/.config/nself/license.json) on first run after upgrade.
		if home, err := os.UserHomeDir(); err == nil {
			_ = license.MigrateLicenseFromV1(home)
		}

		return nil
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() error {
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
	for _, sig := range []string{"module nself", "nself/cmd", "nself/internal"} {
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
		"update": true, "doctor": true, "nself": true,
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
