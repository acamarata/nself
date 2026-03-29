package confirm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrDestructionCanceled is returned when the user does not confirm destruction.
var ErrDestructionCanceled = errors.New("destruction canceled by user")

// ConfirmDestruction prompts the user to type projectName to confirm.
// It writes the prompt to w and reads a response line from r.
// Returns nil if the typed string matches projectName exactly.
// Returns ErrDestructionCanceled if the input does not match or if r is exhausted (EOF).
func ConfirmDestruction(projectName string, r io.Reader, w io.Writer) error {
	fmt.Fprintf(w, "\u26a0  This will permanently destroy project %q.\n", projectName)
	fmt.Fprintf(w, "   All containers, volumes, and data will be deleted.\n")
	fmt.Fprintf(w, "   Type the project name to confirm: ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		// EOF or read error — treat as canceled
		return ErrDestructionCanceled
	}

	input := strings.TrimSpace(scanner.Text())
	if input != projectName {
		return ErrDestructionCanceled
	}

	return nil
}
