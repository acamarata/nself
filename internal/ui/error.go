package ui

import (
	"fmt"
	"os"
)

// UXError prints a structured error to stderr with problem, context, and solutions.
func UXError(problem, context string, solutions []string) {
	fmt.Fprintf(os.Stderr, "\n%s %s\n", C(Red, IconFailure+" Problem:"), problem)
	if context != "" {
		fmt.Fprintf(os.Stderr, "%s %s\n", C(Blue, IconInfo+" Context:"), context)
	}
	fmt.Fprintf(os.Stderr, "\n%s Possible solutions:\n", C(Blue, IconArrow))
	for i, sol := range solutions {
		fmt.Fprintf(os.Stderr, "  %s %s\n", C(Bold, fmt.Sprintf("%d.", i+1)), sol)
	}
	fmt.Fprintf(os.Stderr, "\n%s Run 'nself doctor' for more diagnostics\n\n", C(Blue, IconInfo))
}
