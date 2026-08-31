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
	_, _ = fmt.Fprintf(w, "\u26a0  This will permanently destroy project %q.\n", projectName)
	_, _ = fmt.Fprintf(w, "   All containers, volumes, and data will be deleted.\n")
	_, _ = fmt.Fprintf(w, "   Type the project name to confirm: ")

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

// ConfirmHostWidePrune prompts the user to type "yes" before a command runs a
// host-wide `docker system prune`. Unlike ConfirmDestruction there is no
// project name to type against: the operation is not scoped to a project at
// all, it affects every Docker resource on the machine, so the prompt states
// that plainly instead. Returns nil only when the typed response is exactly
// "yes" (case-insensitive). Returns ErrDestructionCanceled on any other input,
// including EOF.
func ConfirmHostWidePrune(r io.Reader, w io.Writer) error {
	_, _ = fmt.Fprintln(w, "⚠  This will run a host-wide 'docker system prune'.")
	_, _ = fmt.Fprintln(w, "   It removes stopped containers, unused networks, unused images, and")
	_, _ = fmt.Fprintln(w, "   build cache for EVERY Docker project on this machine, not just the")
	_, _ = fmt.Fprintln(w, "   current one. Named volumes are not affected.")
	_, _ = fmt.Fprintf(w, "   Type \"yes\" to continue: ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		// EOF or read error — treat as canceled
		return ErrDestructionCanceled
	}

	input := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if input != "yes" {
		return ErrDestructionCanceled
	}

	return nil
}
