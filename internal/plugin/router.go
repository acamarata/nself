package plugin

import (
	"fmt"
	"os"
	"os/exec"
)

// ProxyCommand checks if a plugin binary exists and executes it.
// If not, it instructs the user to install it.
func ProxyCommand(cmdName string, args []string) error {
	pluginBinary := fmt.Sprintf("nself-%s", cmdName)

	// Check if the binary is in PATH
	path, err := exec.LookPath(pluginBinary)
	if err != nil {
		fmt.Printf("Command '%s' is not a built-in command and the plugin '%s' was not found.\n", cmdName, pluginBinary)
		fmt.Printf("To install this plugin, run:\n  nself plugin install %s\n", cmdName)
		os.Exit(1)
	}

	// Prepare the command
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run process
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return fmt.Errorf("failed to execute plugin: %w", err)
	}

	return nil
}
