package commands

// Purpose: `nself functions delete` split out of functions.go (CLI-R12
// Batch B mechanical file-size split). Removes a deployed function's
// directory from ./functions/<name>/.
// Inputs: cobra command flags (--confirm) and the positional function name.
// Outputs: stdout confirmation on success; errors when --confirm is
// missing or the function directory does not exist.
// Constraints: pure move, no behavior change. functionNamePattern and
// functionsCmd (parent) remain in functions.go.

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var functionsDeleteFlags struct {
	confirm bool
}

var functionsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a deployed function",
	Long: `Remove a deployed function and its directory from ./functions/<name>/.

Requires --confirm to prevent accidental deletion.

Example:
  nself functions delete hello-world --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runFunctionsDelete,
}

func init() {
	functionsDeleteCmd.Flags().BoolVar(&functionsDeleteFlags.confirm, "confirm", false, "Confirm deletion (required)")
}

func runFunctionsDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !functionNamePattern.MatchString(name) {
		return fmt.Errorf("invalid function name %q", name)
	}

	if !functionsDeleteFlags.confirm {
		return fmt.Errorf("--confirm is required to delete a function")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	fnDir := filepath.Join(cwd, "functions", name)
	if _, err := os.Stat(fnDir); os.IsNotExist(err) {
		return fmt.Errorf("function %q not found at %s", name, fnDir)
	}

	if err := os.RemoveAll(fnDir); err != nil {
		return fmt.Errorf("deleting function directory: %w", err)
	}

	fmt.Printf("Function %q deleted.\n", name)
	return nil
}
