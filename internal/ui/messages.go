package ui

import (
	"fmt"
	"os"
)

// Success prints a green checkmark and message to stdout.
func Success(msg string) {
	fmt.Printf("%s %s\n", C(Green, IconSuccess), msg)
}

// Error prints a red cross and message to stderr.
func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", C(Red, IconFailure), msg)
}

// Warn prints a yellow warning icon and message to stderr.
func Warn(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", C(Yellow, IconWarning), msg)
}

// Info prints a blue info icon and message to stdout.
func Info(msg string) {
	fmt.Printf("%s %s\n", C(Blue, IconInfo), msg)
}

// Section prints a newline, blue arrow, and bold title to stdout.
func Section(title string) {
	fmt.Printf("\n%s %s\n", C(Blue, IconArrow), C(Bold, title))
}

// Dimmed prints gray/dim text to stdout.
func Dimmed(msg string) {
	fmt.Printf("%s\n", C(Dim, msg))
}
