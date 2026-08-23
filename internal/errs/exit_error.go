package errs

import "fmt"

// ExitError carries an explicit process exit code out of a command.
//
// Purpose: let any RunE choose the process exit status without calling
// os.Exit, which the repo rule reserves for cmd/nself/main.go. Calling
// os.Exit inside a command skips deferred cleanup, skips the OTel span flush
// in PersistentPostRunE, and makes the path untestable because the test binary
// exits with it.
//
// Inputs: a canonical code from this package plus an optional wrapped error.
//
// Outputs: an error whose ExitCode() main() reads to set the exit status.
//
// Constraints: a nil Err means "exit with this code and print nothing", which
// is the correct shape when the command already wrote its own diagnostics.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

// Unwrap exposes the wrapped error to errors.Is / errors.As.
func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode reports the process exit status this error requests.
func (e *ExitError) ExitCode() int { return e.Code }

// Silent reports whether the error carries no message of its own, meaning the
// command already produced its output and main() must not print anything else.
func (e *ExitError) Silent() bool { return e.Err == nil }

// Exit returns an error that exits with code and prints nothing. Use it when
// the command has already written its own diagnostics to stderr.
func Exit(code int) error {
	return &ExitError{Code: code}
}

// ExitWith returns an error that exits with code and reports err through the
// normal top-level error rendering.
func ExitWith(code int, err error) error {
	if err == nil {
		return &ExitError{Code: code}
	}
	return &ExitError{Code: code, Err: err}
}

// ExitWithf is ExitWith over a formatted message.
func ExitWithf(code int, format string, args ...any) error {
	return &ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

// ExitCoder is implemented by errors that choose the process exit status.
// Both *ExitError and *plugin.ExitCodeError satisfy it, which lets main() and
// ExitCodeFor treat them uniformly without importing either package.
type ExitCoder interface{ ExitCode() int }

// Silencer is implemented by errors main() must not print, because the command
// (or a plugin subprocess sharing the terminal) already reported the failure.
type Silencer interface{ Silent() bool }
