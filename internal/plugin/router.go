package plugin

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ExitCodeError is returned when a plugin process exits with a non-zero code.
// main() reads ExitCode() and mirrors the plugin's status; the message is not
// printed, because the plugin already wrote its own output to the terminal.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("plugin exited with code %d", e.Code)
}

// ExitCode satisfies the interface main() and errs.ExitCodeFor look for.
func (e *ExitCodeError) ExitCode() int { return e.Code }

// Silent reports that main() must not print this error: the plugin subprocess
// inherited stdout/stderr and has already reported the failure itself.
func (e *ExitCodeError) Silent() bool { return true }

// pluginBinDir returns the directory where plugin binaries are installed.
// This is ~/.nself/plugins/bin/ (or /tmp/.nself/plugins/bin/ as fallback).
func pluginBinDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "plugins", "bin")
	}
	return filepath.Join(home, ".nself", "plugins", "bin")
}

// ProxyCommand checks if a plugin binary exists and executes it.
// If not, it instructs the user to install it.
//
// For security, the binary is looked up ONLY in the plugin bin directory,
// never via the full system PATH. This prevents PATH hijacking attacks.
func ProxyCommand(cmdName string, args []string) error {
	pluginBinary := fmt.Sprintf("nself-%s", cmdName)

	// S-002: Only look in the plugin bin directory, never the full PATH.
	binDir := pluginBinDir()
	candidate := filepath.Join(binDir, pluginBinary)
	if runtime.GOOS == "windows" {
		candidate += ".exe"
	}

	path := candidate
	if _, err := os.Stat(path); err != nil {
		// CLI-R19: an unknown command is the moment a user most needs to be told
		// how to get it. The message is the actionable one — `nself install X` —
		// and it goes to the returned error, not only to a slog warning that a
		// normal terminal never shows.
		slog.Warn("plugin binary not found",
			"command", cmdName, "plugin", pluginBinary,
			"install_hint", "nself install "+cmdName)
		return fmt.Errorf("unknown command %q, and no plugin named %q is installed.\n\nIf this is an nSelf plugin, install it with:\n  nself install %s",
			cmdName, cmdName, cmdName)
	}

	// Prepare the command
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run process
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return &ExitCodeError{Code: exitError.ExitCode()}
		}
		return fmt.Errorf("failed to execute plugin: %w", err)
	}

	return nil
}
