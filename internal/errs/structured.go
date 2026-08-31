package errs

import (
	"fmt"
	"strings"
)

// CLIError is the structured error type for all user-facing CLI errors.
// Every error shown to users must use this type to ensure consistent formatting
// with actionable guidance.
type CLIError struct {
	// Code is the error code (e.g., "E001"). Used for documentation cross-reference.
	Code string

	// What describes what happened in plain language.
	What string

	// Why explains the root cause.
	Why string

	// Fix provides exact steps the user can take to resolve the issue.
	Fix string

	// DocsPath is the relative docs URL path (e.g., "reference/error-codes#e001").
	DocsPath string

	// Wrapped is an optional underlying error for programmatic inspection.
	Wrapped error
}

// Error implements the error interface with the structured 4-field format.
func (e *CLIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s", e.Code, e.What)
	if e.Why != "" {
		fmt.Fprintf(&b, "\n  Why: %s", e.Why)
	}
	if e.Fix != "" {
		fmt.Fprintf(&b, "\n  Fix: %s", e.Fix)
	}
	if e.DocsPath != "" {
		fmt.Fprintf(&b, "\n  Docs: https://nself.org/docs/%s", e.DocsPath)
	}
	return b.String()
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *CLIError) Unwrap() error {
	return e.Wrapped
}

// New creates a new CLIError from a registered error code.
// If the code is not registered, it falls back to a generic error with the given message.
func New(code, what string) *CLIError {
	entry, ok := Registry[code]
	if !ok {
		return &CLIError{
			Code: code,
			What: what,
		}
	}
	return &CLIError{
		Code:     code,
		What:     what,
		Why:      entry.DefaultWhy,
		Fix:      entry.DefaultFix,
		DocsPath: entry.DocsPath,
	}
}

// Newf creates a CLIError with a formatted What message.
func Newf(code, format string, args ...interface{}) *CLIError {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap creates a CLIError that wraps an underlying error.
func Wrap(code string, what string, err error) *CLIError {
	e := New(code, what)
	e.Wrapped = err
	return e
}

// WithWhy returns a copy of the error with a custom Why field.
func (e *CLIError) WithWhy(why string) *CLIError {
	copy := *e
	copy.Why = why
	return &copy
}

// WithFix returns a copy of the error with a custom Fix field.
func (e *CLIError) WithFix(fix string) *CLIError {
	copy := *e
	copy.Fix = fix
	return &copy
}
