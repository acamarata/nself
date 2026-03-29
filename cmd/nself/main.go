package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nself-org/cli/cmd/commands"
	"github.com/nself-org/cli/internal/plugin"
)

func main() {
	if err := commands.Execute(); err != nil {
		var exitErr *plugin.ExitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
