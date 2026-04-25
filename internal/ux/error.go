// Package ux provides user-experience primitives for structured CLI error display.
// It wraps the errs.CLIError type with richer terminal rendering.
package ux

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/ui"
)

// UXError is the display layer for a CLIError.
// It holds a structured error and renders it with colored terminal output,
// including what/why/suggestions/docs-link/error-code sections.
type UXError struct {
	// Code is the short error code (e.g. "E001").
	Code string
	// What is a plain-language description of what happened.
	What string
	// Why is the root cause.
	Why string
	// Suggestions is a list of actionable fix steps shown as a numbered list.
	Suggestions []string
	// DocsLink is the full URL to the error reference page.
	DocsLink string
	// Wrapped is an optional underlying error for errors.Is/As.
	Wrapped error
}

// Error implements the error interface.
func (e *UXError) Error() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[%s] %s", e.Code, e.What))
	if e.Why != "" {
		b.WriteString(fmt.Sprintf(" (%s)", e.Why))
	}
	return b.String()
}

// Unwrap returns the wrapped error for errors.Is/As support.
func (e *UXError) Unwrap() error {
	return e.Wrapped
}

// Print renders the UXError to stderr with ANSI formatting.
func (e *UXError) Print() {
	fmt.Fprintf(os.Stderr, "\n%s  %s %s\n",
		ui.C(ui.Red, ui.IconFailure),
		ui.C(ui.Bold, e.Code+":"),
		e.What,
	)
	if e.Why != "" {
		fmt.Fprintf(os.Stderr, "%s  %s %s\n",
			ui.C(ui.Blue, ui.IconInfo),
			ui.C(ui.Dim, "Why:"),
			e.Why,
		)
	}
	if len(e.Suggestions) > 0 {
		fmt.Fprintf(os.Stderr, "\n%s  How to fix:\n", ui.C(ui.Cyan, ui.IconArrow))
		for i, s := range e.Suggestions {
			fmt.Fprintf(os.Stderr, "   %s  %s\n",
				ui.C(ui.Bold, fmt.Sprintf("%d.", i+1)),
				s,
			)
		}
	}
	if e.DocsLink != "" {
		fmt.Fprintf(os.Stderr, "\n%s  Docs: %s\n",
			ui.C(ui.Dim, ui.IconArrow),
			ui.C(ui.Underline, e.DocsLink),
		)
	}
	fmt.Fprintln(os.Stderr)
}

// FromCLIError converts an errs.CLIError to a UXError for rendering.
func FromCLIError(e *errs.CLIError) *UXError {
	ux := &UXError{
		Code:    e.Code,
		What:    e.What,
		Why:     e.Why,
		Wrapped: e.Wrapped,
	}
	if e.Fix != "" {
		ux.Suggestions = []string{e.Fix}
	}
	if e.DocsPath != "" {
		ux.DocsLink = "https://docs.nself.org/" + e.DocsPath
	}
	return ux
}

// New creates a UXError from the error registry. Falls back to a generic error
// when the code is not registered.
func New(code, what string) *UXError {
	cliErr := errs.New(code, what)
	return FromCLIError(cliErr)
}

// Newf creates a UXError with a formatted What message.
func Newf(code, format string, args ...interface{}) *UXError {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap creates a UXError that wraps an underlying error.
func Wrap(code, what string, err error) *UXError {
	ux := New(code, what)
	ux.Wrapped = err
	return ux
}

// WithSuggestions returns a copy of the error with additional suggestions.
func (e *UXError) WithSuggestions(suggestions ...string) *UXError {
	cp := *e
	cp.Suggestions = append(cp.Suggestions, suggestions...)
	return &cp
}

// WithWhy returns a copy of the error with a custom Why field.
func (e *UXError) WithWhy(why string) *UXError {
	cp := *e
	cp.Why = why
	return &cp
}

// PrintAndReturn prints the error to stderr and returns it as a plain error,
// so call sites can write: return ux.Newf("E001", "...").WithWhy("...").PrintAndReturn()
func (e *UXError) PrintAndReturn() error {
	e.Print()
	return e
}
